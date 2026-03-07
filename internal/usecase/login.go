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
	"oidc-tutorial/internal/usecase/dto"
)

// Sentinel errors for LoginUsecase.
var (
	ErrLoginUnknownIdp = errors.New("unknown identity provider")
)

// LoginUsecase orchestrates the /login endpoint: generates state/nonce,
// stores the transaction, and builds the OP authorization URL.
type LoginUsecase struct {
	providers       map[string]model.Provider
	transactionRepo port.TransactionRepository
	discoveryClient port.DiscoveryClient
	randomGen       service.RandomGenerator
	transactionTtl  time.Duration
}

// NewLoginUsecase creates a new LoginUsecase.
func NewLoginUsecase(
	providers map[string]model.Provider,
	transactionRepo port.TransactionRepository,
	discoveryClient port.DiscoveryClient,
	randomGen service.RandomGenerator,
	transactionTtl time.Duration,
) *LoginUsecase {
	return &LoginUsecase{
		providers:       providers,
		transactionRepo: transactionRepo,
		discoveryClient: discoveryClient,
		randomGen:       randomGen,
		transactionTtl:  transactionTtl,
	}
}

// Execute starts the authorization flow and returns the redirect URL.
func (u *LoginUsecase) Execute(ctx context.Context, input dto.LoginInput) (dto.LoginOutput, error) {
	provider, ok := u.providers[input.Idp]
	if !ok {
		return dto.LoginOutput{}, fmt.Errorf("idp %q: %w", input.Idp, ErrLoginUnknownIdp)
	}

	metadata, err := u.discoveryClient.GetProviderMetadata(ctx, provider.Issuer())
	if err != nil {
		return dto.LoginOutput{}, fmt.Errorf("failed to get provider metadata: %w", err)
	}

	state, err := u.randomGen.Generate(32) // 256 bits
	if err != nil {
		return dto.LoginOutput{}, fmt.Errorf("failed to generate state: %w", err)
	}
	nonce, err := u.randomGen.Generate(32) // 256 bits
	if err != nil {
		return dto.LoginOutput{}, fmt.Errorf("failed to generate nonce: %w", err)
	}

	tx := model.NewAuthorizationTransaction(state, nonce, input.ReturnTo, input.Idp)
	if err := u.transactionRepo.Save(ctx, tx, u.transactionTtl); err != nil {
		return dto.LoginOutput{}, fmt.Errorf("failed to save transaction: %w", err)
	}

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
