package handlers

import (
	"context"
	"github.com/jsiebens/ionscale/internal/addr"
	"github.com/jsiebens/ionscale/internal/bind"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/netip"
	"tailscale.com/tailcfg"
	"tailscale.com/util/dnsname"
	"time"
)

func NewRegistrationHandlers(
	createBinder bind.Factory,
	config *config.Config,
	brokers broker.Pubsub,
	repository domain.Repository) *RegistrationHandlers {
	return &RegistrationHandlers{
		createBinder: createBinder,
		pubsub:       brokers,
		repository:   repository,
		config:       config,
	}
}

type RegistrationHandlers struct {
	createBinder bind.Factory
	repository   domain.Repository
	pubsub       broker.Pubsub
	config       *config.Config
}

func (h *RegistrationHandlers) Register(c echo.Context) error {
	ctx := c.Request().Context()

	binder, err := h.createBinder(c)
	if err != nil {
		return err
	}

	req := &tailcfg.RegisterRequest{}
	if err := binder.BindRequest(c, req); err != nil {
		return err
	}

	machineKey := binder.Peer().String()
	nodeKey := req.NodeKey.String()

	var m *domain.Machine
	m, err = h.repository.GetMachineByKeys(ctx, machineKey, nodeKey)

	if err != nil {
		return err
	}

	if m != nil {
		if m.IsExpired() {
			response := tailcfg.RegisterResponse{NodeKeyExpired: true}
			return binder.WriteResponse(c, http.StatusOK, response)
		}

		if !req.Expiry.IsZero() && req.Expiry.Before(time.Now()) {
			m.ExpiresAt = req.Expiry

			if err := h.repository.SaveMachine(ctx, m); err != nil {
				return err
			}

			h.pubsub.Publish(m.TailnetID, &broker.Signal{PeerUpdated: &m.ID})

			response := tailcfg.RegisterResponse{NodeKeyExpired: true}
			return binder.WriteResponse(c, http.StatusOK, response)
		}

		sanitizeHostname := dnsname.SanitizeHostname(req.Hostinfo.Hostname)
		if m.Name != sanitizeHostname {
			nameIdx, err := h.repository.GetNextMachineNameIndex(ctx, m.TailnetID, sanitizeHostname)
			if err != nil {
				return err
			}
			m.Name = sanitizeHostname
			m.NameIdx = nameIdx

		}

		advertisedTags := domain.SanitizeTags(req.Hostinfo.RequestTags)
		m.Tags = append(m.RegisteredTags, advertisedTags...)

		if err := h.repository.SaveMachine(ctx, m); err != nil {
			return err
		}

		response := tailcfg.RegisterResponse{MachineAuthorized: true}
		return binder.WriteResponse(c, http.StatusOK, response)
	}

	return h.authenticateMachine(c, binder, machineKey, req)
}

func (h *RegistrationHandlers) authenticateMachine(c echo.Context, binder bind.Binder, machineKey string, req *tailcfg.RegisterRequest) error {
	ctx := c.Request().Context()

	if req.Followup != "" {
		return h.followup(c, binder, req)
	}

	if req.Auth.AuthKey == "" {
		key := util.RandStringBytes(8)
		authUrl := h.config.CreateUrl("/a/%s", key)

		request := domain.RegistrationRequest{
			MachineKey: machineKey,
			Key:        key,
			CreatedAt:  time.Now().UTC(),
			Data:       domain.RegistrationRequestData(*req),
		}

		err := h.repository.SaveRegistrationRequest(ctx, &request)
		if err != nil {
			response := tailcfg.RegisterResponse{MachineAuthorized: false, Error: "something went wrong"}
			return binder.WriteResponse(c, http.StatusOK, response)
		}

		response := tailcfg.RegisterResponse{AuthURL: authUrl}
		return binder.WriteResponse(c, http.StatusOK, response)
	} else {
		return h.authenticateMachineWithAuthKey(c, binder, machineKey, req)
	}
}

