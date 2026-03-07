package client

import (
	"context"
	"errors"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/port"
)

type orchestrateDiscoveryClient struct {
	discoveryClient      DiscoveryClient
	discoveryCacheClient DiscoveryCacheClient
}

// NewOrchestrateDiscoveryClient creates a DiscoveryClient that orchestrates DiscoveryClient and DiscoveryCacheClient.
func NewOrchestrateDiscoveryClient(fetcher DiscoveryClient, cache DiscoveryCacheClient) port.DiscoveryClient {
	return &orchestrateDiscoveryClient{
		discoveryClient:      fetcher,
		discoveryCacheClient: cache,
	}
}

func (c *orchestrateDiscoveryClient) GetProviderMetadata(ctx context.Context, issuer model.Issuer) (model.ProviderMetadata, error) {
	metadata, err := c.discoveryCacheClient.GetProviderMetadata(ctx, issuer)
	if errors.Is(err, ErrCacheMiss) {
		return c.fetchAndCacheMetadata(ctx, issuer)
	}
	if err != nil {
		return model.ProviderMetadata{}, err
	}
	return metadata, nil
}

func (c *orchestrateDiscoveryClient) fetchAndCacheMetadata(ctx context.Context, issuer model.Issuer) (model.ProviderMetadata, error) {
	metadata, err := c.discoveryClient.GetProviderMetadata(ctx, issuer)
	if err != nil {
		return model.ProviderMetadata{}, err
	}
	_ = c.discoveryCacheClient.SetProviderMetadata(ctx, issuer, metadata)
	return metadata, nil
}

func (c *orchestrateDiscoveryClient) GetJwks(ctx context.Context, issuer model.Issuer) (model.JwkSet, error) {
	jwks, err := c.discoveryCacheClient.GetJwks(ctx, issuer)
	if errors.Is(err, ErrCacheMiss) {
		return c.fetchAndCacheJwks(ctx, issuer)
	}
	if err != nil {
		return model.JwkSet{}, err
	}
	return jwks, nil
}

func (c *orchestrateDiscoveryClient) fetchAndCacheJwks(ctx context.Context, issuer model.Issuer) (model.JwkSet, error) {
	metadata, err := c.GetProviderMetadata(ctx, issuer)
	if err != nil {
		return model.JwkSet{}, err
	}
	jwks, err := c.discoveryClient.GetJwks(ctx, metadata.JwksUri())
	if err != nil {
		return model.JwkSet{}, err
	}
	_ = c.discoveryCacheClient.SetJwks(ctx, issuer, jwks)
	return jwks, nil
}

func (c *orchestrateDiscoveryClient) RefreshJwks(ctx context.Context, issuer model.Issuer, jwksUri string) (model.JwkSet, error) {
	_ = c.discoveryCacheClient.DeleteJwks(ctx, issuer)
	jwks, err := c.discoveryClient.GetJwks(ctx, jwksUri)
	if err != nil {
		return model.JwkSet{}, err
	}
	_ = c.discoveryCacheClient.SetJwks(ctx, issuer, jwks)
	return jwks, nil
}
