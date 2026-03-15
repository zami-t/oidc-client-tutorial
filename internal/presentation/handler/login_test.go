package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/service"
	"oidc-tutorial/internal/logger"
	"oidc-tutorial/internal/presentation/handler"
	"oidc-tutorial/internal/usecase"
)

// --- stubs (shared with handler-level tests) ---

type handlerStubTransactionRepo struct {
	saveErr error
}

func (s *handlerStubTransactionRepo) Save(_ context.Context, _ model.AuthorizationTransaction, _ time.Duration) error {
	return s.saveErr
}

func (s *handlerStubTransactionRepo) FindByState(_ context.Context, _ string) (model.AuthorizationTransaction, error) {
	return model.AuthorizationTransaction{}, nil
}

func (s *handlerStubTransactionRepo) Delete(_ context.Context, _ string) error {
	return nil
}

type handlerStubDiscoveryClient struct {
	metadata model.ProviderMetadata
	err      error
}

func (s *handlerStubDiscoveryClient) GetProviderMetadata(_ context.Context, _ model.Issuer) (model.ProviderMetadata, error) {
	return s.metadata, s.err
}

func (s *handlerStubDiscoveryClient) GetJwks(_ context.Context, _ model.Issuer) (model.JwkSet, error) {
	return model.NewJwkSet(nil), nil
}

func (s *handlerStubDiscoveryClient) RefreshJwks(_ context.Context, _ model.Issuer, _ string) (model.JwkSet, error) {
	return model.NewJwkSet(nil), nil
}

// --- helpers ---

func buildHandlerLoginUsecase(txRepo *handlerStubTransactionRepo, discovery *handlerStubDiscoveryClient) *usecase.LoginUsecase {
	providers := map[string]model.Provider{
		"google": model.NewProvider(
			"google",
			model.NewIssuer("https://accounts.google.com"),
			model.NewClient("client-id", "client-secret", model.NewRedirectUri("http://localhost:8080/callback")),
			[]string{"openid", "email", "profile"},
			model.AuthMethodBasic,
		),
	}
	return usecase.NewLoginUsecase(
		providers,
		txRepo,
		discovery,
		service.RandomGenerator{},
		10*time.Minute,
	)
}

func testLogger() *logger.Logger {
	return logger.New("test", "test")
}

func buildHandlerStubMetadata() model.ProviderMetadata {
	return model.NewProviderMetadata(
		model.NewIssuer("https://accounts.google.com"),
		"https://accounts.google.com/o/oauth2/v2/auth",
		"https://oauth2.googleapis.com/token",
		"https://openidconnect.googleapis.com/v1/userinfo",
		"https://www.googleapis.com/oauth2/v3/certs",
		[]string{"code"},
		[]string{"public"},
		[]string{"RS256"},
	)
}

// --- tests ---

func TestLoginHandler_ValidIdp_Redirects(t *testing.T) {
	txRepo := &handlerStubTransactionRepo{}
	discovery := &handlerStubDiscoveryClient{metadata: buildHandlerStubMetadata()}
	uc := buildHandlerLoginUsecase(txRepo, discovery)
	h := handler.NewLoginHandler(uc, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/login?idp=google", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("Location header is empty")
	}

	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("invalid Location URL: %v", err)
	}
	q := parsed.Query()
	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q, want %q", q.Get("response_type"), "code")
	}
	if q.Get("client_id") != "client-id" {
		t.Errorf("client_id = %q, want %q", q.Get("client_id"), "client-id")
	}
	if q.Get("state") == "" {
		t.Error("state must not be empty in redirect URL")
	}
	if q.Get("nonce") == "" {
		t.Error("nonce must not be empty in redirect URL")
	}
}

func TestLoginHandler_UnknownIdp_Returns400(t *testing.T) {
	txRepo := &handlerStubTransactionRepo{}
	discovery := &handlerStubDiscoveryClient{metadata: buildHandlerStubMetadata()}
	uc := buildHandlerLoginUsecase(txRepo, discovery)
	h := handler.NewLoginHandler(uc, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/login?idp=unknown", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestLoginHandler_MissingIdp_Returns400(t *testing.T) {
	txRepo := &handlerStubTransactionRepo{}
	discovery := &handlerStubDiscoveryClient{metadata: buildHandlerStubMetadata()}
	uc := buildHandlerLoginUsecase(txRepo, discovery)
	h := handler.NewLoginHandler(uc, testLogger())

	// no idp query param → empty string → unknown IdP
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestLoginHandler_DiscoveryError_Returns500(t *testing.T) {
	txRepo := &handlerStubTransactionRepo{}
	discovery := &handlerStubDiscoveryClient{err: errors.New("discovery unavailable")}
	uc := buildHandlerLoginUsecase(txRepo, discovery)
	h := handler.NewLoginHandler(uc, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/login?idp=google", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestLoginHandler_TransactionSaveError_Returns500(t *testing.T) {
	txRepo := &handlerStubTransactionRepo{saveErr: errors.New("redis unavailable")}
	discovery := &handlerStubDiscoveryClient{metadata: buildHandlerStubMetadata()}
	uc := buildHandlerLoginUsecase(txRepo, discovery)
	h := handler.NewLoginHandler(uc, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/login?idp=google", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestLoginHandler_ReturnTo_NotIncludedInRedirectUrl(t *testing.T) {
	// return_to is stored in transaction, NOT forwarded to OP
	txRepo := &handlerStubTransactionRepo{}
	discovery := &handlerStubDiscoveryClient{metadata: buildHandlerStubMetadata()}
	uc := buildHandlerLoginUsecase(txRepo, discovery)
	h := handler.NewLoginHandler(uc, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/login?idp=google&return_to=/dashboard", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	location := rec.Header().Get("Location")
	parsed, _ := url.Parse(location)
	if parsed.Query().Get("return_to") != "" {
		t.Error("return_to must not appear in the OP authorization URL")
	}
}
