package ionscale

import (
	"context"
	"fmt"
	"github.com/jsiebens/ionscale/internal/token"
	"github.com/jsiebens/ionscale/internal/util"
	"google.golang.org/grpc/credentials"
	"tailscale.com/types/key"
)

func HasCredentials(systemAdminKey string) bool {
	return systemAdminKey != ""
}

func LoadClientAuth(systemAdminKey string) (ClientAuth, error) {
	if systemAdminKey != "" {
		k, err := util.ParseMachinePrivateKey(systemAdminKey)
		if err != nil {
			return nil, fmt.Errorf("invalid system admin key")
		}
		return &systemAdminTokenAuth{key: *k}, nil
	}

	return &anonymous{}, nil
}

type ClientAuth interface {
	credentials.PerRPCCredentials
}

type anonymous struct {
}

func (m *anonymous) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return nil, nil
}

func (m *anonymous) RequireTransportSecurity() bool {
	return false
}

type systemAdminTokenAuth struct {
	key key.MachinePrivate
}

func (m *systemAdminTokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	t, err := token.GenerateSystemAdminToken(m.key)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"authorization": "Bearer " + t,
	}, nil
}

func (m *systemAdminTokenAuth) RequireTransportSecurity() bool {
	return false
}

type tokenAuth struct {
	token string
}

func (m *tokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + m.token,
	}, nil
}

func (m *tokenAuth) RequireTransportSecurity() bool {
	return false
}
