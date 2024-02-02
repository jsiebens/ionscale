package handlers

import (
	"context"
	"github.com/jsiebens/ionscale/internal/addr"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/core"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/mapping"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/netip"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
	"tailscale.com/util/dnsname"
	"time"
)

func NewRegistrationHandlers(
	machineKey key.MachinePublic,
	config *config.Config,
	sessionManager core.PollMapSessionManager,
	repository domain.Repository) *RegistrationHandlers {
	return &RegistrationHandlers{
		machineKey:     machineKey,
		sessionManager: sessionManager,
		repository:     repository,
		config:         config,
	}
}

type RegistrationHandlers struct {
	machineKey     key.MachinePublic
	repository     domain.Repository
	sessionManager core.PollMapSessionManager
	config         *config.Config
}

func (h *RegistrationHandlers) Register(c echo.Context) error {
	ctx := c.Request().Context()

	req := &tailcfg.RegisterRequest{}
	if err := c.Bind(req); err != nil {
		return logError(err)
	}

	if req.Version < SupportedCapabilityVersion {
		response := tailcfg.RegisterResponse{MachineAuthorized: false, Error: UnsupportedClientVersionMessage}
		return c.JSON(http.StatusOK, response)
	}

	machineKey := h.machineKey.String()
	nodeKey := req.NodeKey.String()

	var m *domain.Machine
	m, err := h.repository.GetMachineByKeys(ctx, machineKey, nodeKey)

	if err != nil {
		return logError(err)
	}

	if m != nil {
		if m.IsExpired() {
			response := tailcfg.RegisterResponse{NodeKeyExpired: true}
			return c.JSON(http.StatusOK, response)
		}

		if !req.Expiry.IsZero() && req.Expiry.Before(time.Now()) {
			m.ExpiresAt = req.Expiry

			if m.Ephemeral {
				if _, err := h.repository.DeleteMachine(ctx, m.ID); err != nil {
					return logError(err)
				}
				h.sessionManager.NotifyAll(m.TailnetID)
			} else {
				if err := h.repository.SaveMachine(ctx, m); err != nil {
					return logError(err)
				}
				h.sessionManager.NotifyAll(m.TailnetID)
			}

			response := tailcfg.RegisterResponse{NodeKeyExpired: true}
			return c.JSON(http.StatusOK, response)
		}

		sanitizeHostname := dnsname.SanitizeHostname(req.Hostinfo.Hostname)
		if m.Name != sanitizeHostname {
			nameIdx, err := h.repository.GetNextMachineNameIndex(ctx, m.TailnetID, sanitizeHostname)
			if err != nil {
				return logError(err)
			}
			m.Name = sanitizeHostname
			m.NameIdx = nameIdx

		}

		advertisedTags := domain.SanitizeTags(req.Hostinfo.RequestTags)
		m.Tags = append(m.RegisteredTags, advertisedTags...)

		if err := h.repository.SaveMachine(ctx, m); err != nil {
			return logError(err)
		}

		tUser, tLogin := mapping.ToUser(m.User)

		response := tailcfg.RegisterResponse{
			MachineAuthorized: m.Authorized,
			User:              tUser,
			Login:             tLogin,
		}

		return c.JSON(http.StatusOK, response)
	}

	return h.authenticateMachine(c, machineKey, req)
}

func (h *RegistrationHandlers) authenticateMachine(c echo.Context, machineKey string, req *tailcfg.RegisterRequest) error {
	ctx := c.Request().Context()

	if req.Followup != "" {
		return h.followup(c, req)
	}

	if req.Auth.AuthKey == "" {
		key := util.RandStringBytes(8)
		authUrl := h.config.CreateUrl("/a/r/%s", key)

		request := domain.RegistrationRequest{
			MachineKey: machineKey,
			Key:        key,
			CreatedAt:  time.Now().UTC(),
			Data:       domain.RegistrationRequestData(*req),
		}

		err := h.repository.SaveRegistrationRequest(ctx, &request)
		if err != nil {
			response := tailcfg.RegisterResponse{MachineAuthorized: false, Error: "something went wrong"}
			return c.JSON(http.StatusOK, response)
		}

		response := tailcfg.RegisterResponse{AuthURL: authUrl}
		return c.JSON(http.StatusOK, response)
	} else {
		return h.authenticateMachineWithAuthKey(c, machineKey, req)
	}
}

