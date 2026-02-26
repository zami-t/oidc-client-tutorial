package model

import (
	"encoding/base64"
	"fmt"
	"math/big"

	"crypto/rsa"
)

// ProviderMetadata holds OpenID Provider metadata from the discovery endpoint.
type ProviderMetadata struct {
	issuer                           string
	authorizationEndpoint            string
	tokenEndpoint                    string
	userinfoEndpoint                 string
	jwksUri                          string
	responseTypesSupported           []string
	subjectTypesSupported            []string
	idTokenSigningAlgValuesSupported []string
}

// NewProviderMetadata creates a new ProviderMetadata.
func NewProviderMetadata(
	issuer,
	authorizationEndpoint,
	tokenEndpoint,
	userinfoEndpoint,
	jwksUri string,
	responseTypesSupported,
	subjectTypesSupported,
	idTokenSigningAlgValuesSupported []string,
) ProviderMetadata {
	return ProviderMetadata{
		issuer:                           issuer,
		authorizationEndpoint:            authorizationEndpoint,
		tokenEndpoint:                    tokenEndpoint,
		userinfoEndpoint:                 userinfoEndpoint,
		jwksUri:                          jwksUri,
		responseTypesSupported:           responseTypesSupported,
		subjectTypesSupported:            subjectTypesSupported,
		idTokenSigningAlgValuesSupported: idTokenSigningAlgValuesSupported,
	}
}

func (m ProviderMetadata) Issuer() string                             { return m.issuer }
func (m ProviderMetadata) AuthorizationEndpoint() string              { return m.authorizationEndpoint }
func (m ProviderMetadata) TokenEndpoint() string                      { return m.tokenEndpoint }
func (m ProviderMetadata) UserinfoEndpoint() string                   { return m.userinfoEndpoint }
func (m ProviderMetadata) JwksUri() string                            { return m.jwksUri }
func (m ProviderMetadata) ResponseTypesSupported() []string           { return m.responseTypesSupported }
func (m ProviderMetadata) SubjectTypesSupported() []string            { return m.subjectTypesSupported }
func (m ProviderMetadata) IdTokenSigningAlgValuesSupported() []string {
	return m.idTokenSigningAlgValuesSupported
}

// Jwk represents a single JSON Web Key (RSA).
type Jwk struct {
	Kty string
	Kid string
	Alg string
	Use string
	N   string // RSA modulus (base64url, no padding)
	E   string // RSA public exponent (base64url, no padding)
}

// ToRsaPublicKey converts the JWK to an *rsa.PublicKey.
func (k Jwk) ToRsaPublicKey() (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWK modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWK exponent: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())
	return &rsa.PublicKey{N: n, E: e}, nil
}

// JwkSet represents a JSON Web Key Set.
type JwkSet struct {
	keys []Jwk
}

// NewJwkSet creates a new JwkSet.
func NewJwkSet(keys []Jwk) JwkSet {
	return JwkSet{keys: keys}
}

// FindByKid finds a JWK by key ID.
func (s JwkSet) FindByKid(kid string) (Jwk, bool) {
	for _, k := range s.keys {
		if k.Kid == kid {
			return k, true
		}
	}
	return Jwk{}, false
}

// Keys returns all keys in the set.
func (s JwkSet) Keys() []Jwk { return s.keys }
