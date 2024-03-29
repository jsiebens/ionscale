package server

import (
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/key"
	"github.com/jsiebens/ionscale/internal/service"
	apiconnect "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1/ionscalev1connect"
	"net/http"
)

func NewRpcHandler(systemAdminKey *key.ServerPrivate, repository domain.Repository, handler apiconnect.IonscaleServiceHandler) (string, http.Handler) {
	interceptors := connect.WithInterceptors(service.NewErrorInterceptor(), service.AuthenticationInterceptor(systemAdminKey, repository))
	return apiconnect.NewIonscaleServiceHandler(handler, interceptors)
}
