package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/infrastructure/client/dto"
)

const (
	metadataCacheKeyPrefix = "oidc:discovery:metadata:"
	jwksCacheKeyPrefix     = "oidc:discovery:jwks:"
)

type redisDiscoveryCacheClient struct {
	client      *redis.Client
	metadataTtl time.Duration
	jwksTtl     time.Duration
}

// NewRedisDiscoveryCacheClient creates a DiscoveryCacheClient backed by Redis.
func NewRedisDiscoveryCacheClient(client *redis.Client, metadataTtl, jwksTtl time.Duration) DiscoveryCacheClient {
	return &redisDiscoveryCacheClient{
		client:      client,
		metadataTtl: metadataTtl,
		jwksTtl:     jwksTtl,
	}
}

func (c *redisDiscoveryCacheClient) GetProviderMetadata(ctx context.Context, issuer model.Issuer) (model.ProviderMetadata, error) {
	data, err := c.client.Get(ctx, metadataCacheKeyPrefix+issuer.String()).Bytes()
	if err == redis.Nil {
		return model.ProviderMetadata{}, ErrCacheMiss
	}
	if err != nil {
		return model.ProviderMetadata{}, fmt.Errorf("failed to get metadata from redis: %w", err)
	}
	var raw dto.DiscoveryResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return model.ProviderMetadata{}, fmt.Errorf("failed to unmarshal metadata: %w", err)
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

func (c *redisDiscoveryCacheClient) SaveProviderMetadata(ctx context.Context, issuer model.Issuer, metadata model.ProviderMetadata) error {
	raw := dto.DiscoveryResponse{
		Issuer:                           metadata.Issuer().String(),
		AuthorizationEndpoint:            metadata.AuthorizationEndpoint(),
		TokenEndpoint:                    metadata.TokenEndpoint(),
		UserinfoEndpoint:                 metadata.UserinfoEndpoint(),
		JwksUri:                          metadata.JwksUri(),
		ResponseTypesSupported:           metadata.ResponseTypesSupported(),
		SubjectTypesSupported:            metadata.SubjectTypesSupported(),
		IdTokenSigningAlgValuesSupported: metadata.IdTokenSigningAlgValuesSupported(),
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := c.client.Set(ctx, metadataCacheKeyPrefix+issuer.String(), data, c.metadataTtl).Err(); err != nil {
		return fmt.Errorf("failed to set metadata in redis: %w", err)
	}
	return nil
}

func (c *redisDiscoveryCacheClient) GetJwks(ctx context.Context, issuer model.Issuer) (model.JwkSet, error) {
	data, err := c.client.Get(ctx, jwksCacheKeyPrefix+issuer.String()).Bytes()
	if err == redis.Nil {
		return model.JwkSet{}, ErrCacheMiss
	}
	if err != nil {
		return model.JwkSet{}, fmt.Errorf("failed to get jwks from redis: %w", err)
	}
	var raw dto.JwksDto
	if err := json.Unmarshal(data, &raw); err != nil {
		return model.JwkSet{}, fmt.Errorf("failed to unmarshal jwks: %w", err)
	}
	keys := make([]model.Jwk, 0, len(raw.Keys))
	for _, k := range raw.Keys {
		keys = append(keys, model.Jwk{
			Kty: k.Kty, Kid: k.Kid, Alg: k.Alg, Use: k.Use, N: k.N, E: k.E,
		})
	}
	return model.NewJwkSet(keys), nil
}

func (c *redisDiscoveryCacheClient) SaveJwks(ctx context.Context, issuer model.Issuer, jwks model.JwkSet) error {
	keys := make([]dto.JwkDto, 0, len(jwks.Keys()))
	for _, k := range jwks.Keys() {
		keys = append(keys, dto.JwkDto{
			Kty: k.Kty, Kid: k.Kid, Alg: k.Alg, Use: k.Use, N: k.N, E: k.E,
		})
	}
	data, err := json.Marshal(dto.JwksDto{Keys: keys})
	if err != nil {
		return fmt.Errorf("failed to marshal jwks: %w", err)
	}
	if err := c.client.Set(ctx, jwksCacheKeyPrefix+issuer.String(), data, c.jwksTtl).Err(); err != nil {
		return fmt.Errorf("failed to set jwks in redis: %w", err)
	}
	return nil
}

func (c *redisDiscoveryCacheClient) DeleteJwks(ctx context.Context, issuer model.Issuer) error {
	if err := c.client.Del(ctx, jwksCacheKeyPrefix+issuer.String()).Err(); err != nil {
		return fmt.Errorf("failed to delete jwks from redis: %w", err)
	}
	return nil
}
