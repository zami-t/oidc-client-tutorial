package port

import (
	"context"
	"time"

	"oidc-tutorial/internal/domain/model"
)

// SessionRepository manages application sessions (oidc:sess:{session_id}).
type SessionRepository interface {
	Save(ctx context.Context, session model.AppSession, ttl time.Duration) error
	FindById(ctx context.Context, id string) (model.AppSession, error)
	Delete(ctx context.Context, id string) error
}
