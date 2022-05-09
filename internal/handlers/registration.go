package handlers

import (
	"context"
	"github.com/jsiebens/ionscale/internal/addr"
	"github.com/jsiebens/ionscale/internal/bind"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/labstack/echo/v4"
	"github.com/patrickmn/go-cache"
	"inet.af/netaddr"
	"net/http"
	"tailscale.com/tailcfg"
	"tailscale.com/util/dnsname"
	"time"
)

func NewRegistrationHandlers(
	createBinder bind.Factory,
	config *config.Config,
	repository domain.Repository,
	pendingMachineRegistrationRequests *cache.Cache) *RegistrationHandlers {
	return &RegistrationHandlers{
		createBinder:                       createBinder,
		repository:                         repository,
		config:                             config,
		pendingMachineRegistrationRequests: pendingMachineRegistrationRequests,
	}
}

type pendingMachineRegistrationRequest struct {
	machineKey string
	request    *tailcfg.RegisterRequest
}

type RegistrationHandlers struct {
	createBinder                       bind.Factory
	repository                         domain.Repository
	config                             *config.Config
	pendingMachineRegistrationRequests *cache.Cache
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
		if m.ExpiresAt != nil && !m.ExpiresAt.IsZero() && m.ExpiresAt.Before(time.Now()) {
			response := tailcfg.RegisterResponse{NodeKeyExpired: true}
			return binder.WriteResponse(c, http.StatusOK, response)
		}

		if !req.Expiry.IsZero() && req.Expiry.Before(time.Now()) {
			m.ExpiresAt = &req.Expiry

			if err := h.repository.SaveMachine(ctx, m); err != nil {
				return err
			}

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

func (h *RegistrationHandlers) authenticateMachine(c echo.Context, binder bind.Binder, id string, req *tailcfg.RegisterRequest) error {
	if req.Followup != "" {
		response := tailcfg.RegisterResponse{AuthURL: req.Followup}
		return binder.WriteResponse(c, http.StatusOK, response)
	}

	if req.Auth.AuthKey == "" {
		key := util.RandStringBytes(8)
		authUrl := h.config.CreateUrl("/a/%s", key)

		h.pendingMachineRegistrationRequests.Set(key, &pendingMachineRegistrationRequest{
			machineKey: id,
			request:    req,
		}, cache.DefaultExpiration)

		response := tailcfg.RegisterResponse{AuthURL: authUrl}
		return binder.WriteResponse(c, http.StatusOK, response)
	} else {
		return h.authenticateMachineWithAuthKey(c, binder, id, req)
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
		return c.String(http.StatusBadRequest, "invalid auth key")
	}

	tailnet := authKey.Tailnet
	user := authKey.User

	var m *domain.Machine

	m, err = h.repository.GetMachineByKey(ctx, tailnet.ID, machineKey)
	if err != nil {
		return err
	}

	if m == nil {
		now := time.Now().UTC()

		registeredTags := authKey.Tags
		advertisedTags := domain.SanitizeTags(req.Hostinfo.RequestTags)
		tags := append(registeredTags, advertisedTags...)

		sanitizeHostname := dnsname.SanitizeHostname(req.Hostinfo.Hostname)
		nameIdx, err := h.repository.GetNextMachineNameIndex(ctx, tailnet.ID, sanitizeHostname)
		if err != nil {
			return err
		}

		m = &domain.Machine{
			ID:             util.NextID(),
			Name:           sanitizeHostname,
			NameIdx:        nameIdx,
			MachineKey:     machineKey,
			NodeKey:        nodeKey,
			Ephemeral:      authKey.Ephemeral,
			RegisteredTags: registeredTags,
			Tags:           domain.SanitizeTags(tags),
			CreatedAt:      now,

			User:    user,
			Tailnet: tailnet,
		}

		if !req.Expiry.IsZero() {
			m.ExpiresAt = &req.Expiry
		}

		ipv4, ipv6, err := addr.SelectIP(checkIP(ctx, h.repository.CountMachinesWithIPv4))
		if err != nil {
			return err
		}
		m.IPv4 = ipv4.String()
		m.IPv6 = ipv6.String()
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
		m.ExpiresAt = nil
	}

	if err := h.repository.SaveMachine(ctx, m); err != nil {
		return err
	}

	response := tailcfg.RegisterResponse{MachineAuthorized: true}
	return binder.WriteResponse(c, http.StatusOK, response)
}

func checkIP(cxt context.Context, s Selector) addr.Predicate {
	return func(ip netaddr.IP) (bool, error) {
		c, err := s(cxt, ip.String())
		if err != nil {
			return false, err
		}
		return c == 0, nil
	}
}

type Selector func(ctx context.Context, ip string) (int64, error)
