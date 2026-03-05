package usecase

import (
	"context"
	"errors"
	"fmt"

	"oidc-tutorial/internal/domain/port"
	"oidc-tutorial/internal/usecase/dto"
)

// Sentinel errors for LogoutUsecase.
var (
	ErrLogoutSessionNotFound = errors.New("session not found")
)

// LogoutUsecase handles session termination.
type LogoutUsecase struct {
	sessionRepo port.SessionRepository
}

// NewLogoutUsecase creates a new LogoutUsecase.
func NewLogoutUsecase(sessionRepo port.SessionRepository) *LogoutUsecase {
	return &LogoutUsecase{sessionRepo: sessionRepo}
}

// Execute deletes the session identified by the session ID.
func (u *LogoutUsecase) Execute(ctx context.Context, input dto.LogoutInput) error {
	if _, err := u.sessionRepo.FindById(ctx, input.SessionId); err != nil {
		if errors.Is(err, port.ErrSessionNotFound) {
			return fmt.Errorf("session not found: %w", ErrLogoutSessionNotFound)
		}
		return fmt.Errorf("failed to lookup session: %w", err)
	}
	if err := u.sessionRepo.Delete(ctx, input.SessionId); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}
