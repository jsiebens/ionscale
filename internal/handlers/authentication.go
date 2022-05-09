package handlers

import (
	"github.com/jsiebens/ionscale/internal/addr"
	"net/http"
	"time"

	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/labstack/echo/v4"
	"github.com/patrickmn/go-cache"
	"tailscale.com/util/dnsname"
)

func NewAuthenticationHandlers(
	config *config.Config,
	repository domain.Repository,
	pendingMachineRegistrationRequests *cache.Cache) *AuthenticationHandlers {
	return &AuthenticationHandlers{
		config:                             config,
		repository:                         repository,
		pendingMachineRegistrationRequests: pendingMachineRegistrationRequests,
	}
}

type AuthenticationHandlers struct {
	repository                         domain.Repository
	config                             *config.Config
	pendingMachineRegistrationRequests *cache.Cache
}

func (h *AuthenticationHandlers) StartAuth(c echo.Context) error {
	key := c.Param("key")
	authKey := c.FormValue("ak")

	if _, ok := h.pendingMachineRegistrationRequests.Get(key); !ok {
		return c.Redirect(http.StatusFound, "/a/error")
	}

	if authKey != "" {
		return h.endMachineRegistrationFlow(c, key, authKey)
	}

	return c.Render(http.StatusOK, "auth.html", nil)
}

func (h *AuthenticationHandlers) Success(c echo.Context) error {
	return c.Render(http.StatusOK, "success.html", nil)
}

func (h *AuthenticationHandlers) Error(c echo.Context) error {
	e := c.QueryParam("e")
	switch e {
	case "iak":
		return c.Render(http.StatusForbidden, "invalidauthkey.html", nil)
	}
	return c.Render(http.StatusOK, "error.html", nil)
}

func (h *AuthenticationHandlers) endMachineRegistrationFlow(c echo.Context, registrationKey, authKeyParam string) error {
	ctx := c.Request().Context()

	defer h.pendingMachineRegistrationRequests.Delete(registrationKey)

	preqItem, preqOK := h.pendingMachineRegistrationRequests.Get(registrationKey)
	if !preqOK {
		return c.Redirect(http.StatusFound, "/a/error")
	}

	preq := preqItem.(*pendingMachineRegistrationRequest)
	req := preq.request
	machineKey := preq.machineKey
	nodeKey := req.NodeKey.String()

	authKey, err := h.repository.LoadAuthKey(ctx, authKeyParam)
	if err != nil {
		return err
	}

	if authKey == nil {
		return c.Redirect(http.StatusFound, "/a/error?e=iak")
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

	return c.Redirect(http.StatusFound, "/a/success")
}
