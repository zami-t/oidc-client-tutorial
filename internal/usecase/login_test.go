package usecase_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/service"
	"oidc-tutorial/internal/usecase"
	ucDto "oidc-tutorial/internal/usecase/dto"
)

// --- stubs ---

type stubTransactionRepo struct {
	saveErr error
	saved   []model.AuthorizationTransaction
}

func (s *stubTransactionRepo) Save(_ context.Context, tx model.AuthorizationTransaction, _ time.Duration) error {
	s.saved = append(s.saved, tx)
	return s.saveErr
}

func (s *stubTransactionRepo) FindByState(_ context.Context, _ string) (model.AuthorizationTransaction, error) {
	return model.AuthorizationTransaction{}, nil
}

func (s *stubTransactionRepo) Delete(_ context.Context, _ string) error {
	return nil
}

type stubDiscoveryClient struct {
	metadata model.ProviderMetadata
	err      error
}

func (s *stubDiscoveryClient) GetProviderMetadata(_ context.Context, _ model.Issuer) (model.ProviderMetadata, error) {
	return s.metadata, s.err
}

func (s *stubDiscoveryClient) GetJwks(_ context.Context, _ model.Issuer) (model.JwkSet, error) {
	return model.NewJwkSet(nil), nil
}

func (s *stubDiscoveryClient) RefreshJwks(_ context.Context, _ model.Issuer, _ string) (model.JwkSet, error) {
	return model.NewJwkSet(nil), nil
}

// --- helpers ---

func buildLoginUsecase(txRepo *stubTransactionRepo, discovery *stubDiscoveryClient) *usecase.LoginUsecase {
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

func stubMetadata() model.ProviderMetadata {
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

func TestLoginUsecase_Execute_ValidIdp(t *testing.T) {
	txRepo := &stubTransactionRepo{}
	discovery := &stubDiscoveryClient{metadata: stubMetadata()}
	uc := buildLoginUsecase(txRepo, discovery)

	out, err := uc.Execute(context.Background(), ucDto.LoginInput{
		Idp:      "google",
		ReturnTo: "/dashboard",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.RedirectUrl == "" {
		t.Fatal("expected non-empty redirect URL")
	}

	parsed, err := url.Parse(out.RedirectUrl)
	if err != nil {
		t.Fatalf("invalid redirect URL: %v", err)
	}
	q := parsed.Query()

	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q, want %q", q.Get("response_type"), "code")
	}
	if q.Get("client_id") != "client-id" {
		t.Errorf("client_id = %q, want %q", q.Get("client_id"), "client-id")
	}
	if q.Get("redirect_uri") != "http://localhost:8080/callback" {
		t.Errorf("redirect_uri = %q, want %q", q.Get("redirect_uri"), "http://localhost:8080/callback")
	}
	if q.Get("state") == "" {
		t.Error("state must not be empty")
	}
	if q.Get("nonce") == "" {
		t.Error("nonce must not be empty")
	}
}

func TestLoginUsecase_Execute_TransactionSaved(t *testing.T) {
	txRepo := &stubTransactionRepo{}
	discovery := &stubDiscoveryClient{metadata: stubMetadata()}
	uc := buildLoginUsecase(txRepo, discovery)

	_, err := uc.Execute(context.Background(), ucDto.LoginInput{Idp: "google", ReturnTo: "/home"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txRepo.saved) != 1 {
		t.Fatalf("expected 1 saved transaction, got %d", len(txRepo.saved))
	}
	tx := txRepo.saved[0]
	if tx.State() == "" {
		t.Error("saved state must not be empty")
	}
	if tx.Nonce() == "" {
		t.Error("saved nonce must not be empty")
	}
	if tx.ReturnTo() != "/home" {
		t.Errorf("ReturnTo = %q, want %q", tx.ReturnTo(), "/home")
	}
	if tx.Idp() != "google" {
		t.Errorf("Idp = %q, want %q", tx.Idp(), "google")
	}
}

func TestLoginUsecase_Execute_UnknownIdp(t *testing.T) {
	txRepo := &stubTransactionRepo{}
	discovery := &stubDiscoveryClient{metadata: stubMetadata()}
	uc := buildLoginUsecase(txRepo, discovery)

	_, err := uc.Execute(context.Background(), ucDto.LoginInput{Idp: "unknown"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, usecase.ErrLoginUnknownIdp) {
		t.Errorf("error = %v, want ErrLoginUnknownIdp", err)
	}
}

func TestLoginUsecase_Execute_DiscoveryError(t *testing.T) {
	txRepo := &stubTransactionRepo{}
	discoveryErr := errors.New("discovery failed")
	discovery := &stubDiscoveryClient{err: discoveryErr}
	uc := buildLoginUsecase(txRepo, discovery)

	_, err := uc.Execute(context.Background(), ucDto.LoginInput{Idp: "google"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, discoveryErr) {
		t.Errorf("error chain does not contain discovery error: %v", err)
	}
}

func TestLoginUsecase_Execute_TransactionSaveError(t *testing.T) {
	saveErr := errors.New("redis unavailable")
	txRepo := &stubTransactionRepo{saveErr: saveErr}
	discovery := &stubDiscoveryClient{metadata: stubMetadata()}
	uc := buildLoginUsecase(txRepo, discovery)

	_, err := uc.Execute(context.Background(), ucDto.LoginInput{Idp: "google"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, saveErr) {
		t.Errorf("error chain does not contain save error: %v", err)
	}
}

func TestLoginUsecase_Execute_StateAndNonceAreUnique(t *testing.T) {
	txRepo := &stubTransactionRepo{}
	discovery := &stubDiscoveryClient{metadata: stubMetadata()}
	uc := buildLoginUsecase(txRepo, discovery)

	out1, err := uc.Execute(context.Background(), ucDto.LoginInput{Idp: "google"})
	if err != nil {
		t.Fatalf("first execute: %v", err)
	}
	out2, err := uc.Execute(context.Background(), ucDto.LoginInput{Idp: "google"})
	if err != nil {
		t.Fatalf("second execute: %v", err)
	}

	p1, _ := url.Parse(out1.RedirectUrl)
	p2, _ := url.Parse(out2.RedirectUrl)

	if p1.Query().Get("state") == p2.Query().Get("state") {
		t.Error("state values must be unique across executions")
	}
	if p1.Query().Get("nonce") == p2.Query().Get("nonce") {
		t.Error("nonce values must be unique across executions")
	}
}
