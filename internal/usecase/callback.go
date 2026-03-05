package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/port"
	"oidc-tutorial/internal/domain/service"
	"oidc-tutorial/internal/usecase/dto"
)

// Sentinel errors for CallbackUsecase.
var (
	ErrCallbackAuthorizationError      = errors.New("authorization error from OP")
	ErrCallbackStateMismatch           = errors.New("state mismatch")
	ErrCallbackTokenVerificationFailed = errors.New("token verification failed")
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
		return dto.CallbackOutput{}, fmt.Errorf("OP returned %q: %w", input.Error, ErrCallbackAuthorizationError)
	}
	if input.State == "" {
		return dto.CallbackOutput{}, fmt.Errorf("state parameter missing: %w", ErrCallbackStateMismatch)
	}
	if input.Code == "" {
		return dto.CallbackOutput{}, fmt.Errorf("code parameter missing: %w", ErrCallbackStateMismatch)
	}

	// Look up the transaction by state (validates CSRF)
	tx, err := u.transactionRepo.FindByState(ctx, input.State)
	if err != nil {
		if errors.Is(err, port.ErrTransactionNotFound) {
			return dto.CallbackOutput{}, fmt.Errorf("state not found or expired: %w", ErrCallbackStateMismatch)
		}
		return dto.CallbackOutput{}, fmt.Errorf("failed to lookup transaction: %w", err)
	}

	// Delete the transaction immediately — one-time use (replay prevention)
	if err := u.transactionRepo.Delete(ctx, input.State); err != nil {
		return dto.CallbackOutput{}, fmt.Errorf("failed to delete transaction: %w", err)
	}

	provider, ok := u.providers[tx.Idp()]
	if !ok {
		return dto.CallbackOutput{}, fmt.Errorf("unknown IdP %q in transaction", tx.Idp())
	}

	metadata, err := u.discoveryClient.GetProviderMetadata(ctx, provider.Issuer())
	if err != nil {
		return dto.CallbackOutput{}, fmt.Errorf("failed to get provider metadata: %w", err)
	}

	// Step 3.2: Exchange authorization code for tokens
	tokenResp, err := u.tokenClient.Exchange(ctx, port.TokenExchangeRequest{
		TokenEndpoint: metadata.TokenEndpoint(),
		Code:          input.Code,
		RedirectUri:   provider.RedirectUri(),
		Provider:      provider,
	})
	if err != nil {
		return dto.CallbackOutput{}, fmt.Errorf("token exchange failed: %w", ErrCallbackTokenVerificationFailed)
	}

	// Validate token_type (RFC 6750)
	if !strings.EqualFold(tokenResp.TokenType, "Bearer") {
		return dto.CallbackOutput{}, fmt.Errorf("unsupported token type %q: %w", tokenResp.TokenType, ErrCallbackTokenVerificationFailed)
	}

	// Step 4: Verify the ID token
	claims, err := u.tokenVerifier.Verify(ctx, tokenResp.IdToken, tx.Nonce(), provider.ClientId(), provider.Issuer())
	if err != nil {
		return dto.CallbackOutput{}, fmt.Errorf("ID token verification failed: %w", ErrCallbackTokenVerificationFailed)
	}

	// Create session
	sessionId, err := u.randomGen.Generate(32) // 256 bits
	if err != nil {
		return dto.CallbackOutput{}, fmt.Errorf("failed to generate session ID: %w", err)
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
		return dto.CallbackOutput{}, fmt.Errorf("failed to save session: %w", err)
	}

	return dto.CallbackOutput{
		ReturnTo:  tx.ReturnTo(),
		SessionId: sessionId,
	}, nil
}
