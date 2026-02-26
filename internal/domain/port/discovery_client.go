package port

import (
	"context"

	"oidc-tutorial/internal/domain/model"
)

// DiscoveryClient fetches and caches OpenID Provider metadata and JWKS.
type DiscoveryClient interface {
	// GetProviderMetadata returns the provider metadata for the given issuer,
	// using the cache if available.
	GetProviderMetadata(ctx context.Context, issuer string) (model.ProviderMetadata, error)

	// GetJwks returns the JWKS for the given issuer, using the cache if available.
	GetJwks(ctx context.Context, issuer string) (model.JwkSet, error)

	// RefreshJwks forces a re-fetch of JWKS from jwksUri and updates the cache.
	// Called when an unknown kid is encountered (key rotation).
	RefreshJwks(ctx context.Context, issuer string, jwksUri string) (model.JwkSet, error)
}
