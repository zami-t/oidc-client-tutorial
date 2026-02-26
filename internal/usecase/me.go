package usecase

import (
	"context"

	"oidc-tutorial/internal/domain/port"
	"oidc-tutorial/internal/usecase/dto"
)

// MeUsecase returns the logged-in user's information from their session.
type MeUsecase struct {
	sessionRepo port.SessionRepository
}

// NewMeUsecase creates a new MeUsecase.
func NewMeUsecase(sessionRepo port.SessionRepository) *MeUsecase {
	return &MeUsecase{sessionRepo: sessionRepo}
}

// Execute returns user information for the given session.
func (u *MeUsecase) Execute(ctx context.Context, input dto.MeInput) (dto.MeOutput, error) {
	session, err := u.sessionRepo.FindById(ctx, input.SessionId)
	if err != nil {
		return dto.MeOutput{}, err
	}
	return dto.MeOutput{
		Subject: session.Subject(),
		Issuer:  session.Issuer(),
		Email:   session.Email(),
		Name:    session.Name(),
		Picture: session.Picture(),
	}, nil
}
