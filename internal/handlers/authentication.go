package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jsiebens/ionscale/internal/addr"
	"github.com/jsiebens/ionscale/internal/auth"
	tpl "github.com/jsiebens/ionscale/internal/templates"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mr-tron/base58"
	"net/http"
	"tailscale.com/tailcfg"
	"time"

	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/labstack/echo/v4"
	"tailscale.com/util/dnsname"
)

func NewAuthenticationHandlers(
	config *config.Config,
	authProvider auth.Provider,
	systemIAMPolicy *domain.IAMPolicy,
	repository domain.Repository) *AuthenticationHandlers {

	return &AuthenticationHandlers{
		config:          config,
		authProvider:    authProvider,
		repository:      repository,
		systemIAMPolicy: systemIAMPolicy,
	}
}

type AuthenticationHandlers struct {
	repository      domain.Repository
	authProvider    auth.Provider
	config          *config.Config
	systemIAMPolicy *domain.IAMPolicy
}

type AuthInput struct {
	Key     string   `param:"key"`
	Flow    AuthFlow `param:"flow"`
	AuthKey string   `query:"ak" form:"ak"`
	Oidc    bool     `query:"oidc" form:"oidc"`
}

type EndAuthForm struct {
	AccountID     uint64 `form:"aid"`
	TailnetID     uint64 `form:"tid"`
	AsSystemAdmin bool   `form:"sad"`
	AuthKey       string `form:"ak"`
	State         string `form:"state"`
}

type oauthState struct {
	Key  string
	Flow AuthFlow
}

type AuthFlow string

const (
	AuthFlowMachineRegistration = "r"
	AuthFlowClient              = "c"
	AuthFlowSSHCheckFlow        = "s"
)

func (h *AuthenticationHandlers) StartAuth(c echo.Context) error {
	ctx := c.Request().Context()

	var input AuthInput
	if err := c.Bind(&input); err != nil {
		return logError(err)
	}

	// machine registration auth flow
	if input.Flow == AuthFlowMachineRegistration {
		req, err := h.repository.GetRegistrationRequestByKey(ctx, input.Key)
		if err != nil || req == nil {
			return logError(err)
		}

		if input.Oidc && h.authProvider != nil {
			goto startOidc
		}

		if input.AuthKey != "" {
			return h.endMachineRegistrationFlow(c, EndAuthForm{AuthKey: input.AuthKey}, req)
		}

		csrf := c.Get(middleware.DefaultCSRFConfig.ContextKey).(string)
		return c.Render(http.StatusOK, "", tpl.Auth(h.authProvider != nil, csrf))
	}

	// cli auth flow
	if input.Flow == AuthFlowClient {
		if s, err := h.repository.GetAuthenticationRequest(ctx, input.Key); err != nil || s == nil {
			return logError(err)
		}
	}

	// ssh check auth flow
	if input.Flow == AuthFlowSSHCheckFlow {
		if s, err := h.repository.GetSSHActionRequest(ctx, input.Key); err != nil || s == nil {
			return logError(err)
		}
	}

	if h.authProvider == nil {
		return logError(fmt.Errorf("unable to start auth flow as no auth provider is configured"))
	}

startOidc:

	state, err := h.createState(input.Flow, input.Key)
	if err != nil {
		return logError(err)
	}

	redirectUrl := h.authProvider.GetLoginURL(h.config.CreateUrl("/a/callback"), state)

	return c.Redirect(http.StatusFound, redirectUrl)
}

func (h *AuthenticationHandlers) ProcessAuth(c echo.Context) error {
	ctx := c.Request().Context()

	var input AuthInput
	if err := c.Bind(&input); err != nil {
		return logError(err)
	}

	req, err := h.repository.GetRegistrationRequestByKey(ctx, input.Key)
	if err != nil || req == nil {
		return logError(err)
	}

	if input.AuthKey != "" {
		return h.endMachineRegistrationFlow(c, EndAuthForm{AuthKey: input.AuthKey}, req)
	}

	if input.Oidc {
		state, err := h.createState(input.Flow, input.Key)
		if err != nil {
			return logError(err)
		}

		redirectUrl := h.authProvider.GetLoginURL(h.config.CreateUrl("/a/callback"), state)

		return c.Redirect(http.StatusFound, redirectUrl)
	}

	return c.Redirect(http.StatusFound, fmt.Sprintf("/a/%s/%s", input.Flow, input.Key))
}

