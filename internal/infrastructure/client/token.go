package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/port"
)

type tokenClient struct {
	httpClient *http.Client
}

// NewTokenClient creates a TokenClient that exchanges authorization codes for tokens.
func NewTokenClient() port.TokenClient {
	return &tokenClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *tokenClient) Exchange(ctx context.Context, req port.TokenExchangeRequest) (port.TokenResponse, error) {
	formValues := url.Values{
		"code":         {req.Code},
		"redirect_uri": {req.RedirectUri},
		"grant_type":   {"authorization_code"},
	}

	var authHeader string
	switch req.Provider.AuthMethod() {
	case model.AuthMethodBasic:
		// RFC 6749 §2.3.1: URL-encode each credential, then Base64-encode "id:secret"
		encodedId := url.QueryEscape(req.Provider.Client().Id())
		encodedSecret := url.QueryEscape(req.Provider.Client().Secret())
		credentials := base64.StdEncoding.EncodeToString([]byte(encodedId + ":" + encodedSecret))
		authHeader = "Basic " + credentials
	case model.AuthMethodPost:
		formValues.Set("client_id", req.Provider.Client().Id())
		formValues.Set("client_secret", req.Provider.Client().Secret())
	default:
		return port.TokenResponse{}, fmt.Errorf("unsupported auth method: %s", req.Provider.AuthMethod())
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		req.TokenEndpoint,
		strings.NewReader(formValues.Encode()),
	)
	if err != nil {
		return port.TokenResponse{}, fmt.Errorf("failed to create token request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if authHeader != "" {
		httpReq.Header.Set("Authorization", authHeader)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return port.TokenResponse{}, fmt.Errorf("failed to send token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return port.TokenResponse{}, fmt.Errorf("failed to read token response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if jsonErr := json.Unmarshal(body, &errBody); jsonErr == nil && errBody.Error != "" {
			return port.TokenResponse{}, fmt.Errorf("token endpoint error %q: %s", errBody.Error, errBody.ErrorDescription)
		}
		return port.TokenResponse{}, fmt.Errorf("token endpoint returned HTTP %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		IdToken     string `json:"id_token"`
		Scope       string `json:"scope"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return port.TokenResponse{}, fmt.Errorf("failed to decode token response: %w", err)
	}

	return port.TokenResponse{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresIn:   tokenResp.ExpiresIn,
		IdToken:     tokenResp.IdToken,
		Scope:       tokenResp.Scope,
	}, nil
}
