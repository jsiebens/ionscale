package handlers

import (
	"context"
	"encoding/json"
	"github.com/jsiebens/ionscale/internal/addr"
	"github.com/jsiebens/ionscale/internal/provider"
	"github.com/mr-tron/base58"
	"net/http"
	"strconv"
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
		pendingOAuthUsers:                  cache.New(5*time.Minute, 10*time.Minute),
	}
}

type AuthenticationHandlers struct {
	repository                         domain.Repository
	config                             *config.Config
	pendingMachineRegistrationRequests *cache.Cache
	pendingOAuthUsers                  *cache.Cache
}

type AuthFormData struct {
	AuthMethods []domain.AuthMethod
}

type TailnetSelectionData struct {
	Tailnets []domain.Tailnet
}

type oauthState struct {
	Key        string
	AuthMethod uint64
}

func (h *AuthenticationHandlers) StartAuth(c echo.Context) error {
	ctx := c.Request().Context()
	key := c.Param("key")

	if _, ok := h.pendingMachineRegistrationRequests.Get(key); !ok {
		return c.Redirect(http.StatusFound, "/a/error")
	}

	methods, err := h.repository.ListAuthMethods(ctx)
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "auth.html", &AuthFormData{AuthMethods: methods})
}

func (h *AuthenticationHandlers) ProcessAuth(c echo.Context) error {
	ctx := c.Request().Context()

	key := c.Param("key")
	authKey := c.FormValue("ak")
	authMethodId := c.FormValue("s")

	if _, ok := h.pendingMachineRegistrationRequests.Get(key); !ok {
		return c.Redirect(http.StatusFound, "/a/error")
	}

	if authKey != "" {
		return h.endMachineRegistrationFlow(c, &oauthState{Key: key})
	}

	if authMethodId != "" {
		id, err := strconv.ParseUint(authMethodId, 10, 64)
		if err != nil {
			return err
		}

		method, err := h.repository.GetAuthMethod(ctx, id)
		if err != nil {
			return err
		}

		state, err := h.createState(key, method.ID)
		if err != nil {
			return err
		}

		authProvider, err := provider.NewProvider(method)
		if err != nil {
			return err
		}

		redirectUrl := authProvider.GetLoginURL(h.config.CreateUrl("/a/callback"), state)

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

	user, err := h.exchangeUser(ctx, code, state)
	if err != nil {
		return err
	}

	tailnets, err := h.repository.ListTailnets(ctx)
	if err != nil {
		return err
	}

	account, _, err := h.repository.GetOrCreateAccount(ctx, state.AuthMethod, user.ID, user.Name)
	if err != nil {
		return err
	}

	h.pendingOAuthUsers.Set(state.Key, account, cache.DefaultExpiration)

	return c.Render(http.StatusOK, "tailnets.html", &TailnetSelectionData{Tailnets: tailnets})
}

func (h *AuthenticationHandlers) EndOAuth(c echo.Context) error {
	state, err := h.readState(c.QueryParam("state"))
	if err != nil {
		return err
	}

	return h.endMachineRegistrationFlow(c, state)
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

func (h *AuthenticationHandlers) endMachineRegistrationFlow(c echo.Context, state *oauthState) error {
	ctx := c.Request().Context()

	defer h.pendingMachineRegistrationRequests.Delete(state.Key)

	preqItem, preqOK := h.pendingMachineRegistrationRequests.Get(state.Key)
	if !preqOK {
		return c.Redirect(http.StatusFound, "/a/error")
	}

	authKeyParam := c.FormValue("ak")
	tailnetIDParam := c.FormValue("s")

	preq := preqItem.(*pendingMachineRegistrationRequest)
	req := preq.request
	machineKey := preq.machineKey
	nodeKey := req.NodeKey.String()

	var tailnet *domain.Tailnet
	var user *domain.User
	var ephemeral bool
	var tags = []string{}

	if authKeyParam != "" {
		authKey, err := h.repository.LoadAuthKey(ctx, authKeyParam)
		if err != nil {
			return err
		}

		if authKey == nil {
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
	}

	var m *domain.Machine

	m, err := h.repository.GetMachineByKey(ctx, tailnet.ID, machineKey)
	if err != nil {
		return err
	}

	if m == nil {
		now := time.Now().UTC()

		registeredTags := tags
		advertisedTags := domain.SanitizeTags(req.Hostinfo.RequestTags)
		tags := append(registeredTags, advertisedTags...)

		if len(tags) != 0 {
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
			ID:             util.NextID(),
			Name:           sanitizeHostname,
			NameIdx:        nameIdx,
			MachineKey:     machineKey,
			NodeKey:        nodeKey,
			Ephemeral:      ephemeral,
			RegisteredTags: registeredTags,
			Tags:           domain.SanitizeTags(tags),
			CreatedAt:      now,

			User:    *user,
			Tailnet: *tailnet,
		}

		if !req.Expiry.IsZero() {
			m.ExpiresAt = &req.Expiry
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
		m.ExpiresAt = nil
	}

	if err := h.repository.SaveMachine(ctx, m); err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, "/a/success")
}

func (h *AuthenticationHandlers) getAuthProvider(ctx context.Context, authMethodId uint64) (provider.AuthProvider, error) {
	authMethod, err := h.repository.GetAuthMethod(ctx, authMethodId)
	if err != nil {
		return nil, err
	}
	return provider.NewProvider(authMethod)
}

func (h *AuthenticationHandlers) exchangeUser(ctx context.Context, code string, state *oauthState) (*provider.User, error) {
	redirectUrl := h.config.CreateUrl("/a/callback")

	authProvider, err := h.getAuthProvider(ctx, state.AuthMethod)
	if err != nil {
		return nil, err
	}

	user, err := authProvider.Exchange(redirectUrl, code)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (h *AuthenticationHandlers) createState(key string, authMethodId uint64) (string, error) {
	stateMap := oauthState{Key: key, AuthMethod: authMethodId}
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
