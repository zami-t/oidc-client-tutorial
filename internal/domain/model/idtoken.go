package model

// IdTokenClaims holds the verified claims parsed from an ID token.
type IdTokenClaims struct {
	issuer          string
	subject         string
	audience        []string
	authorizedParty string
	expiresAt       int64
	issuedAt        int64
	nonce           string
	email           string
	name            string
	picture         string
}

// NewIdTokenClaims creates a new IdTokenClaims.
func NewIdTokenClaims(
	issuer string,
	subject string,
	audience []string,
	authorizedParty string,
	expiresAt int64,
	issuedAt int64,
	nonce string,
	email string,
	name string,
	picture string,
) IdTokenClaims {
	return IdTokenClaims{
		issuer:          issuer,
		subject:         subject,
		audience:        audience,
		authorizedParty: authorizedParty,
		expiresAt:       expiresAt,
		issuedAt:        issuedAt,
		nonce:           nonce,
		email:           email,
		name:            name,
		picture:         picture,
	}
}

func (c IdTokenClaims) Issuer() string          { return c.issuer }
func (c IdTokenClaims) Subject() string         { return c.subject }
func (c IdTokenClaims) Audience() []string      { return c.audience }
func (c IdTokenClaims) AuthorizedParty() string { return c.authorizedParty }
func (c IdTokenClaims) ExpiresAt() int64        { return c.expiresAt }
func (c IdTokenClaims) IssuedAt() int64         { return c.issuedAt }
func (c IdTokenClaims) Nonce() string           { return c.nonce }
func (c IdTokenClaims) Email() string           { return c.email }
func (c IdTokenClaims) Name() string            { return c.name }
func (c IdTokenClaims) Picture() string         { return c.picture }
