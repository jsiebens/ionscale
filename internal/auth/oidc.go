package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jsiebens/ionscale/internal/config"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type OIDCProvider struct {
	clientID     string
	clientSecret string
	scopes       []string
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
}

func NewOIDCProvider(c *config.AuthProvider) (*OIDCProvider, error) {
	defaultScopes := []string{oidc.ScopeOpenID, "email", "profile"}
	provider, err := oidc.NewProvider(context.Background(), c.Issuer)
	if err != nil {
		return nil, err
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: c.ClientID, SkipClientIDCheck: c.ClientID == ""})

	return &OIDCProvider{
		clientID:     c.ClientID,
		clientSecret: c.ClientSecret,
		scopes:       append(defaultScopes, c.Scopes...),
		provider:     provider,
		verifier:     verifier,
	}, nil
}

func (p *OIDCProvider) GetLoginURL(redirectURI, state string) string {
	oauth2Config := oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     p.provider.Endpoint(),
		Scopes:       p.scopes,
	}

	return oauth2Config.AuthCodeURL(state, oauth2.ApprovalForce)
}

func (p *OIDCProvider) Exchange(redirectURI, code string) (*User, error) {
	oauth2Config := oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     p.provider.Endpoint(),
		Scopes:       p.scopes,
	}

	oauth2Token, err := oauth2Config.Exchange(context.Background(), code)

	if err != nil {
		return nil, err
	}

	// Extract the ID Token from OAuth2 token.
	rawIdToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok || strings.TrimSpace(rawIdToken) == "" {
		return nil, fmt.Errorf("id_token missing")
	}

	// Parse and verify ID Token payload.
	idToken, err := p.verifier.Verify(context.Background(), rawIdToken)
	if err != nil {
		return nil, err
	}

	sub, tokenClaims, err := p.getTokenClaims(idToken)
	if err != nil {
		return nil, err
	}

	userInfoClaims, err := p.getUserInfoClaims(oauth2Config, oauth2Token)
	if err != nil {
		return nil, err
	}

	var email string
	if v, ok := userInfoClaims["email"]; ok {
		email, ok = v.(string)
		if !ok {
			return nil, fmt.Errorf("failed to convert email into string: %v", email)
		}
	}

	if email == "" {
		return nil, errors.New("userinfo does not contain email, please check scopes")
	}

	domain := strings.Split(email, "@")[1]

	return &User{
		ID:   sub,
		Name: email,
		Attr: map[string]interface{}{
			"email":    email,
			"domain":   domain,
			"token":    tokenClaims,
			"userinfo": userInfoClaims,
		},
	}, nil
}

func (p *OIDCProvider) getTokenClaims(idToken *oidc.IDToken) (string, map[string]interface{}, error) {
	var raw = make(map[string]interface{})
	var claims struct {
		Sub string `json:"sub"`
	}

	// Extract default claims.
	if err := idToken.Claims(&claims); err != nil {
		return "", nil, fmt.Errorf("failed to parse id_token claims: %v", err)
	}

	// Extract raw claims.
	if err := idToken.Claims(&raw); err != nil {
		return "", nil, fmt.Errorf("failed to parse id_token claims: %v", err)
	}

	return claims.Sub, raw, nil
}

func (p *OIDCProvider) getUserInfoClaims(config oauth2.Config, token *oauth2.Token) (map[string]interface{}, error) {
	var raw = make(map[string]interface{})

	source := config.TokenSource(context.Background(), token)

	info, err := p.provider.UserInfo(context.Background(), source)
	if err != nil {
		return nil, err
	}

	if err := info.Claims(&raw); err != nil {
		return nil, fmt.Errorf("failed to parse user info claims: %v", err)
	}

	return raw, nil
}
