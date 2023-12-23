package handlers

import (
	"fmt"
	"github.com/go-jose/go-jose/v3"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jsiebens/ionscale/internal/bind"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/labstack/echo/v4"
	"net/http"
	"tailscale.com/tailcfg"
	"time"
)

func NewIDTokenHandlers(createBinder bind.Factory, config *config.Config, repository domain.Repository) *IDTokenHandlers {
	return &IDTokenHandlers{
		issuer:       config.ServerUrl,
		jwksUri:      config.CreateUrl("/.well-known/jwks"),
		createBinder: createBinder,
		repository:   repository,
	}
}

type IDTokenHandlers struct {
	issuer       string
	jwksUri      string
	createBinder bind.Factory
	repository   domain.Repository
}

func (h *IDTokenHandlers) OpenIDConfig(c echo.Context) error {
	v := map[string]interface{}{}

	v["issuer"] = h.issuer
	v["jwks_uri"] = h.jwksUri
	v["subject_types_supported"] = []string{"public"}
	v["response_types_supported"] = []string{"id_token"}
	v["scopes_supported"] = []string{"openid"}
	v["id_token_signing_alg_values_supported"] = []string{"RS256"}
	v["claims_supported"] = []string{
		"sub",
		"aud",
		"exp",
		"iat",
		"iss",
		"jti",
		"nbf",
	}

	return c.JSON(http.StatusOK, v)
}

func (h *IDTokenHandlers) Jwks(c echo.Context) error {
	keySet, err := h.repository.GetJSONWebKeySet(c.Request().Context())
	if err != nil {
		return logError(err)
	}

	pub := jose.JSONWebKey{Key: keySet.Key.Public(), KeyID: keySet.Key.Id, Algorithm: "RS256", Use: "sig"}
	set := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{pub}}
	return c.JSON(http.StatusOK, set)
}

func (h *IDTokenHandlers) FetchToken(c echo.Context) error {
	ctx := c.Request().Context()

	keySet, err := h.repository.GetJSONWebKeySet(c.Request().Context())
	if err != nil {
		return logError(err)
	}

	binder, err := h.createBinder(c)
	if err != nil {
		return logError(err)
	}

	req := &tailcfg.TokenRequest{}
	if err := binder.BindRequest(c, req); err != nil {
		return logError(err)
	}

	machineKey := binder.Peer().String()
	nodeKey := req.NodeKey.String()

	var m *domain.Machine
	m, err = h.repository.GetMachineByKeys(ctx, machineKey, nodeKey)
	if err != nil {
		return logError(err)
	}

	if m == nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	_, tailnetDomain, sub := h.names(m)

	now := time.Now()

	claims := jwt.MapClaims{
		"jit": fmt.Sprintf("%d", util.NextID()),
		"iss": h.issuer,
		"sub": sub,
		"aud": []string{req.Audience},
		"exp": jwt.NewNumericDate(now.Add(5 * time.Minute)),
		"nbf": jwt.NewNumericDate(now),
		"iat": jwt.NewNumericDate(now),

		"key":       m.NodeKey,
		"addresses": []string{m.IPv4.String(), m.IPv6.String()},
		"nid":       m.ID,
		"node":      sub,
		"domain":    tailnetDomain,
	}

	if m.HasTags() {
		tags := []string{}
		for _, t := range m.Tags {
			tags = append(tags, fmt.Sprintf("%s:%s", tailnetDomain, t))
		}
		claims["tags"] = tags
	} else {
		claims["user"] = fmt.Sprintf("%s:%s", tailnetDomain, m.User.Name)
		claims["uid"] = m.UserID
	}

	unsignedToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	unsignedToken.Header["kid"] = keySet.Key.Id

	jwtB64, err := unsignedToken.SignedString(&keySet.Key.PrivateKey)
	if err != nil {
		return logError(err)
	}

	resp := tailcfg.TokenResponse{IDToken: jwtB64}
	return binder.WriteResponse(c, http.StatusOK, resp)
}

func (h *IDTokenHandlers) names(m *domain.Machine) (string, string, string) {
	var name = m.Name
	if m.NameIdx != 0 {
		name = fmt.Sprintf("%s-%d", m.Name, m.NameIdx)
	}

	sanitizedTailnetName := domain.SanitizeTailnetName(m.Tailnet.Name)
	return name, sanitizedTailnetName, fmt.Sprintf("%s.%s", name, sanitizedTailnetName)
}
