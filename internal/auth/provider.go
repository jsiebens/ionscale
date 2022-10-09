package auth

type Provider interface {
	GetLoginURL(redirectURI, state string) string
	Exchange(redirectURI, code string) (*User, error)
}

type User struct {
	ID   string
	Name string
	Attr map[string]interface{}
}
