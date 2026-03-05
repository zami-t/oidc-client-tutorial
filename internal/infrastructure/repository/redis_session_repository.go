package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/port"
)

const sessionKeyPrefix = "oidc:sess:"

type redisSessionRepository struct {
	client *redis.Client
}

// NewRedisSessionRepository creates a Redis-backed SessionRepository.
func NewRedisSessionRepository(client *redis.Client) port.SessionRepository {
	return &redisSessionRepository{client: client}
}

func (r *redisSessionRepository) Save(ctx context.Context, session model.AppSession, ttl time.Duration) error {
	key := sessionKeyPrefix + session.Id()
	fields := map[string]any{
		"subject": session.Subject(),
		"issuer":  session.Issuer(),
		"email":   session.Email(),
		"name":    session.Name(),
		"picture": session.Picture(),
	}
	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to save session to redis: %w", err)
	}
	return nil
}

func (r *redisSessionRepository) FindById(ctx context.Context, sessionId string) (model.AppSession, error) {
	key := sessionKeyPrefix + sessionId
	fields, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return model.AppSession{}, fmt.Errorf("failed to get session from redis: %w", err)
	}
	if len(fields) == 0 {
		return model.AppSession{}, fmt.Errorf("session %q: %w", sessionId, port.ErrSessionNotFound)
	}
	return model.NewAppSession(
		sessionId,
		fields["subject"],
		fields["issuer"],
		fields["email"],
		fields["name"],
		fields["picture"],
	), nil
}

func (r *redisSessionRepository) Delete(ctx context.Context, id string) error {
	key := sessionKeyPrefix + id
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session from redis: %w", err)
	}
	return nil
}
