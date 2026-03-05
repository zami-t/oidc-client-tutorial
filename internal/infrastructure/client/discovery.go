package client

import (
	"context"

	"oidc-tutorial/internal/domain/model"
)

// DiscoveryClient fetches data directly from an OpenID Provider without caching.
type DiscoveryClient interface {
	GetProviderMetadata(ctx context.Context, issuer string) (model.ProviderMetadata, error)
	GetJwks(ctx context.Context, jwksUri string) (model.JwkSet, error)
}
