package client

import (
	"context"
	"errors"

	"oidc-tutorial/internal/domain/model"
)

// ErrCacheMiss is returned when a cache key is not found.
var ErrCacheMiss = errors.New("cache miss")

// DiscoveryCacheClient manages cached OpenID Provider metadata and JWKS.
// ErrCacheMiss is returned on a cache miss; other errors indicate a storage failure.
type DiscoveryCacheClient interface {
	GetProviderMetadata(ctx context.Context, issuer string) (model.ProviderMetadata, error)
	SetProviderMetadata(ctx context.Context, issuer string, metadata model.ProviderMetadata) error
	GetJwks(ctx context.Context, issuer string) (model.JwkSet, error)
	SetJwks(ctx context.Context, issuer string, jwks model.JwkSet) error
	DeleteJwks(ctx context.Context, issuer string) error
}
