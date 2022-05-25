package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jsiebens/ionscale/internal/domain"
	"golang.org/x/oauth2"
)

type OIDCProvider struct {
	clientID     string
	clientSecret string
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
}

func NewOIDCProvider(c *domain.AuthMethod) (*OIDCProvider, error) {
	provider, err := oidc.NewProvider(context.Background(), c.Issuer)
	if err != nil {
		return nil, err
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: c.ClientId})

	return &OIDCProvider{
		clientID:     c.ClientId,
		clientSecret: c.ClientSecret,
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
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return oauth2Config.AuthCodeURL(state, oauth2.ApprovalForce)
}

func (p *OIDCProvider) Exchange(redirectURI, code string) (*User, error) {
	oauth2Config := oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     p.provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
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

	sub, email, tokenClaims, err := p.getTokenClaims(idToken)
	if err != nil {
		return nil, err
	}

	userInfoClaims, err := p.getUserInfoClaims(oauth2Config, oauth2Token)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:   sub,
		Name: email,
		Attr: map[string]interface{}{
			"token":    tokenClaims,
			"userinfo": userInfoClaims,
		},
	}, nil
}

func (p *OIDCProvider) getTokenClaims(idToken *oidc.IDToken) (string, string, map[string]interface{}, error) {
	var raw = make(map[string]interface{})
	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
	}

	// Extract default claims.
	if err := idToken.Claims(&claims); err != nil {
		return "", "", nil, fmt.Errorf("failed to parse id_token claims: %v", err)
	}

	// Extract raw claims.
	if err := idToken.Claims(&raw); err != nil {
		return "", "", nil, fmt.Errorf("failed to parse id_token claims: %v", err)
	}

	return claims.Sub, claims.Email, raw, nil
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
