package ionscale

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1/ionscalev1connect"
	"net/http"
)

func NewClient(clientAuth ClientAuth, serverURL string, insecureSkipVerify bool) (api.IonscaleServiceClient, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	interceptors := connect.WithInterceptors(NewAuthenticationInterceptor(clientAuth))
	return api.NewIonscaleServiceClient(client, serverURL, interceptors), nil
}

func NewAuthenticationInterceptor(clientAuth ClientAuth) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			token, _ := clientAuth.GetToken()
			req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
			return next(ctx, req)
		}
	}
}
