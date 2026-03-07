package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/infrastructure/client/dto"
)

type httpDiscoveryClient struct {
	httpClient *http.Client
}

// NewHttpDiscoveryClient creates a DiscoveryClient that fetches directly from the OP.
func NewHttpDiscoveryClient(timeout time.Duration) DiscoveryClient {
	return &httpDiscoveryClient{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *httpDiscoveryClient) GetProviderMetadata(ctx context.Context, issuer model.Issuer) (model.ProviderMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, issuer.String()+"/.well-known/openid-configuration", nil)
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
	var raw dto.DiscoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return model.ProviderMetadata{}, fmt.Errorf("failed to decode provider metadata: %w", err)
	}
	return model.NewProviderMetadata(
		model.Issuer(raw.Issuer),
		raw.AuthorizationEndpoint,
		raw.TokenEndpoint,
		raw.UserinfoEndpoint,
		raw.JwksUri,
		raw.ResponseTypesSupported,
		raw.SubjectTypesSupported,
		raw.IdTokenSigningAlgValuesSupported,
	), nil
}

func (c *httpDiscoveryClient) GetJwks(ctx context.Context, jwksUri string) (model.JwkSet, error) {
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
	return model.NewJwkSet(keys), nil
}