func (h *RegistrationHandlers) authenticateMachineWithAuthKey(c echo.Context, binder bind.Binder, machineKey string, req *tailcfg.RegisterRequest) error {
	ctx := c.Request().Context()
	nodeKey := req.NodeKey.String()

	authKey, err := h.repository.LoadAuthKey(ctx, req.Auth.AuthKey)
	if err != nil {
		return err
	}

	if authKey == nil {
		response := tailcfg.RegisterResponse{MachineAuthorized: false, Error: "invalid auth key"}
		return binder.WriteResponse(c, http.StatusOK, response)
	}

	tailnet := authKey.Tailnet
	user := authKey.User

	if err := tailnet.ACLPolicy.CheckTagOwners(req.Hostinfo.RequestTags, &user); err != nil {
		response := tailcfg.RegisterResponse{MachineAuthorized: false, Error: err.Error()}
		return binder.WriteResponse(c, http.StatusOK, response)
	}

	var m *domain.Machine

	m, err = h.repository.GetMachineByKey(ctx, tailnet.ID, machineKey)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	if m == nil {
		registeredTags := authKey.Tags
		advertisedTags := domain.SanitizeTags(req.Hostinfo.RequestTags)
		tags := append(registeredTags, advertisedTags...)

		sanitizeHostname := dnsname.SanitizeHostname(req.Hostinfo.Hostname)
		nameIdx, err := h.repository.GetNextMachineNameIndex(ctx, tailnet.ID, sanitizeHostname)
		if err != nil {
			return err
		}

		m = &domain.Machine{
			ID:                util.NextID(),
			Name:              sanitizeHostname,
			NameIdx:           nameIdx,
			MachineKey:        machineKey,
			NodeKey:           nodeKey,
			Ephemeral:         authKey.Ephemeral,
			RegisteredTags:    registeredTags,
			Tags:              domain.SanitizeTags(tags),
			CreatedAt:         now,
			ExpiresAt:         now.Add(180 * 24 * time.Hour).UTC(),
			KeyExpiryDisabled: len(tags) != 0,

			User:    user,
			Tailnet: tailnet,
		}

		if !req.Expiry.IsZero() {
			m.ExpiresAt = req.Expiry
		}

		ipv4, ipv6, err := addr.SelectIP(checkIP(ctx, h.repository.CountMachinesWithIPv4))
		if err != nil {
			return err
		}
		m.IPv4 = domain.IP{Addr: ipv4}
		m.IPv6 = domain.IP{Addr: ipv6}
	} else {
		registeredTags := authKey.Tags
		advertisedTags := domain.SanitizeTags(req.Hostinfo.RequestTags)
		tags := append(registeredTags, advertisedTags...)

		sanitizeHostname := dnsname.SanitizeHostname(req.Hostinfo.Hostname)
		if m.Name != sanitizeHostname {
			nameIdx, err := h.repository.GetNextMachineNameIndex(ctx, tailnet.ID, sanitizeHostname)
			if err != nil {
				return err
			}
			m.Name = sanitizeHostname
			m.NameIdx = nameIdx
		}
		m.NodeKey = nodeKey
		m.Ephemeral = authKey.Ephemeral
		m.RegisteredTags = registeredTags
		m.Tags = domain.SanitizeTags(tags)
		m.UserID = user.ID
		m.User = user
		m.TailnetID = tailnet.ID
		m.Tailnet = tailnet
		m.ExpiresAt = now.Add(180 * 24 * time.Hour).UTC()
	}

	if err := h.repository.SaveMachine(ctx, m); err != nil {
		return err
	}

	response := tailcfg.RegisterResponse{MachineAuthorized: true}
	return binder.WriteResponse(c, http.StatusOK, response)
}

func (h *RegistrationHandlers) followup(c echo.Context, binder bind.Binder, req *tailcfg.RegisterRequest) error {
	// Listen to connection close
	ctx := c.Request().Context()
	notify := ctx.Done()
	tick := time.NewTicker(2 * time.Second)

	defer func() { tick.Stop() }()

	machineKey := binder.Peer().String()

	for {
		select {
		case <-tick.C:
			m, err := h.repository.GetRegistrationRequestByMachineKey(ctx, machineKey)

			if err != nil || m == nil {
				response := tailcfg.RegisterResponse{MachineAuthorized: false, Error: "something went wrong"}
				return binder.WriteResponse(c, http.StatusOK, response)
			}

			if m != nil && m.IsFinished() {
				response := tailcfg.RegisterResponse{MachineAuthorized: len(m.Error) != 0, Error: m.Error}
				return binder.WriteResponse(c, http.StatusOK, response)
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
