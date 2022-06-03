package server

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/key"
	"github.com/jsiebens/ionscale/internal/token"
	apiconnect "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1/ionscalev1connect"
	"net/http"
	"strings"
)

var (
	errInvalidToken = connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token"))
)

func NewRpcHandler(systemAdminKey key.ServerPrivate, handler apiconnect.IonscaleServiceHandler) (string, http.Handler) {
	interceptors := connect.WithInterceptors(authenticationInterceptor(systemAdminKey))
	return apiconnect.NewIonscaleServiceHandler(handler, interceptors)
}

func authenticationInterceptor(systemAdminKey key.ServerPrivate) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			name := req.Spec().Procedure

			if strings.HasSuffix(name, "/GetVersion") {
				return next(ctx, req)
			}

			authorizationHeader := req.Header().Get("Authorization")

			valid := validateAuthorizationToken(systemAdminKey, authorizationHeader)

			if valid {
				return next(ctx, req)
			}

			return nil, errInvalidToken
		}
	}
}

func validateAuthorizationToken(systemAdminKey key.ServerPrivate, authorization string) bool {
	if len(authorization) == 0 {
		return false
	}

	bearerToken := strings.TrimPrefix(authorization, "Bearer ")

	if token.IsSystemAdminToken(bearerToken) {
		_, err := token.ParseSystemAdminToken(systemAdminKey, bearerToken)
		return err == nil
	}

	return false
}
