package provider

import (
	"fmt"

	"github.com/jsiebens/ionscale/internal/domain"
)

type AuthProvider interface {
	GetLoginURL(redirectURI, state string) string
	Exchange(redirectURI, code string) (*User, error)
}

type User struct {
	ID   string
	Name string
	Attr map[string]interface{}
}

func NewProvider(m *domain.AuthMethod) (AuthProvider, error) {
	switch m.Type {
	case "oidc":
		return NewOIDCProvider(m)
	default:
		return nil, fmt.Errorf("unknown auth method type")
	}
}
