package service

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/key"
	"github.com/jsiebens/ionscale/internal/token"
	"strings"
)

var (
	errInvalidToken = connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token"))
)

const (
	principalKey = "principalKay"
)

type Principal struct {
	SystemRole domain.SystemRole
	User       *domain.User
	UserRole   domain.UserRole
}

func (p Principal) IsSystemAdmin() bool {
	return p.SystemRole.IsAdmin()
}

func (p Principal) IsTailnetAdmin(tailnetID uint64) bool {
	return p.User.TailnetID == tailnetID && p.UserRole.IsAdmin()
}

func (p Principal) IsTailnetMember(tailnetID uint64) bool {
	return p.User.TailnetID == tailnetID
}

func (p Principal) UserMatches(userID uint64) bool {
	return p.User.ID == userID
}

func CurrentPrincipal(ctx context.Context) Principal {
	p := ctx.Value(principalKey)
	if p == nil {
		return Principal{SystemRole: domain.SystemRoleNone, UserRole: domain.UserRoleNone}
	}
	return p.(Principal)
}

func AuthenticationInterceptor(systemAdminKey key.ServerPrivate, repository domain.Repository) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			name := req.Spec().Procedure

			if strings.HasSuffix(name, "/GetVersion") {
				return next(ctx, req)
			}

			authorizationHeader := req.Header().Get("Authorization")
			bearerToken := strings.TrimPrefix(authorizationHeader, "Bearer ")

			if principal := exchangeToken(ctx, systemAdminKey, repository, bearerToken); principal != nil {
				return next(context.WithValue(ctx, principalKey, *principal), req)
			}

			return nil, errInvalidToken
		}
	}
}

func exchangeToken(ctx context.Context, systemAdminKey key.ServerPrivate, repository domain.Repository, value string) *Principal {
	if len(value) == 0 {
		return nil
	}

	if token.IsSystemAdminToken(value) {
		_, err := token.ParseSystemAdminToken(systemAdminKey, value)
		if err == nil {
			return &Principal{SystemRole: domain.SystemRoleAdmin}
		}
	}

	apiKey, err := repository.LoadApiKey(ctx, value)
	if err == nil && apiKey != nil {
		user := apiKey.User
		tailnet := apiKey.Tailnet
		role := tailnet.IAMPolicy.GetRole(user)

		return &Principal{User: &apiKey.User, SystemRole: domain.SystemRoleNone, UserRole: role}
	}

	systemApiKey, err := repository.LoadSystemApiKey(ctx, value)
	if err == nil && systemApiKey != nil {
		return &Principal{SystemRole: domain.SystemRoleAdmin}
	}

	return nil
}
