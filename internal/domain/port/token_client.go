package port

import (
	"context"

	"oidc-tutorial/internal/domain/model"
)

// TokenExchangeRequest holds the parameters for the token endpoint request.
type TokenExchangeRequest struct {
	TokenEndpoint string
	Code          string
	RedirectUri   string
	Provider      model.Provider
}

// TokenResponse holds the response from the token endpoint.
type TokenResponse struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int
	IdToken     string
	Scope       string
}

// TokenClient exchanges an authorization code for tokens.
type TokenClient interface {
	Exchange(ctx context.Context, req TokenExchangeRequest) (TokenResponse, error)
}
