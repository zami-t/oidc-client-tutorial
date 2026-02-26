package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/port"
	"oidc-tutorial/internal/domain/service"
	"oidc-tutorial/internal/usecase/dto"
)

// CallbackUsecase handles the /callback endpoint: validates state, exchanges the
// authorization code for tokens, verifies the ID token, and creates a session.
type CallbackUsecase struct {
	providers       map[string]model.Provider
	transactionRepo port.TransactionRepository
	sessionRepo     port.SessionRepository
	discoveryClient port.DiscoveryClient
	tokenClient     port.TokenClient
	tokenVerifier   *service.IdTokenVerifier
	randomGen       service.RandomGenerator
	sessionTtl      time.Duration
}

// NewCallbackUsecase creates a new CallbackUsecase.
func NewCallbackUsecase(
	providers map[string]model.Provider,
	transactionRepo port.TransactionRepository,
	sessionRepo port.SessionRepository,
	discoveryClient port.DiscoveryClient,
	tokenClient port.TokenClient,
	tokenVerifier *service.IdTokenVerifier,
	randomGen service.RandomGenerator,
	sessionTtl time.Duration,
) *CallbackUsecase {
	return &CallbackUsecase{
		providers:       providers,
		transactionRepo: transactionRepo,
		sessionRepo:     sessionRepo,
		discoveryClient: discoveryClient,
		tokenClient:     tokenClient,
		tokenVerifier:   tokenVerifier,
		randomGen:       randomGen,
		sessionTtl:      sessionTtl,
	}
}

// Execute processes the callback, returning the session ID and post-login redirect URL.
func (u *CallbackUsecase) Execute(ctx context.Context, input dto.CallbackInput) (dto.CallbackOutput, error) {
	// Step 3.1: Handle OP-side authorization errors
	if input.Error != "" {
		return dto.CallbackOutput{}, model.NewAppError(
			model.ErrCodeAuthorizationError,
			fmt.Sprintf("authorization error from OP: %s", input.Error),
			nil,
		)
	}
	if input.State == "" {
		return dto.CallbackOutput{}, model.NewAppError(model.ErrCodeStateMismatch, "state parameter is missing", nil)
	}
	if input.Code == "" {
		return dto.CallbackOutput{}, model.NewAppError(model.ErrCodeStateMismatch, "code parameter is missing", nil)
	}

	// Look up the transaction by state (validates CSRF)
	tx, err := u.transactionRepo.FindByState(ctx, input.State)
	if err != nil {
		return dto.CallbackOutput{}, err // already an *AppError
	}

	// Delete the transaction immediately — one-time use (replay prevention)
	if err := u.transactionRepo.Delete(ctx, input.State); err != nil {
		return dto.CallbackOutput{}, model.NewAppError(model.ErrCodeServerError, "failed to delete transaction", err)
	}

	provider, ok := u.providers[tx.Idp()]
	if !ok {
		return dto.CallbackOutput{}, model.NewAppError(
			model.ErrCodeServerError,
			fmt.Sprintf("unknown IdP in transaction: %q", tx.Idp()),
			nil,
		)
	}

	metadata, err := u.discoveryClient.GetProviderMetadata(ctx, provider.Issuer())
	if err != nil {
		return dto.CallbackOutput{}, model.NewAppError(model.ErrCodeServerError, "failed to get provider metadata", err)
	}

	// Step 3.2: Exchange authorization code for tokens
	tokenResp, err := u.tokenClient.Exchange(ctx, port.TokenExchangeRequest{
		TokenEndpoint: metadata.TokenEndpoint(),
		Code:          input.Code,
		RedirectUri:   provider.RedirectUri(),
		Provider:      provider,
	})
	if err != nil {
		return dto.CallbackOutput{}, model.NewAppError(model.ErrCodeTokenVerificationFailed, "token exchange failed", err)
	}

	// Validate token_type (RFC 6750)
	if !strings.EqualFold(tokenResp.TokenType, "Bearer") {
		return dto.CallbackOutput{}, model.NewAppError(
			model.ErrCodeTokenVerificationFailed,
			fmt.Sprintf("unsupported token type: %q", tokenResp.TokenType),
			nil,
		)
	}

	// Step 4: Verify the ID token
	claims, err := u.tokenVerifier.Verify(ctx, tokenResp.IdToken, tx.Nonce(), provider.ClientId(), provider.Issuer())
	if err != nil {
		return dto.CallbackOutput{}, model.NewAppError(model.ErrCodeTokenVerificationFailed, "ID token verification failed", err)
	}

	// Create session
	sessionId, err := u.randomGen.Generate(32) // 256 bits
	if err != nil {
		return dto.CallbackOutput{}, model.NewAppError(model.ErrCodeServerError, "failed to generate session ID", err)
	}

	session := model.NewAppSession(
		sessionId,
		claims.Subject,
		claims.Issuer,
		claims.Email,
		claims.Name,
		claims.Picture,
	)
	if err := u.sessionRepo.Save(ctx, session, u.sessionTtl); err != nil {
		return dto.CallbackOutput{}, model.NewAppError(model.ErrCodeServerError, "failed to save session", err)
	}

	return dto.CallbackOutput{
		ReturnTo:  tx.ReturnTo(),
		SessionId: sessionId,
	}, nil
}
