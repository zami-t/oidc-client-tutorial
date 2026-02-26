package port

import (
	"context"
	"time"

	"oidc-tutorial/internal/domain/model"
)

// TransactionRepository manages authorization transaction state (oidc:tx:{state}).
type TransactionRepository interface {
	Save(ctx context.Context, tx model.AuthorizationTransaction, ttl time.Duration) error
	FindByState(ctx context.Context, state string) (model.AuthorizationTransaction, error)
	Delete(ctx context.Context, state string) error
}
