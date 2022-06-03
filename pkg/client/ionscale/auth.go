package ionscale

import (
	"fmt"
	"github.com/jsiebens/ionscale/internal/key"
	"github.com/jsiebens/ionscale/internal/token"
)

func LoadClientAuth(systemAdminKey string) (ClientAuth, error) {
	if systemAdminKey != "" {
		k, err := key.ParsePrivateKey(systemAdminKey)
		if err != nil {
			return nil, fmt.Errorf("invalid system admin key")
		}
		return &systemAdminTokenAuth{key: *k}, nil
	}

	return &anonymous{}, nil
}

type ClientAuth interface {
	GetToken() (string, error)
}

type anonymous struct {
}

func (m *anonymous) GetToken() (string, error) {
	return "", nil
}

type systemAdminTokenAuth struct {
	key key.ServerPrivate
}

func (m *systemAdminTokenAuth) GetToken() (string, error) {
	return token.GenerateSystemAdminToken(m.key)
}
