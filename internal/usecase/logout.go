package usecase

import (
	"context"
	"errors"
	"fmt"

	"oidc-tutorial/internal/domain/port"
	"oidc-tutorial/internal/logger"
	"oidc-tutorial/internal/usecase/dto"
)

// Sentinel errors for LogoutUsecase.
var (
	ErrLogoutSessionNotFound = errors.New("session not found")
)

// LogoutUsecase handles session termination.
type LogoutUsecase struct {
	sessionRepo port.SessionRepository
	log         *logger.Logger
}

// NewLogoutUsecase creates a new LogoutUsecase.
func NewLogoutUsecase(sessionRepo port.SessionRepository, log *logger.Logger) *LogoutUsecase {
	return &LogoutUsecase{sessionRepo: sessionRepo, log: log}
}

// Execute deletes the session identified by the session ID.
func (u *LogoutUsecase) Execute(ctx context.Context, input dto.LogoutInput) error {
	if _, err := u.sessionRepo.FindById(ctx, input.SessionId); err != nil {
		if errors.Is(err, port.ErrSessionNotFound) {
			wrapped := fmt.Errorf("session not found: %w", ErrLogoutSessionNotFound)
			u.log.Info(ctx, "logout: session not found")
			return wrapped
		}
		wrapped := fmt.Errorf("failed to lookup session: %w", err)
		u.log.Error(ctx, "logout: failed to lookup session", "LOGOUT_SESSION_LOOKUP_FAILED", wrapped)
		return wrapped
	}
	if err := u.sessionRepo.Delete(ctx, input.SessionId); err != nil {
		wrapped := fmt.Errorf("failed to delete session: %w", err)
		u.log.Error(ctx, "logout: failed to delete session", "LOGOUT_SESSION_DELETE_FAILED", wrapped)
		return wrapped
	}
	u.log.Info(ctx, "logout: session deleted")
	return nil
}
