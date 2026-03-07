package service

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/port"
)

// allowedAlgorithms is the whitelist of supported JWS signing algorithms.
var allowedAlgorithms = map[string]bool{
	"RS256": true,
}

// jwtHeader is the decoded JWT header.
type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

// jwtRawClaims holds the raw JSON from the JWT payload.
// Audience is json.RawMessage to handle both string and []string forms (RFC 7519 §4.1.3).
type jwtRawClaims struct {
	Iss     string          `json:"iss"`
	Sub     string          `json:"sub"`
	Aud     json.RawMessage `json:"aud"`
	Azp     string          `json:"azp"`
	Exp     int64           `json:"exp"`
	Iat     int64           `json:"iat"`
	Nonce   string          `json:"nonce"`
	Email   string          `json:"email"`
	Name    string          `json:"name"`
	Picture string          `json:"picture"`
}

// IdTokenVerifier verifies OIDC ID tokens (JWS RS256).
type IdTokenVerifier struct {
	discoveryClient port.DiscoveryClient
}

// NewIdTokenVerifier creates a new IdTokenVerifier.
func NewIdTokenVerifier(discoveryClient port.DiscoveryClient) *IdTokenVerifier {
	return &IdTokenVerifier{discoveryClient: discoveryClient}
}

// Verify parses, verifies, and validates the ID token.
// Returns the claims if all checks pass.
func (v *IdTokenVerifier) Verify(
	ctx context.Context,
	rawToken string,
	expectedNonce string,
	expectedClientId string,
	expectedIssuer model.Issuer,
) (model.IdTokenClaims, error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return model.IdTokenClaims{}, fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	// Step 4.1: Decode and parse header
	headerData, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return model.IdTokenClaims{}, fmt.Errorf("failed to decode JWT header: %w", err)
	}
	var header jwtHeader
	if err := json.Unmarshal(headerData, &header); err != nil {
		return model.IdTokenClaims{}, fmt.Errorf("failed to parse JWT header: %w", err)
	}

	// Reject unsupported algorithms (whitelist check)
	if !allowedAlgorithms[header.Alg] {
		return model.IdTokenClaims{}, fmt.Errorf("unsupported signing algorithm: %s", header.Alg)
	}

	// Step 4.2: Get JWKS (cached)
	jwks, err := v.discoveryClient.GetJwks(ctx, expectedIssuer)
	if err != nil {
		return model.IdTokenClaims{}, fmt.Errorf("failed to get JWKS: %w", err)
	}

	// Find the JWK matching the token's kid
	jwk, found := jwks.FindByKid(header.Kid)
	if !found {
		// Unknown kid: may be due to key rotation — force refresh
		metadata, err := v.discoveryClient.GetProviderMetadata(ctx, expectedIssuer)
		if err != nil {
			return model.IdTokenClaims{}, fmt.Errorf("failed to get provider metadata for JWKS refresh: %w", err)
		}
		jwks, err = v.discoveryClient.RefreshJwks(ctx, expectedIssuer, metadata.JwksUri())
		if err != nil {
			return model.IdTokenClaims{}, fmt.Errorf("failed to refresh JWKS: %w", err)
		}
		jwk, found = jwks.FindByKid(header.Kid)
		if !found {
			return model.IdTokenClaims{}, fmt.Errorf("unknown key ID: %s", header.Kid)
		}
	}

	// Step 4.3: Verify RS256 signature over "header.payload"
	signingInput := parts[0] + "." + parts[1]
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return model.IdTokenClaims{}, fmt.Errorf("failed to decode JWT signature: %w", err)
	}
	if err := verifyRS256(jwk, signingInput, sigBytes); err != nil {
		return model.IdTokenClaims{}, fmt.Errorf("signature verification failed: %w", err)
	}

	// Decode and parse payload
	payloadData, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return model.IdTokenClaims{}, fmt.Errorf("failed to decode JWT payload: %w", err)
	}
	var rawClaims jwtRawClaims
	if err := json.Unmarshal(payloadData, &rawClaims); err != nil {
		return model.IdTokenClaims{}, fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	audience, err := parseAudience(rawClaims.Aud)
	if err != nil {
		return model.IdTokenClaims{}, fmt.Errorf("failed to parse aud claim: %w", err)
	}

	claims := model.NewIdTokenClaims(
		rawClaims.Iss,
		rawClaims.Sub,
		audience,
		rawClaims.Azp,
		rawClaims.Exp,
		rawClaims.Iat,
		rawClaims.Nonce,
		rawClaims.Email,
		rawClaims.Name,
		rawClaims.Picture,
	)

	// Step 4.4: Validate all claims
	if err := validateClaims(claims, expectedNonce, expectedClientId, expectedIssuer); err != nil {
		return model.IdTokenClaims{}, err
	}

	return claims, nil
}

func verifyRS256(jwk model.Jwk, signingInput string, signature []byte) error {
	pubKey, err := jwk.ToRsaPublicKey()
	if err != nil {
		return fmt.Errorf("failed to construct RSA public key: %w", err)
	}
	hash := sha256.Sum256([]byte(signingInput))
	return rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], signature)
}

// validateClaims validates all required ID token claims per OIDC Core 3.1.3.7 and google-idp.md.
func validateClaims(claims model.IdTokenClaims, expectedNonce, expectedClientId string, expectedIssuer model.Issuer) error {
	now := time.Now()

	// iss: must match issuer from discovery (accept both https://issuer and issuer)
	if claims.Issuer() != expectedIssuer.String() {
		altIssuer := strings.TrimPrefix(expectedIssuer.String(), "https://")
		if claims.Issuer() != altIssuer {
			return fmt.Errorf("iss mismatch: got %q, expected %q", claims.Issuer(), expectedIssuer)
		}
	}

	// aud: must contain client_id
	audContainsClientId := false
	for _, a := range claims.Audience() {
		if a == expectedClientId {
			audContainsClientId = true
			break
		}
	}
	if !audContainsClientId {
		return fmt.Errorf("aud claim does not contain client_id %q", expectedClientId)
	}

	// azp: if aud has multiple values, azp must equal client_id
	if len(claims.Audience()) > 1 && claims.AuthorizedParty() != expectedClientId {
		return fmt.Errorf("azp mismatch: got %q, expected %q", claims.AuthorizedParty(), expectedClientId)
	}

	// exp: must not be in the past (allow 5-minute clock skew)
	const clockSkew = 5 * time.Minute
	if time.Unix(claims.ExpiresAt(), 0).Add(clockSkew).Before(now) {
		return fmt.Errorf("ID token has expired (exp=%d)", claims.ExpiresAt())
	}

	// iat: reject tokens issued more than 10 minutes ago (RP policy)
	const maxAge = 10 * time.Minute
	if time.Unix(claims.IssuedAt(), 0).Add(maxAge).Before(now) {
		return fmt.Errorf("ID token is too old (iat=%d)", claims.IssuedAt())
	}

	// nonce: must match the value stored in the transaction
	if claims.Nonce() != expectedNonce {
		return fmt.Errorf("nonce mismatch")
	}

	// sub: must be present
	if claims.Subject() == "" {
		return fmt.Errorf("sub claim is empty")
	}

	return nil
}

// parseAudience handles aud as either a JSON string or a JSON array of strings.
func parseAudience(raw json.RawMessage) ([]string, error) {
	if raw == nil {
		return nil, fmt.Errorf("aud claim is missing")
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return []string{single}, nil
	}
	var multi []string
	if err := json.Unmarshal(raw, &multi); err == nil {
		return multi, nil
	}
	return nil, fmt.Errorf("aud claim has unexpected format")
}
