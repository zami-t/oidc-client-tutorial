package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/port"
)

const txKeyPrefix = "oidc:tx:"

type redisTransactionRepository struct {
	client *redis.Client
}

// NewRedisTransactionRepository creates a Redis-backed TransactionRepository.
func NewRedisTransactionRepository(client *redis.Client) port.TransactionRepository {
	return &redisTransactionRepository{client: client}
}

func (r *redisTransactionRepository) Save(ctx context.Context, tx model.AuthorizationTransaction, ttl time.Duration) error {
	key := txKeyPrefix + tx.State()
	fields := map[string]any{
		"nonce":     tx.Nonce(),
		"return_to": tx.ReturnTo(),
		"idp":       tx.Idp(),
	}
	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to save authorization transaction to redis: %w", err)
	}
	return nil
}

func (r *redisTransactionRepository) FindByState(ctx context.Context, state string) (model.AuthorizationTransaction, error) {
	key := txKeyPrefix + state
	fields, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return model.AuthorizationTransaction{}, fmt.Errorf("failed to get authorization transaction from redis: %w", err)
	}
	if len(fields) == 0 {
		return model.AuthorizationTransaction{}, fmt.Errorf("state %q: %w", state, port.ErrTransactionNotFound)
	}
	return model.NewAuthorizationTransaction(state, fields["nonce"], fields["return_to"], fields["idp"]), nil
}

func (r *redisTransactionRepository) Delete(ctx context.Context, state string) error {
	key := txKeyPrefix + state
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete authorization transaction from redis: %w", err)
	}
	return nil
}
