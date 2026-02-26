package model

// AuthMethod represents the client authentication method for the token endpoint.
type AuthMethod string

const (
	AuthMethodBasic AuthMethod = "client_secret_basic"
	AuthMethodPost  AuthMethod = "client_secret_post"
)

// Provider holds the configuration for an OpenID Provider.
type Provider struct {
	id           string
	issuer       string
	clientId     string
	clientSecret string
	redirectUri  string
	scopes       []string
	authMethod   AuthMethod
}

// NewProvider creates a new Provider.
func NewProvider(id, issuer, clientId, clientSecret, redirectUri string, scopes []string, authMethod AuthMethod) Provider {
	return Provider{
		id:           id,
		issuer:       issuer,
		clientId:     clientId,
		clientSecret: clientSecret,
		redirectUri:  redirectUri,
		scopes:       scopes,
		authMethod:   authMethod,
	}
}

func (p Provider) Id() string           { return p.id }
func (p Provider) Issuer() string       { return p.issuer }
func (p Provider) ClientId() string     { return p.clientId }
func (p Provider) ClientSecret() string { return p.clientSecret }
func (p Provider) RedirectUri() string  { return p.redirectUri }
func (p Provider) Scopes() []string     { return p.scopes }
func (p Provider) AuthMethod() AuthMethod { return p.authMethod }
