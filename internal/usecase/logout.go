package usecase

import (
	"context"

	"oidc-tutorial/internal/domain/port"
	"oidc-tutorial/internal/usecase/dto"
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
	// Verify the session exists before deleting (returns SESSION_NOT_FOUND if absent)
	if _, err := u.sessionRepo.FindById(ctx, input.SessionId); err != nil {
		return err
	}
	return u.sessionRepo.Delete(ctx, input.SessionId)
}
