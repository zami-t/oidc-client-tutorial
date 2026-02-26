package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/port"
	"oidc-tutorial/internal/infrastructure/client/dto"
)

type cachedMetadata struct {
	metadata  model.ProviderMetadata
	expiresAt time.Time
}

type cachedJwks struct {
	jwks      model.JwkSet
	expiresAt time.Time
}

type discoveryClient struct {
	httpClient    *http.Client
	metadataCache sync.Map // map[issuer string]*cachedMetadata
	jwksCache     sync.Map // map[issuer string]*cachedJwks
	metadataTtl   time.Duration
	jwksTtl       time.Duration
}

// NewDiscoveryClient creates a DiscoveryClient that caches metadata and JWKS in memory.
func NewDiscoveryClient(metadataTtl, jwksTtl time.Duration) port.DiscoveryClient {
	return &discoveryClient{
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		metadataTtl: metadataTtl,
		jwksTtl:     jwksTtl,
	}
}

func (c *discoveryClient) GetProviderMetadata(ctx context.Context, issuer string) (model.ProviderMetadata, error) {
	if v, ok := c.metadataCache.Load(issuer); ok {
		entry := v.(*cachedMetadata)
		if time.Now().Before(entry.expiresAt) {
			return entry.metadata, nil
		}
	}
	return c.fetchAndCacheMetadata(ctx, issuer)
}

func (c *discoveryClient) fetchAndCacheMetadata(ctx context.Context, issuer string) (model.ProviderMetadata, error) {
	discoveryUrl := issuer + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryUrl, nil)
	if err != nil {
		return model.ProviderMetadata{}, fmt.Errorf("failed to create discovery request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return model.ProviderMetadata{}, fmt.Errorf("failed to fetch provider metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return model.ProviderMetadata{}, fmt.Errorf("discovery endpoint returned HTTP %d", resp.StatusCode)
	}

	var raw dto.ProviderMetadataDto
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return model.ProviderMetadata{}, fmt.Errorf("failed to decode provider metadata: %w", err)
	}

	metadata := model.NewProviderMetadata(
		raw.Issuer,
		raw.AuthorizationEndpoint,
		raw.TokenEndpoint,
		raw.UserinfoEndpoint,
		raw.JwksUri,
		raw.ResponseTypesSupported,
		raw.SubjectTypesSupported,
		raw.IdTokenSigningAlgValuesSupported,
	)

	c.metadataCache.Store(issuer, &cachedMetadata{
		metadata:  metadata,
		expiresAt: time.Now().Add(c.metadataTtl),
	})
	return metadata, nil
}

func (c *discoveryClient) GetJwks(ctx context.Context, issuer string) (model.JwkSet, error) {
	if v, ok := c.jwksCache.Load(issuer); ok {
		entry := v.(*cachedJwks)
		if time.Now().Before(entry.expiresAt) {
			return entry.jwks, nil
		}
	}
	metadata, err := c.GetProviderMetadata(ctx, issuer)
	if err != nil {
		return model.JwkSet{}, err
	}
	return c.fetchAndCacheJwks(ctx, issuer, metadata.JwksUri())
}

func (c *discoveryClient) RefreshJwks(ctx context.Context, issuer string, jwksUri string) (model.JwkSet, error) {
	c.jwksCache.Delete(issuer)
	return c.fetchAndCacheJwks(ctx, issuer, jwksUri)
}

func (c *discoveryClient) fetchAndCacheJwks(ctx context.Context, issuer, jwksUri string) (model.JwkSet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksUri, nil)
	if err != nil {
		return model.JwkSet{}, fmt.Errorf("failed to create JWKS request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return model.JwkSet{}, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return model.JwkSet{}, fmt.Errorf("JWKS endpoint returned HTTP %d", resp.StatusCode)
	}

	var raw dto.JwksDto
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return model.JwkSet{}, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	keys := make([]model.Jwk, 0, len(raw.Keys))
	for _, k := range raw.Keys {
		keys = append(keys, model.Jwk{
			Kty: k.Kty,
			Kid: k.Kid,
			Alg: k.Alg,
			Use: k.Use,
			N:   k.N,
			E:   k.E,
		})
	}
	jwks := model.NewJwkSet(keys)

	c.jwksCache.Store(issuer, &cachedJwks{
		jwks:      jwks,
		expiresAt: time.Now().Add(c.jwksTtl),
	})
	return jwks, nil
}
