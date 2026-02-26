package dto

// ProviderMetadataDto is the raw JSON from the OpenID Connect discovery endpoint.
type ProviderMetadataDto struct {
	Issuer                           string   `json:"issuer"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	UserinfoEndpoint                 string   `json:"userinfo_endpoint"`
	JwksUri                          string   `json:"jwks_uri"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IdTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
}

// JwkDto is the raw JSON for a single JWK from the JWKS endpoint.
type JwkDto struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JwksDto is the raw JSON from the JWKS endpoint.
type JwksDto struct {
	Keys []JwkDto `json:"keys"`
}
