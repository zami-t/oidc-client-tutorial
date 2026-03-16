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
	"oidc-tutorial/internal/logger"
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
	log             *logger.Logger
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
	log *logger.Logger,
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
		log:             log,
	}
}

// Execute processes the callback, returning the session ID and post-login redirect URL.
func (u *CallbackUsecase) Execute(ctx context.Context, input dto.CallbackInput) (dto.CallbackOutput, error) {
	// Step 3.1: Handle OP-side authorization errors
	if input.Error != "" {
		err := fmt.Errorf("OP returned %q: %w", input.Error, ErrCallbackAuthorizationError)
		u.log.Info(ctx, "callback: authorization error from OP")
		return dto.CallbackOutput{}, err
	}
	if input.State == "" {
		err := fmt.Errorf("state parameter missing: %w", ErrCallbackStateMismatch)
		u.log.Info(ctx, "callback: state parameter missing")
		return dto.CallbackOutput{}, err
	}
	if input.Code == "" {
		err := fmt.Errorf("code parameter missing: %w", ErrCallbackStateMismatch)
		u.log.Info(ctx, "callback: code parameter missing")
		return dto.CallbackOutput{}, err
	}

	// Look up the transaction by state (validates CSRF)
	tx, err := u.transactionRepo.FindByState(ctx, input.State)
	if err != nil {
		if errors.Is(err, port.ErrTransactionNotFound) {
			wrapped := fmt.Errorf("state not found or expired: %w", ErrCallbackStateMismatch)
			u.log.Warn(ctx, "callback: state not found or expired", wrapped)
			return dto.CallbackOutput{}, wrapped
		}
		wrapped := fmt.Errorf("failed to lookup transaction: %w", err)
		u.log.Error(ctx, "callback: failed to lookup transaction", "CALLBACK_TRANSACTION_LOOKUP_FAILED", wrapped)
		return dto.CallbackOutput{}, wrapped
	}

	// Delete the transaction immediately — one-time use (replay prevention)
	if err := u.transactionRepo.Delete(ctx, input.State); err != nil {
		wrapped := fmt.Errorf("failed to delete transaction: %w", err)
		u.log.Error(ctx, "callback: failed to delete transaction", "CALLBACK_TRANSACTION_DELETE_FAILED", wrapped)
		return dto.CallbackOutput{}, wrapped
	}

	u.log.Info(ctx, "callback: state validated")

	provider, ok := u.providers[tx.Idp()]
	if !ok {
		err := fmt.Errorf("unknown IdP %q in transaction", tx.Idp())
		u.log.Error(ctx, "callback: unknown idp in transaction", "CALLBACK_UNKNOWN_IDP", err)
		return dto.CallbackOutput{}, err
	}

	metadata, err := u.discoveryClient.GetProviderMetadata(ctx, provider.Issuer())
	if err != nil {
		wrapped := fmt.Errorf("failed to get provider metadata: %w", err)
		u.log.Error(ctx, "callback: failed to get provider metadata", "CALLBACK_DISCOVERY_FAILED", wrapped)
		return dto.CallbackOutput{}, wrapped
	}

	// Step 3.2: Exchange authorization code for tokens
	tokenResp, err := u.tokenClient.Exchange(ctx, port.TokenExchangeRequest{
		TokenEndpoint: metadata.TokenEndpoint(),
		Code:          input.Code,
		RedirectUri:   provider.Client().RedirectUri().String(),
		Provider:      provider,
	})
	if err != nil {
		wrapped := fmt.Errorf("token exchange failed: %w", ErrCallbackTokenVerificationFailed)
		u.log.Error(ctx, "callback: token exchange failed", "CALLBACK_TOKEN_EXCHANGE_FAILED", wrapped)
		return dto.CallbackOutput{}, wrapped
	}

	// Validate token_type (RFC 6750)
	if !strings.EqualFold(tokenResp.TokenType, "Bearer") {
		wrapped := fmt.Errorf("unsupported token type %q: %w", tokenResp.TokenType, ErrCallbackTokenVerificationFailed)
		u.log.Error(ctx, "callback: unsupported token type", "CALLBACK_TOKEN_EXCHANGE_FAILED", wrapped)
		return dto.CallbackOutput{}, wrapped
	}

	u.log.Info(ctx, "callback: token exchanged")

	// Step 4: Verify the ID token
	claims, err := u.tokenVerifier.Verify(ctx, tokenResp.IdToken, tx.Nonce(), provider.Client().Id(), provider.Issuer())
	if err != nil {
		wrapped := fmt.Errorf("ID token verification failed: %w", ErrCallbackTokenVerificationFailed)
		u.log.Error(ctx, "callback: id token verification failed", "CALLBACK_TOKEN_VERIFY_FAILED", wrapped)
		return dto.CallbackOutput{}, wrapped
	}

	u.log.Info(ctx, "callback: id token verified")

	// Create session
	sessionId, err := u.randomGen.Generate(32) // 256 bits
	if err != nil {
		wrapped := fmt.Errorf("failed to generate session ID: %w", err)
		u.log.Error(ctx, "callback: failed to generate session id", "CALLBACK_SESSION_CREATE_FAILED", wrapped)
		return dto.CallbackOutput{}, wrapped
	}

	session := model.NewAppSession(
		sessionId,
		claims.Subject(),
		claims.Issuer(),
		claims.Email(),
		claims.Name(),
		claims.Picture(),
	)
	if err := u.sessionRepo.Save(ctx, session, u.sessionTtl); err != nil {
		wrapped := fmt.Errorf("failed to save session: %w", err)
		u.log.Error(ctx, "callback: failed to save session", "CALLBACK_SESSION_CREATE_FAILED", wrapped)
		return dto.CallbackOutput{}, wrapped
	}

	u.log.Info(ctx, "callback: session created")

	return dto.CallbackOutput{
		ReturnTo:  tx.ReturnTo(),
		SessionId: sessionId,
	}, nil
}
