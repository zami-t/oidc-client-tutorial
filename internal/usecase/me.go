package usecase

import (
	"context"
	"errors"
	"fmt"

	"oidc-tutorial/internal/domain/port"
	"oidc-tutorial/internal/logger"
	"oidc-tutorial/internal/usecase/dto"
)

// Sentinel errors for MeUsecase.
var (
	ErrMeSessionNotFound = errors.New("session not found")
)

// MeUsecase returns the logged-in user's information from their session.
type MeUsecase struct {
	sessionRepo port.SessionRepository
	log         *logger.Logger
}

// NewMeUsecase creates a new MeUsecase.
func NewMeUsecase(sessionRepo port.SessionRepository, log *logger.Logger) *MeUsecase {
	return &MeUsecase{sessionRepo: sessionRepo, log: log}
}

// Execute returns user information for the given session.
func (u *MeUsecase) Execute(ctx context.Context, input dto.MeInput) (dto.MeOutput, error) {
	session, err := u.sessionRepo.FindById(ctx, input.SessionId)
	if err != nil {
		if errors.Is(err, port.ErrSessionNotFound) {
			return dto.MeOutput{}, fmt.Errorf("session not found: %w", ErrMeSessionNotFound)
		}
		wrapped := fmt.Errorf("failed to lookup session: %w", err)
		u.log.Error(ctx, "me: failed to lookup session", "ME_SESSION_LOOKUP_FAILED", wrapped)
		return dto.MeOutput{}, wrapped
	}
	return dto.MeOutput{
		Subject: session.Subject(),
		Issuer:  session.Issuer(),
		Email:   session.Email(),
		Name:    session.Name(),
		Picture: session.Picture(),
	}, nil
}
