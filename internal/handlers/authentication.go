package handlers

import (
	"context"
	"encoding/json"
	"github.com/jsiebens/ionscale/internal/addr"
	"github.com/jsiebens/ionscale/internal/provider"
	"github.com/mr-tron/base58"
	"net/http"
	"strconv"
	"tailscale.com/tailcfg"
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
	authProvider provider.AuthProvider,
	repository domain.Repository) *AuthenticationHandlers {
	return &AuthenticationHandlers{
		config:            config,
		authProvider:      authProvider,
		repository:        repository,
		pendingOAuthUsers: cache.New(5*time.Minute, 10*time.Minute),
	}
}

type AuthenticationHandlers struct {
	repository        domain.Repository
	authProvider      provider.AuthProvider
	config            *config.Config
	pendingOAuthUsers *cache.Cache
}

type AuthFormData struct {
	ProviderAvailable bool
}

type TailnetSelectionData struct {
	Tailnets []domain.Tailnet
}

type oauthState struct {
	Key  string
	Flow string
}

func (h *AuthenticationHandlers) StartCliAuth(c echo.Context) error {
	ctx := c.Request().Context()
	key := c.Param("key")

	if s, err := h.repository.GetAuthenticationRequest(ctx, key); err != nil || s == nil {
		return c.Redirect(http.StatusFound, "/a/error")
	}

	if h.authProvider == nil {
		return c.Redirect(http.StatusFound, "/a/error")
	}

	state, err := h.createState("c", key)
	if err != nil {
		return err
	}

	redirectUrl := h.authProvider.GetLoginURL(h.config.CreateUrl("/a/callback"), state)

	return c.Redirect(http.StatusFound, redirectUrl)
}

func (h *AuthenticationHandlers) StartAuth(c echo.Context) error {
	ctx := c.Request().Context()
	key := c.Param("key")

	if req, err := h.repository.GetRegistrationRequestByKey(ctx, key); err != nil || req == nil {
		return c.Redirect(http.StatusFound, "/a/error")
	}

	return c.Render(http.StatusOK, "auth.html", &AuthFormData{ProviderAvailable: h.authProvider != nil})
}

func (h *AuthenticationHandlers) ProcessAuth(c echo.Context) error {
	ctx := c.Request().Context()

	key := c.Param("key")
	authKey := c.FormValue("ak")
	interactive := c.FormValue("s")

	req, err := h.repository.GetRegistrationRequestByKey(ctx, key)
	if err != nil || req == nil {
		return c.Redirect(http.StatusFound, "/a/error")
	}

	if authKey != "" {
		return h.endMachineRegistrationFlow(c, req, &oauthState{Key: key})
	}

	if interactive != "" {
		state, err := h.createState("r", key)
		if err != nil {
			return err
		}

		redirectUrl := h.authProvider.GetLoginURL(h.config.CreateUrl("/a/callback"), state)

		return c.Redirect(http.StatusFound, redirectUrl)
	}

	return c.Redirect(http.StatusFound, "/a/"+key)
}

func (h *AuthenticationHandlers) Callback(c echo.Context) error {
	ctx := c.Request().Context()

	code := c.QueryParam("code")
	state, err := h.readState(c.QueryParam("state"))
	if err != nil {
		return err
	}

	user, err := h.exchangeUser(code)
	if err != nil {
		return err
	}

	tailnets, err := h.listAvailableTailnets(ctx, user)
	if err != nil {
		return err
	}

	if len(tailnets) == 0 {
		if state.Flow == "r" {
			req, err := h.repository.GetRegistrationRequestByKey(ctx, state.Key)
			if err == nil && req != nil {
				req.Error = "unauthorized"
				_ = h.repository.SaveRegistrationRequest(ctx, req)
			}
		} else {
			req, err := h.repository.GetAuthenticationRequest(ctx, state.Key)
			if err == nil && req != nil {
				req.Error = "unauthorized"
				_ = h.repository.SaveAuthenticationRequest(ctx, req)
			}
		}
		return c.Redirect(http.StatusFound, "/a/error?e=ua")
	}

	account, _, err := h.repository.GetOrCreateAccount(ctx, user.ID, user.Name)
	if err != nil {
		return err
	}

	h.pendingOAuthUsers.Set(state.Key, account, cache.DefaultExpiration)

	return c.Render(http.StatusOK, "tailnets.html", &TailnetSelectionData{Tailnets: tailnets})
}