func (h *RegistrationHandlers) authenticateMachineWithAuthKey(c echo.Context, machineKey string, req *tailcfg.RegisterRequest) error {
	ctx := c.Request().Context()
	nodeKey := req.NodeKey.String()

	authKey, err := h.repository.LoadAuthKey(ctx, req.Auth.AuthKey)
	if err != nil {
		return logError(err)
	}

	if authKey == nil {
		response := tailcfg.RegisterResponse{MachineAuthorized: false, Error: "invalid auth key"}
		return c.JSON(http.StatusOK, response)
	}

	tailnet := authKey.Tailnet
	user := authKey.User

	if err := tailnet.ACLPolicy.Get().CheckTagOwners(req.Hostinfo.RequestTags, &user); err != nil {
		response := tailcfg.RegisterResponse{MachineAuthorized: false, Error: err.Error()}
		return c.JSON(http.StatusOK, response)
	}

	registeredTags := authKey.Tags
	advertisedTags := domain.SanitizeTags(req.Hostinfo.RequestTags)
	tags := append(registeredTags, advertisedTags...)

	autoAllowIPs := tailnet.ACLPolicy.Get().FindAutoApprovedIPs(req.Hostinfo.RoutableIPs, tags, &user)

	var m *domain.Machine

	m, err = h.repository.GetMachineByKeyAndUser(ctx, machineKey, user.ID)
	if err != nil {
		return logError(err)
	}

	now := time.Now().UTC()

	if m == nil {
		sanitizeHostname := dnsname.SanitizeHostname(req.Hostinfo.Hostname)
		nameIdx, err := h.repository.GetNextMachineNameIndex(ctx, tailnet.ID, sanitizeHostname)
		if err != nil {
			return logError(err)
		}

		m = &domain.Machine{
			ID:                util.NextID(),
			Name:              sanitizeHostname,
			NameIdx:           nameIdx,
			MachineKey:        machineKey,
			NodeKey:           nodeKey,
			Ephemeral:         authKey.Ephemeral || req.Ephemeral,
			RegisteredTags:    registeredTags,
			Tags:              domain.SanitizeTags(tags),
			AutoAllowIPs:      autoAllowIPs,
			CreatedAt:         now,
			ExpiresAt:         now.Add(180 * 24 * time.Hour).UTC(),
			KeyExpiryDisabled: len(tags) != 0,
			Authorized:        !tailnet.MachineAuthorizationEnabled || authKey.PreAuthorized,

			User:      user,
			UserID:    user.ID,
			Tailnet:   tailnet,
			TailnetID: tailnet.ID,
		}

		if !req.Expiry.IsZero() {
			m.ExpiresAt = req.Expiry
		}

		ipv4, ipv6, err := addr.SelectIP(checkIP(ctx, h.repository.CountMachinesWithIPv4))
		if err != nil {
			return logError(err)
		}
		m.IPv4 = domain.IP{Addr: ipv4}
		m.IPv6 = domain.IP{Addr: ipv6}
	} else {
		sanitizeHostname := dnsname.SanitizeHostname(req.Hostinfo.Hostname)
		if m.Name != sanitizeHostname {
			nameIdx, err := h.repository.GetNextMachineNameIndex(ctx, tailnet.ID, sanitizeHostname)
			if err != nil {
				return logError(err)
			}
			m.Name = sanitizeHostname
			m.NameIdx = nameIdx
		}
		m.NodeKey = nodeKey
		m.Ephemeral = authKey.Ephemeral || req.Ephemeral
		m.RegisteredTags = registeredTags
		m.Tags = domain.SanitizeTags(tags)
		m.AutoAllowIPs = autoAllowIPs
		m.UserID = user.ID
		m.User = user
		m.TailnetID = tailnet.ID
		m.Tailnet = tailnet
		m.ExpiresAt = now.Add(180 * 24 * time.Hour).UTC()
	}

	if err := h.repository.SaveMachine(ctx, m); err != nil {
		return logError(err)
	}

	tUser, tLogin := mapping.ToUser(m.User)
	response := tailcfg.RegisterResponse{
		MachineAuthorized: true,
		User:              tUser,
		Login:             tLogin,
	}

	return c.JSON(http.StatusOK, response)
}

func (h *RegistrationHandlers) followup(c echo.Context, req *tailcfg.RegisterRequest) error {
	// Listen to connection close
	ctx := c.Request().Context()
	notify := ctx.Done()
	tick := time.NewTicker(2 * time.Second)

	defer func() { tick.Stop() }()

	machineKey := h.machineKey.String()

	for {
		select {
		case <-tick.C:
			m, err := h.repository.GetRegistrationRequestByMachineKey(ctx, machineKey)

			if err != nil || m == nil {
				response := tailcfg.RegisterResponse{MachineAuthorized: false, Error: "something went wrong"}
				return c.JSON(http.StatusOK, response)
			}

			if m != nil && m.Authenticated {
				user, err := h.repository.GetUser(ctx, m.UserID)
				if err != nil {
					return err
				}

				u, l := mapping.ToUser(*user)

				response := tailcfg.RegisterResponse{
					MachineAuthorized: len(m.Error) != 0,
					Error:             m.Error,
					User:              u,
					Login:             l,
				}
				return c.JSON(http.StatusOK, response)
			}

			if m != nil && len(m.Error) != 0 {
				response := tailcfg.RegisterResponse{
					MachineAuthorized: len(m.Error) != 0,
					Error:             m.Error,
				}
				return c.JSON(http.StatusOK, response)
			}
		case <-notify:
			return nil
		}
	}
}

func checkIP(cxt context.Context, s Selector) addr.Predicate {
	return func(ip netip.Addr) (bool, error) {
		c, err := s(cxt, ip.String())
		if err != nil {
			return false, err
		}
		return c == 0, nil
	}
}

type Selector func(ctx context.Context, ip string) (int64, error)
