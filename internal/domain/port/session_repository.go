package port

import (
	"context"
	"errors"
	"time"

	"oidc-tutorial/internal/domain/model"
)

// ErrSessionNotFound is returned by FindById when the session does not exist
// or has expired.
var ErrSessionNotFound = errors.New("session not found")

// SessionRepository manages application sessions (oidc:sess:{session_id}).
type SessionRepository interface {
	Save(ctx context.Context, session model.AppSession, ttl time.Duration) error
	FindById(ctx context.Context, id string) (model.AppSession, error)
	Delete(ctx context.Context, id string) error
}