func (h *AuthenticationHandlers) Callback(c echo.Context) error {
	ctx := c.Request().Context()

	code := c.QueryParam("code")
	state, err := h.readState(c.QueryParam("state"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid state parameter")
	}

	user, err := h.exchangeUser(code)
	if err != nil {
		return logError(err)
	}

	account, _, err := h.repository.GetOrCreateAccount(ctx, user.ID, user.Name)
	if err != nil {
		return logError(err)
	}

	if err := h.repository.SetAccountLastAuthenticated(ctx, account.ID); err != nil {
		return logError(err)
	}

	if state.Flow == AuthFlowSSHCheckFlow {
		sshActionReq, err := h.repository.GetSSHActionRequest(ctx, state.Key)
		if err != nil || sshActionReq == nil {
			return c.Redirect(http.StatusFound, "/a/error?e=ua")
		}

		machine, err := h.repository.GetMachine(ctx, sshActionReq.SrcMachineID)
		if err != nil || sshActionReq == nil {
			return logError(err)
		}

		if !machine.HasTags() && machine.User.AccountID != nil && *machine.User.AccountID == account.ID {
			sshActionReq.Action = "accept"

			err := h.repository.Transaction(func(rp domain.Repository) error {
				if err := rp.SetUserLastAuthenticated(ctx, machine.UserID, time.Now().UTC()); err != nil {
					return err
				}
				if err := rp.SaveSSHActionRequest(ctx, sshActionReq); err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return logError(err)
			}

			return c.Redirect(http.StatusFound, "/a/success")
		}

		sshActionReq.Action = "reject"
		if err := h.repository.SaveSSHActionRequest(ctx, sshActionReq); err != nil {
			return logError(err)
		}
		return c.Redirect(http.StatusFound, "/a/error?e=nmo")
	}

	tailnets, err := h.listAvailableTailnets(ctx, user)
	if err != nil {
		return logError(err)
	}

	csrf := c.Get(middleware.DefaultCSRFConfig.ContextKey).(string)

	if state.Flow == AuthFlowMachineRegistration {
		if len(tailnets) == 0 {
			registrationRequest, err := h.repository.GetRegistrationRequestByKey(ctx, state.Key)
			if err == nil && registrationRequest != nil {
				registrationRequest.Error = "unauthorized"
				_ = h.repository.SaveRegistrationRequest(ctx, registrationRequest)
			}
			return c.Redirect(http.StatusFound, "/a/error?e=ua")
		}

		if len(tailnets) == 1 {
			req, err := h.repository.GetRegistrationRequestByKey(ctx, state.Key)
			if err != nil {
				return logError(err)
			}
			if req == nil {
				return logError(fmt.Errorf("invalid registration key"))
			}
			return h.endMachineRegistrationFlow(c, EndAuthForm{AccountID: account.ID, TailnetID: tailnets[0].ID}, req)
		}

		return c.Render(http.StatusOK, "", tpl.Tailnets(account.ID, false, tailnets, csrf))
	}

	if state.Flow == AuthFlowClient {
		isSystemAdmin, err := h.isSystemAdmin(user)
		if err != nil {
			return logError(err)
		}

		if !isSystemAdmin && len(tailnets) == 0 {
			req, err := h.repository.GetAuthenticationRequest(ctx, state.Key)
			if err == nil && req != nil {
				req.Error = "unauthorized"
				_ = h.repository.SaveAuthenticationRequest(ctx, req)
			}
			return c.Redirect(http.StatusFound, "/a/error?e=ua")
		}

		return c.Render(http.StatusOK, "", tpl.Tailnets(account.ID, isSystemAdmin, tailnets, csrf))
	}

	return echo.NewHTTPError(http.StatusNotFound)
}

func (h *AuthenticationHandlers) EndAuth(c echo.Context) error {
	ctx := c.Request().Context()

	var form EndAuthForm
	if err := c.Bind(&form); err != nil {
		return logError(err)
	}

	state, err := h.readState(form.State)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid state parameter")
	}

	if state.Flow == AuthFlowMachineRegistration {
		req, err := h.repository.GetRegistrationRequestByKey(ctx, state.Key)
		if err != nil || req == nil {
			return logError(err)
		}

		return h.endMachineRegistrationFlow(c, form, req)
	}

	if state.Flow == AuthFlowClient {
		req, err := h.repository.GetAuthenticationRequest(ctx, state.Key)
		if err != nil || req == nil {
			return logError(err)
		}

		return h.endCliAuthenticationFlow(c, form, req)
	}

	return echo.NewHTTPError(http.StatusBadRequest, "Invalid state parameter")
}

func (h *AuthenticationHandlers) Success(c echo.Context) error {
	s := c.QueryParam("s")
	switch s {
	case "nma":
		return c.Render(http.StatusOK, "", tpl.NewMachine())
	}
	return c.Render(http.StatusOK, "", tpl.Success())
}

func (h *AuthenticationHandlers) Error(c echo.Context) error {
	e := c.QueryParam("e")
	switch e {
	case "iak":
		return c.Render(http.StatusForbidden, "", tpl.InvalidAuthKey())
	case "ua":
		return c.Render(http.StatusForbidden, "", tpl.Unauthorized())
	case "nto":
		return c.Render(http.StatusForbidden, "", tpl.NotTagOwner())
	case "nmo":
		return c.Render(http.StatusForbidden, "", tpl.NotMachineOwner())
	}
	return c.Render(http.StatusOK, "", tpl.Error())
}

func (h *AuthenticationHandlers) endCliAuthenticationFlow(c echo.Context, form EndAuthForm, req *domain.AuthenticationRequest) error {
	ctx := c.Request().Context()

	account, err := h.repository.GetAccount(ctx, form.AccountID)
	if err != nil {
		return logError(err)
	}

	// continue as system admin?
	if form.AsSystemAdmin {
		expiresAt := time.Now().Add(24 * time.Hour)
		token, apiKey := domain.CreateSystemApiKey(account, &expiresAt)
		req.Token = token

		err := h.repository.Transaction(func(rp domain.Repository) error {
			if err := rp.SaveSystemApiKey(ctx, apiKey); err != nil {
				return logError(err)
			}
			if err := rp.SaveAuthenticationRequest(ctx, req); err != nil {
				return logError(err)
			}
			return nil
		})
		if err != nil {
			return logError(err)
		}
		return c.Redirect(http.StatusFound, "/a/success")
	}

	tailnet, err := h.repository.GetTailnet(ctx, form.TailnetID)
	if err != nil {
		return logError(err)
	}

	user, _, err := h.repository.GetOrCreateUserWithAccount(ctx, tailnet, account)
	if err != nil {
		return logError(err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	token, apiKey := domain.CreateApiKey(tailnet, user, &expiresAt)
	req.Token = token
	req.TailnetID = &tailnet.ID

	err = h.repository.Transaction(func(rp domain.Repository) error {
		if err := rp.SetUserLastAuthenticated(ctx, user.ID, time.Now().UTC()); err != nil {
			return err
		}
		if err := rp.SaveApiKey(ctx, apiKey); err != nil {
			return err
		}
		if err := rp.SaveAuthenticationRequest(ctx, req); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return logError(err)
	}

	return c.Redirect(http.StatusFound, "/a/success")
}

func (h *AuthenticationHandlers) endMachineRegistrationFlow(c echo.Context, form EndAuthForm, registrationRequest *domain.RegistrationRequest) error {
	ctx := c.Request().Context()

	req := tailcfg.RegisterRequest(registrationRequest.Data)
	machineKey := registrationRequest.MachineKey
	nodeKey := req.NodeKey.String()

	var tailnet *domain.Tailnet
	var user *domain.User
	var ephemeral bool
	var tags = []string{}
	var authorized = false

	if form.AuthKey != "" {
		authKey, err := h.repository.LoadAuthKey(ctx, form.AuthKey)
		if err != nil {
			return logError(err)
		}

		if authKey == nil {

			registrationRequest.Authenticated = false
			registrationRequest.Error = "invalid auth key"

			if err := h.repository.SaveRegistrationRequest(ctx, registrationRequest); err != nil {
				return logError(err)
			}

			return c.Redirect(http.StatusFound, "/a/error?e=iak")
		}

		tailnet = &authKey.Tailnet
		user = &authKey.User
		tags = authKey.Tags
		ephemeral = authKey.Ephemeral
		authorized = authKey.PreAuthorized
	} else {
		selectedTailnet, err := h.repository.GetTailnet(ctx, form.TailnetID)
		if err != nil {
			return logError(err)
		}

		account, err := h.repository.GetAccount(ctx, form.AccountID)
		if err != nil {
			return logError(err)
		}

		selectedUser, _, err := h.repository.GetOrCreateUserWithAccount(ctx, selectedTailnet, account)
		if err != nil {
			return logError(err)
		}

		user = selectedUser
		tailnet = selectedTailnet
		ephemeral = false
	}

	if err := tailnet.ACLPolicy.CheckTagOwners(registrationRequest.Data.Hostinfo.RequestTags, user); err != nil {
		registrationRequest.Authenticated = false
		registrationRequest.Error = err.Error()
		if err := h.repository.SaveRegistrationRequest(ctx, registrationRequest); err != nil {
			return logError(err)
		}
		return c.Redirect(http.StatusFound, "/a/error?e=nto")
	}

	autoAllowIPs := tailnet.ACLPolicy.FindAutoApprovedIPs(req.Hostinfo.RoutableIPs, tags, user)

	var m *domain.Machine

	m, err := h.repository.GetMachineByKeyAndUser(ctx, machineKey, user.ID)
	if err != nil {
		return logError(err)
	}

	now := time.Now().UTC()

	if m == nil {
		registeredTags := tags
		advertisedTags := domain.SanitizeTags(req.Hostinfo.RequestTags)
		tags := append(registeredTags, advertisedTags...)

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
			Ephemeral:         ephemeral || req.Ephemeral,
			RegisteredTags:    registeredTags,
			Tags:              domain.SanitizeTags(tags),
			AutoAllowIPs:      autoAllowIPs,
			CreatedAt:         now,
			ExpiresAt:         now.Add(180 * 24 * time.Hour).UTC(),
			KeyExpiryDisabled: len(tags) != 0,
			Authorized:        !tailnet.MachineAuthorizationEnabled || authorized,

			User:      *user,
			UserID:    user.ID,
			Tailnet:   *tailnet,
			TailnetID: tailnet.ID,
		}

		ipv4, ipv6, err := addr.SelectIP(checkIP(ctx, h.repository.CountMachinesWithIPv4))
		if err != nil {
			return logError(err)
		}
		m.IPv4 = domain.IP{Addr: ipv4}
		m.IPv6 = domain.IP{Addr: ipv6}
	} else {
		registeredTags := tags
		advertisedTags := domain.SanitizeTags(req.Hostinfo.RequestTags)
		tags := append(registeredTags, advertisedTags...)

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
		m.Ephemeral = ephemeral || req.Ephemeral
		m.RegisteredTags = registeredTags
		m.Tags = domain.SanitizeTags(tags)
		m.AutoAllowIPs = autoAllowIPs
		m.UserID = user.ID
		m.User = *user
		m.TailnetID = tailnet.ID
		m.Tailnet = *tailnet
		m.ExpiresAt = now.Add(180 * 24 * time.Hour).UTC()
	}

	err = h.repository.Transaction(func(rp domain.Repository) error {
		registrationRequest.Authenticated = true
		registrationRequest.Error = ""
		registrationRequest.UserID = user.ID

		if err := rp.SetUserLastAuthenticated(ctx, m.UserID, time.Now().UTC()); err != nil {
			return err
		}

		if err := rp.SaveMachine(ctx, m); err != nil {
			return err
		}

		if err := rp.SaveRegistrationRequest(ctx, registrationRequest); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return logError(err)
	}

	if m.Authorized {
		return c.Redirect(http.StatusFound, "/a/success")
	} else {
		return c.Redirect(http.StatusFound, "/a/success?s=nma")
	}
}

func (h *AuthenticationHandlers) isSystemAdmin(u *auth.User) (bool, error) {
	return h.systemIAMPolicy.EvaluatePolicy(&domain.Identity{UserID: u.ID, Email: u.Name, Attr: u.Attr})
}

func (h *AuthenticationHandlers) listAvailableTailnets(ctx context.Context, u *auth.User) ([]domain.Tailnet, error) {
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

func (h *AuthenticationHandlers) exchangeUser(code string) (*auth.User, error) {
	redirectUrl := h.config.CreateUrl("/a/callback")

	user, err := h.authProvider.Exchange(redirectUrl, code)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (h *AuthenticationHandlers) createState(flow AuthFlow, key string) (string, error) {
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
