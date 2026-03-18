package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/port"
	"oidc-tutorial/internal/domain/service"
	"oidc-tutorial/internal/logger"
	"oidc-tutorial/internal/usecase/dto"
)

// Sentinel errors for LoginUsecase.
var (
	ErrLoginUnknownIdp = errors.New("unknown identity provider")
)

// LoginUsecase orchestrates the /login endpoint: generates state/nonce,
// stores the transaction, and builds the OP authorization URL.
type LoginUsecase struct {
	transactionRepo port.TransactionRepository
	discoveryClient port.DiscoveryClient
	randomGen       service.RandomGenerator
	transactionTtl  time.Duration
	log             *logger.Logger
}

// NewLoginUsecase creates a new LoginUsecase.
func NewLoginUsecase(
	transactionRepo port.TransactionRepository,
	discoveryClient port.DiscoveryClient,
	randomGen service.RandomGenerator,
	transactionTtl time.Duration,
	log *logger.Logger,
) *LoginUsecase {
	return &LoginUsecase{
		transactionRepo: transactionRepo,
		discoveryClient: discoveryClient,
		randomGen:       randomGen,
		transactionTtl:  transactionTtl,
		log:             log,
	}
}

// Execute starts the authorization flow and returns the redirect URL.
func (u *LoginUsecase) Execute(ctx context.Context, input dto.LoginInput) (dto.LoginOutput, error) {
	provider, ok := model.Registry.Get(input.Idp)
	if !ok {
		err := fmt.Errorf("idp %q: %w", input.Idp, ErrLoginUnknownIdp)
		u.log.Info(ctx, "login: unknown idp requested")
		return dto.LoginOutput{}, err
	}

	metadata, err := u.discoveryClient.GetProviderMetadata(ctx, provider.Issuer())
	if err != nil {
		wrapped := fmt.Errorf("failed to get provider metadata: %w", err)
		u.log.Error(ctx, "login: failed to get provider metadata", "LOGIN_DISCOVERY_FAILED", wrapped)
		return dto.LoginOutput{}, wrapped
	}

	state, err := u.randomGen.Generate(32) // 256 bits
	if err != nil {
		wrapped := fmt.Errorf("failed to generate state: %w", err)
		u.log.Error(ctx, "login: failed to generate state", "LOGIN_RANDOM_FAILED", wrapped)
		return dto.LoginOutput{}, wrapped
	}
	nonce, err := u.randomGen.Generate(32) // 256 bits
	if err != nil {
		wrapped := fmt.Errorf("failed to generate nonce: %w", err)
		u.log.Error(ctx, "login: failed to generate nonce", "LOGIN_RANDOM_FAILED", wrapped)
		return dto.LoginOutput{}, wrapped
	}

	tx := model.NewAuthorizationTransaction(state, nonce, input.ReturnTo, input.Idp)
	if err := u.transactionRepo.Save(ctx, tx, u.transactionTtl); err != nil {
		wrapped := fmt.Errorf("failed to save transaction: %w", err)
		u.log.Error(ctx, "login: failed to save transaction", "LOGIN_TRANSACTION_SAVE_FAILED", wrapped)
		return dto.LoginOutput{}, wrapped
	}

	u.log.Info(ctx, "login: transaction saved")
	redirectUrl := buildAuthorizationUrl(metadata.AuthorizationEndpoint(), provider, state, nonce)
	return dto.LoginOutput{RedirectUrl: redirectUrl}, nil
}

func buildAuthorizationUrl(endpoint string, provider model.Provider, state, nonce string) string {
	params := url.Values{
		"response_type": {"code"},
		"client_id":     {provider.Client().Id()},
		"redirect_uri":  {provider.Client().RedirectUri().String()},
		"scope":         {strings.Join(provider.Scopes(), " ")},
		"state":         {state},
		"nonce":         {nonce},
	}
	return endpoint + "?" + params.Encode()
}
