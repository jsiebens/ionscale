package service

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/key"
	"github.com/jsiebens/ionscale/internal/token"
	"go.uber.org/zap"
	"strings"
)

var (
	errInvalidToken = connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token"))
)

const (
	principalKey = "principalKay"
)

func CurrentPrincipal(ctx context.Context) domain.Principal {
	p := ctx.Value(principalKey)
	if p == nil {
		return domain.Principal{SystemRole: domain.SystemRoleNone, UserRole: domain.UserRoleNone}
	}
	return p.(domain.Principal)
}

func AuthenticationInterceptor(systemAdminKey *key.ServerPrivate, repository domain.Repository) connect.UnaryInterceptorFunc {
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

func exchangeToken(ctx context.Context, systemAdminKey *key.ServerPrivate, repository domain.Repository, value string) *domain.Principal {
	if len(value) == 0 {
		return nil
	}

	if systemAdminKey != nil && token.IsSystemAdminToken(value) {
		_, err := token.ParseSystemAdminToken(*systemAdminKey, value)
		if err == nil {
			return &domain.Principal{SystemRole: domain.SystemRoleAdmin}
		}
	}

	apiKey, err := repository.LoadApiKey(ctx, value)
	if err == nil && apiKey != nil {
		user := apiKey.User
		tailnet := apiKey.Tailnet
		role := tailnet.IAMPolicy.Get().GetRole(user)

		return &domain.Principal{User: &apiKey.User, SystemRole: domain.SystemRoleNone, UserRole: role}
	}

	systemApiKey, err := repository.LoadSystemApiKey(ctx, value)
	if err == nil && systemApiKey != nil {
		return &domain.Principal{SystemRole: domain.SystemRoleAdmin}
	}

	return nil
}

func NewErrorInterceptor() *ErrorInterceptor {
	return &ErrorInterceptor{}
}

type ErrorInterceptor struct {
}

func (e *ErrorInterceptor) handleError(err error) error {
	if err == nil {
		return err
	}

	switch err.(type) {
	case *connect.Error:
		return err
	default:
		return connect.NewError(connect.CodeInternal, fmt.Errorf("internal server error"))
	}
}

func (e *ErrorInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
		response, err := next(ctx, request)
		return response, e.handleError(err)
	}
}

func (e *ErrorInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}

func (e *ErrorInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		err := next(ctx, conn)
		return e.handleError(err)
	}
}

func logError(err error) error {
	zap.L().WithOptions(zap.AddCallerSkip(1)).Error("error processing request", zap.Error(err))
	return err
}
