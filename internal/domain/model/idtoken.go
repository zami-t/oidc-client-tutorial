package model

// IdTokenClaims holds the verified claims parsed from an ID token.
type IdTokenClaims struct {
	Issuer          string
	Subject         string
	Audience        []string
	AuthorizedParty string
	ExpiresAt       int64
	IssuedAt        int64
	Nonce           string
	Email           string
	Name            string
	Picture         string
}
