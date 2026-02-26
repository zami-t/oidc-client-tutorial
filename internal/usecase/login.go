package usecase

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/port"
	"oidc-tutorial/internal/domain/service"
	"oidc-tutorial/internal/usecase/dto"
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
		return dto.LoginOutput{}, model.NewAppError(
			model.ErrCodeUnknownIdp,
			fmt.Sprintf("unknown IdP: %q", input.Idp),
			nil,
		)
	}

	metadata, err := u.discoveryClient.GetProviderMetadata(ctx, provider.Issuer())
	if err != nil {
		return dto.LoginOutput{}, model.NewAppError(model.ErrCodeServerError, "failed to get provider metadata", err)
	}

	state, err := u.randomGen.Generate(32) // 256 bits
	if err != nil {
		return dto.LoginOutput{}, model.NewAppError(model.ErrCodeServerError, "failed to generate state", err)
	}
	nonce, err := u.randomGen.Generate(32) // 256 bits
	if err != nil {
		return dto.LoginOutput{}, model.NewAppError(model.ErrCodeServerError, "failed to generate nonce", err)
	}

	tx := model.NewAuthorizationTransaction(state, nonce, input.ReturnTo, input.Idp)
	if err := u.transactionRepo.Save(ctx, tx, u.transactionTtl); err != nil {
		return dto.LoginOutput{}, model.NewAppError(model.ErrCodeServerError, "failed to save transaction", err)
	}

	redirectUrl := buildAuthorizationUrl(metadata.AuthorizationEndpoint(), provider, state, nonce)
	return dto.LoginOutput{RedirectUrl: redirectUrl}, nil
}

func buildAuthorizationUrl(endpoint string, provider model.Provider, state, nonce string) string {
	params := url.Values{
		"response_type": {"code"},
		"client_id":     {provider.ClientId()},
		"redirect_uri":  {provider.RedirectUri()},
		"scope":         {strings.Join(provider.Scopes(), " ")},
		"state":         {state},
		"nonce":         {nonce},
	}
	return endpoint + "?" + params.Encode()
}