func (h *AuthenticationHandlers) listAvailableTailnets(ctx context.Context, u *provider.User) ([]domain.Tailnet, error) {
	var result = []domain.Tailnet{}
	tailnets, err := h.repository.ListTailnets(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range tailnets {
		approved, err := t.IAMPolicy.EvaluatePolicy(&domain.Identity{UserID: u.ID, Email: u.Name, Attr: u.Attr})
		if err != nil {
			return nil, err
		}
		if approved {
			result = append(result, t)
		}
	}
	return result, nil
}

func (h *AuthenticationHandlers) EndOAuth(c echo.Context) error {
	ctx := c.Request().Context()

	state, err := h.readState(c.QueryParam("state"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/a/error")
	}

	if state.Flow == "r" {
		req, err := h.repository.GetRegistrationRequestByKey(ctx, state.Key)
		if err != nil || req == nil {
			return c.Redirect(http.StatusFound, "/a/error")
		}

		return h.endMachineRegistrationFlow(c, req, state)
	}

	req, err := h.repository.GetAuthenticationRequest(ctx, state.Key)
	if err != nil || req == nil {
		return c.Redirect(http.StatusFound, "/a/error")
	}

	return h.endCliAuthenticationFlow(c, req, state)
}

func (h *AuthenticationHandlers) Success(c echo.Context) error {
	return c.Render(http.StatusOK, "success.html", nil)
}

func (h *AuthenticationHandlers) Error(c echo.Context) error {
	e := c.QueryParam("e")
	switch e {
	case "iak":
		return c.Render(http.StatusForbidden, "invalidauthkey.html", nil)
	case "ua":
		return c.Render(http.StatusForbidden, "unauthorized.html", nil)
	}
	return c.Render(http.StatusOK, "error.html", nil)
}

func (h *AuthenticationHandlers) endCliAuthenticationFlow(c echo.Context, req *domain.AuthenticationRequest, state *oauthState) error {
	ctx := c.Request().Context()

	tailnetIDParam := c.FormValue("s")

	parseUint, err := strconv.ParseUint(tailnetIDParam, 10, 64)
	if err != nil {
		return err
	}
	tailnet, err := h.repository.GetTailnet(ctx, parseUint)
	if err != nil {
		return err
	}

	item, ok := h.pendingOAuthUsers.Get(state.Key)
	if !ok {
		return c.Redirect(http.StatusFound, "/a/error")
	}

	oa := item.(*domain.Account)

	user, _, err := h.repository.GetOrCreateUserWithAccount(ctx, tailnet, oa)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	token, apiKey := domain.CreateApiKey(tailnet, user, &expiresAt)
	req.Token = token

	err = h.repository.Transaction(func(rp domain.Repository) error {
		if err := rp.SaveApiKey(ctx, apiKey); err != nil {
			return err
		}
		if err := rp.SaveAuthenticationRequest(ctx, req); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, "/a/success")
}

func (h *AuthenticationHandlers) endMachineRegistrationFlow(c echo.Context, registrationRequest *domain.RegistrationRequest, state *oauthState) error {
	ctx := c.Request().Context()

	authKeyParam := c.FormValue("ak")
	tailnetIDParam := c.FormValue("s")

	req := tailcfg.RegisterRequest(registrationRequest.Data)
	machineKey := registrationRequest.MachineKey
	nodeKey := req.NodeKey.String()

	var tailnet *domain.Tailnet
	var user *domain.User
	var ephemeral bool
	var tags = []string{}
	var expiryDisabled bool

	if authKeyParam != "" {
		authKey, err := h.repository.LoadAuthKey(ctx, authKeyParam)
		if err != nil {
			return err
		}

		if authKey == nil {

			registrationRequest.Authenticated = false
			registrationRequest.Error = "invalid auth key"

			if err := h.repository.SaveRegistrationRequest(ctx, registrationRequest); err != nil {
				return c.Redirect(http.StatusFound, "/a/error")
			}

			return c.Redirect(http.StatusFound, "/a/error?e=iak")
		}

		tailnet = &authKey.Tailnet
		user = &authKey.User
		tags = authKey.Tags
		ephemeral = authKey.Ephemeral
	} else {
		parseUint, err := strconv.ParseUint(tailnetIDParam, 10, 64)
		if err != nil {
			return err
		}
		tailnet, err = h.repository.GetTailnet(ctx, parseUint)
		if err != nil {
			return err
		}

		item, ok := h.pendingOAuthUsers.Get(state.Key)
		if !ok {
			return c.Redirect(http.StatusFound, "/a/error")
		}

		oa := item.(*domain.Account)

		user, _, err = h.repository.GetOrCreateUserWithAccount(ctx, tailnet, oa)
		if err != nil {
			return err
		}

		ephemeral = false
		expiryDisabled = false
	}

	var m *domain.Machine

	m, err := h.repository.GetMachineByKey(ctx, tailnet.ID, machineKey)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	if m == nil {
		registeredTags := tags
		advertisedTags := domain.SanitizeTags(req.Hostinfo.RequestTags)
		tags := append(registeredTags, advertisedTags...)

		if len(tags) != 0 {
			expiryDisabled = true
			user, _, err = h.repository.GetOrCreateServiceUser(ctx, tailnet)
			if err != nil {
				return err
			}
		}

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
			Ephemeral:         ephemeral,
			RegisteredTags:    registeredTags,
			Tags:              domain.SanitizeTags(tags),
			CreatedAt:         now,
			ExpiresAt:         now.Add(180 * 24 * time.Hour).UTC(),
			KeyExpiryDisabled: expiryDisabled,

			User:    *user,
			Tailnet: *tailnet,
		}

		ipv4, ipv6, err := addr.SelectIP(checkIP(ctx, h.repository.CountMachinesWithIPv4))
		if err != nil {
			return err
		}
		m.IPv4 = domain.IP{IP: ipv4}
		m.IPv6 = domain.IP{IP: ipv6}
	} else {
		registeredTags := tags
		advertisedTags := domain.SanitizeTags(req.Hostinfo.RequestTags)
		tags := append(registeredTags, advertisedTags...)

		if len(tags) != 0 {
			expiryDisabled = true
			user, _, err = h.repository.GetOrCreateServiceUser(ctx, tailnet)
			if err != nil {
				return err
			}
		}

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
		m.Ephemeral = ephemeral
		m.RegisteredTags = registeredTags
		m.Tags = domain.SanitizeTags(tags)
		m.UserID = user.ID
		m.User = *user
		m.TailnetID = tailnet.ID
		m.Tailnet = *tailnet
		m.ExpiresAt = now.Add(180 * 24 * time.Hour).UTC()
		m.KeyExpiryDisabled = expiryDisabled
	}

	err = h.repository.Transaction(func(rp domain.Repository) error {
		registrationRequest.Authenticated = true
		registrationRequest.Error = ""

		if err := rp.SaveMachine(ctx, m); err != nil {
			return err
		}

		if err := rp.SaveRegistrationRequest(ctx, registrationRequest); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, "/a/success")
}

func (h *AuthenticationHandlers) exchangeUser(code string) (*provider.User, error) {
	redirectUrl := h.config.CreateUrl("/a/callback")

	user, err := h.authProvider.Exchange(redirectUrl, code)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (h *AuthenticationHandlers) createState(flow string, key string) (string, error) {
	stateMap := oauthState{Key: key, Flow: flow}
	marshal, err := json.Marshal(&stateMap)
	if err != nil {
		return "", err
	}
	return base58.FastBase58Encoding(marshal), nil
}

func (h *AuthenticationHandlers) readState(s string) (*oauthState, error) {
	decodedState, err := base58.FastBase58Decoding(s)
	if err != nil {
		return nil, err
	}

	var state = &oauthState{}
	if err := json.Unmarshal(decodedState, state); err != nil {
		return nil, err
	}
	return state, nil
}
