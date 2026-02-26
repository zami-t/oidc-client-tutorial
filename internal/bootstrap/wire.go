package bootstrap

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/service"
	infraClient "oidc-tutorial/internal/infrastructure/client"
	"oidc-tutorial/internal/infrastructure/repository"
	"oidc-tutorial/internal/presentation/handler"
	"oidc-tutorial/internal/usecase"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	Port               string
	GoogleClientId     string
	GoogleClientSecret string
	RedirectUri        string
	AuthMethod         model.AuthMethod
	SessionTtl         time.Duration
	TransactionTtl     time.Duration
	SecureCookie       bool
}

// App holds the configured HTTP router.
type App struct {
	Router http.Handler
}

// InitializeApp wires all dependencies and returns the application.
func InitializeApp() (*App, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Infrastructure
	transactionRepo := repository.NewMemoryTransactionRepository()
	sessionRepo := repository.NewMemorySessionRepository()
	discoveryClient := infraClient.NewDiscoveryClient(
		6*time.Hour, // metadata TTL
		1*time.Hour, // JWKS TTL
	)
	tokenClient := infraClient.NewTokenClient()

	// Domain
	randomGen := service.RandomGenerator{}
	tokenVerifier := service.NewIdTokenVerifier(discoveryClient)

	// Providers
	googleProvider := model.NewProvider(
		"google",
		"https://accounts.google.com",
		cfg.GoogleClientId,
		cfg.GoogleClientSecret,
		cfg.RedirectUri,
		[]string{"openid", "email", "profile"},
		cfg.AuthMethod,
	)
	providers := map[string]model.Provider{
		"google": googleProvider,
	}

	// Usecases
	loginUC := usecase.NewLoginUsecase(
		providers,
		transactionRepo,
		discoveryClient,
		randomGen,
		cfg.TransactionTtl,
	)
	callbackUC := usecase.NewCallbackUsecase(
		providers,
		transactionRepo,
		sessionRepo,
		discoveryClient,
		tokenClient,
		tokenVerifier,
		randomGen,
		cfg.SessionTtl,
	)
	logoutUC := usecase.NewLogoutUsecase(sessionRepo)
	meUC := usecase.NewMeUsecase(sessionRepo)

	// Handlers
	sameSite := http.SameSiteLaxMode
	loginH := handler.NewLoginHandler(loginUC)
	callbackH := handler.NewCallbackHandler(callbackUC, sameSite, cfg.SecureCookie)
	logoutH := handler.NewLogoutHandler(logoutUC, sameSite, cfg.SecureCookie)
	meH := handler.NewMeHandler(meUC)
	healthH := &handler.HealthHandler{}

	// Router (Go 1.22+ method+path routing)
	mux := http.NewServeMux()
	mux.Handle("GET /login", loginH)
	mux.Handle("GET /callback", callbackH)
	mux.Handle("POST /logout", logoutH)
	mux.Handle("GET /me", meH)
	mux.Handle("GET /health", healthH)

	return &App{Router: mux}, nil
}

func loadConfig() (*Config, error) {
	clientId := os.Getenv("GOOGLE_CLIENT_ID")
	if clientId == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID is required")
	}
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if clientSecret == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_SECRET is required")
	}
	redirectUri := os.Getenv("REDIRECT_URI")
	if redirectUri == "" {
		return nil, fmt.Errorf("REDIRECT_URI is required")
	}

	authMethod := model.AuthMethodBasic
	if m := os.Getenv("AUTH_METHOD"); m == string(model.AuthMethodPost) {
		authMethod = model.AuthMethodPost
	}

	sessionTtl := 60 * time.Minute
	if v := os.Getenv("SESSION_TTL_MINUTES"); v != "" {
		minutes, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid SESSION_TTL_MINUTES: %w", err)
		}
		sessionTtl = time.Duration(minutes) * time.Minute
	}

	transactionTtl := 10 * time.Minute
	if v := os.Getenv("TRANSACTION_TTL_MINUTES"); v != "" {
		minutes, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid TRANSACTION_TTL_MINUTES: %w", err)
		}
		transactionTtl = time.Duration(minutes) * time.Minute
	}

	secureCookie := os.Getenv("SECURE_COOKIE") != "false"

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		Port:               port,
		GoogleClientId:     clientId,
		GoogleClientSecret: clientSecret,
		RedirectUri:        redirectUri,
		AuthMethod:         authMethod,
		SessionTtl:         sessionTtl,
		TransactionTtl:     transactionTtl,
		SecureCookie:       secureCookie,
	}, nil
}
