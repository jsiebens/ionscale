package handlers

import (
	"fmt"
	"github.com/jsiebens/ionscale/internal/bind"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/labstack/echo/v4"
	"net/http"
	"tailscale.com/tailcfg"
	"time"
)

func NewSSHActionHandlers(createBinder bind.Factory, config *config.Config, repository domain.Repository) *SSHActionHandlers {
	return &SSHActionHandlers{
		createBinder: createBinder,
		repository:   repository,
		config:       config,
	}
}

type SSHActionHandlers struct {
	createBinder bind.Factory
	repository   domain.Repository
	config       *config.Config
}

type sshActionRequestData struct {
	SrcMachineID uint64 `param:"src_machine_id"`
	DstMachineID uint64 `param:"dst_machine_id"`
	CheckPeriod  string `param:"check_period"`
}

func (h *SSHActionHandlers) StartAuth(c echo.Context) error {
	ctx := c.Request().Context()

	binder, err := h.createBinder(c)
	if err != nil {
		return logError(err)
	}

	data := new(sshActionRequestData)
	if err = c.Bind(data); err != nil {
		return logError(err)
	}

	if data.CheckPeriod != "" && data.CheckPeriod != "always" {
		checkPeriod, err := time.ParseDuration(data.CheckPeriod)
		if err != nil {
			_ = logError(err)
			goto check
		}

		machine, err := h.repository.GetMachine(ctx, data.SrcMachineID)
		if err != nil {
			return logError(err)
		}

		if machine.User.Account != nil && machine.User.Account.LastAuthenticated != nil {
			sinceLastAuthentication := time.Since(*machine.User.Account.LastAuthenticated)

			if sinceLastAuthentication < checkPeriod {
				resp := &tailcfg.SSHAction{
					Accept:                   true,
					AllowAgentForwarding:     true,
					AllowLocalPortForwarding: true,
				}

				return binder.WriteResponse(c, http.StatusOK, resp)
			}
		}
	}

check:
	key := util.RandStringBytes(8)
	request := &domain.SSHActionRequest{
		Key:          key,
		SrcMachineID: data.SrcMachineID,
		DstMachineID: data.DstMachineID,
		CreatedAt:    time.Now().UTC(),
	}

	authUrl := h.config.CreateUrl("/a/s/%s", key)

	if err := h.repository.SaveSSHActionRequest(ctx, request); err != nil {
		return logError(err)
	}

	resp := &tailcfg.SSHAction{
		Message:         fmt.Sprintf("# Tailscale SSH requires an additional check.\n# To authenticate, visit: %s\n", authUrl),
		HoldAndDelegate: fmt.Sprintf("https://unused/machine/ssh/action/check/%s", key),
	}

	return binder.WriteResponse(c, http.StatusOK, resp)
}

func (h *SSHActionHandlers) CheckAuth(c echo.Context) error {
	// Listen to connection close
	ctx := c.Request().Context()
	notify := ctx.Done()

	binder, err := h.createBinder(c)
	if err != nil {
		return logError(err)
	}

	tick := time.NewTicker(2 * time.Second)

	defer func() { tick.Stop() }()

	key := c.Param("key")

	for {
		select {
		case <-tick.C:
			m, err := h.repository.GetSSHActionRequest(ctx, key)

			if err != nil || m == nil {
				return binder.WriteResponse(c, http.StatusOK, &tailcfg.SSHAction{Reject: true})
			}

			if m.Action == "accept" {
				action := &tailcfg.SSHAction{
					Accept:                   true,
					AllowAgentForwarding:     true,
					AllowLocalPortForwarding: true,
				}
				_ = h.repository.DeleteSSHActionRequest(ctx, key)
				return binder.WriteResponse(c, http.StatusOK, action)
			}

			if m.Action == "reject" {
				action := &tailcfg.SSHAction{Reject: true}
				_ = h.repository.DeleteSSHActionRequest(ctx, key)
				return binder.WriteResponse(c, http.StatusOK, action)
			}
		case <-notify:
			return nil
		}
	}
}
