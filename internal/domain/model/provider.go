package model

// Issuer represents the URL that uniquely identifies an OpenID Provider.
type Issuer string

// NewIssuer creates an Issuer value object.
func NewIssuer(value string) Issuer { return Issuer(value) }

// String returns the issuer URL string.
func (i Issuer) String() string { return string(i) }

// AuthMethod represents the client authentication method for the token endpoint.
type AuthMethod string

const (
	AuthMethodBasic AuthMethod = "client_secret_basic"
	AuthMethodPost  AuthMethod = "client_secret_post"
)

// Provider holds the configuration for an OpenID Provider.
type Provider struct {
	id         string
	issuer     Issuer
	client     Client
	scopes     []string
	authMethod AuthMethod
}

// NewProvider creates a new Provider.
func NewProvider(id string, issuer Issuer, client Client, scopes []string, authMethod AuthMethod) Provider {
	return Provider{
		id:         id,
		issuer:     issuer,
		client:     client,
		scopes:     scopes,
		authMethod: authMethod,
	}
}

func (p Provider) Id() string             { return p.id }
func (p Provider) Issuer() Issuer         { return p.issuer }
func (p Provider) Client() Client         { return p.client }
func (p Provider) Scopes() []string       { return p.scopes }
func (p Provider) AuthMethod() AuthMethod { return p.authMethod }
