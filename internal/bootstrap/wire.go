package bootstrap

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"oidc-tutorial/internal/domain/model"
	"oidc-tutorial/internal/domain/service"
	infraClient "oidc-tutorial/internal/infrastructure/client"
	"oidc-tutorial/internal/infrastructure/repository"
	"oidc-tutorial/internal/logger"
	"oidc-tutorial/internal/presentation/handler"
	"oidc-tutorial/internal/usecase"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	Port                    string
	Version                 string
	GoogleClientId          string
	GoogleClientSecret      string
	RedirectUri             string
	AuthMethod              model.AuthMethod
	SessionTtl              time.Duration
	TransactionTtl          time.Duration
	SecureCookie               bool
	RedisAddr                  string
	DiscoveryTimeoutSeconds    int
	AllowedReturnToOrigins     []string
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

	log := logger.New("oidc-client", cfg.Version)

	// Infrastructure
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	transactionRepo := repository.NewRedisTransactionRepository(redisClient)
	sessionRepo := repository.NewRedisSessionRepository(redisClient)
	discoveryClient := infraClient.NewHttpDiscoveryClient(time.Duration(cfg.DiscoveryTimeoutSeconds) * time.Second)
	discoveryCacheClient := infraClient.NewRedisDiscoveryCacheClient(
		redisClient,
		6*time.Hour, // metadata TTL
		1*time.Hour, // JWKS TTL
	)
	orchestrator := infraClient.NewOrchestrateDiscoveryClient(discoveryClient, discoveryCacheClient)
	tokenClient := infraClient.NewTokenClient()

	// Domain
	randomGen := service.RandomGenerator{}
	tokenVerifier := service.NewIdTokenVerifier(orchestrator)

	// Providers — RP-global registry of supported identity providers
	model.Registry = model.ProviderRegistry{
		"google": model.NewProvider(
			"google",
			model.NewIssuer("https://accounts.google.com"),
			model.NewClient(cfg.GoogleClientId, cfg.GoogleClientSecret, model.NewRedirectUri(cfg.RedirectUri)),
			[]string{"openid", "email", "profile"},
			cfg.AuthMethod,
		),
	}

	// Usecases
	loginUC := usecase.NewLoginUsecase(
		transactionRepo,
		orchestrator,
		randomGen,
		cfg.TransactionTtl,
		cfg.AllowedReturnToOrigins,
		log,
	)
	callbackUC := usecase.NewCallbackUsecase(
		transactionRepo,
		sessionRepo,
		orchestrator,
		tokenClient,
		tokenVerifier,
		randomGen,
		cfg.SessionTtl,
		log,
	)
	logoutUC := usecase.NewLogoutUsecase(sessionRepo, log)
	meUC := usecase.NewMeUsecase(sessionRepo, log)

	// Handlers
	sameSite := http.SameSiteLaxMode
	loginH := handler.NewLoginHandler(loginUC, log)
	callbackH := handler.NewCallbackHandler(callbackUC, sameSite, cfg.SecureCookie, log)
	logoutH := handler.NewLogoutHandler(logoutUC, sameSite, cfg.SecureCookie, log)
	meH := handler.NewMeHandler(meUC, log)
	healthH := &handler.HealthHandler{}

	// Router (Go 1.22+ method+path routing)
	mux := http.NewServeMux()
	mux.Handle("GET /login", loginH)
	mux.Handle("GET /callback", callbackH)
	mux.Handle("POST /logout", logoutH)
	mux.Handle("GET /me", meH)
	mux.Handle("GET /health", healthH)

	middleware := handler.NewTraceMiddleware(log, mux)
	return &App{Router: middleware}, nil
}

func loadConfig() (*Config, error) {
	version := os.Getenv("VERSION")
	if version == "" {
		version = "unknown"
	}

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

	secureCookie := false
	if v := os.Getenv("SECURE_COOKIE"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			secureCookie = parsed
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	discoveryTimeoutSeconds := 10
	if v := os.Getenv("DISCOVERY_TIMEOUT_SECONDS"); v != "" {
		seconds, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid DISCOVERY_TIMEOUT_SECONDS: %w", err)
		}
		discoveryTimeoutSeconds = seconds
	}

	var allowedReturnToOrigins []string
	if v := os.Getenv("ALLOWED_RETURN_TO_ORIGINS"); v != "" {
		allowedReturnToOrigins = strings.Split(v, ",")
	}

	return &Config{
		Port:                    port,
		Version:                 version,
		GoogleClientId:          clientId,
		GoogleClientSecret:      clientSecret,
		RedirectUri:             redirectUri,
		AuthMethod:              authMethod,
		SessionTtl:              sessionTtl,
		TransactionTtl:          transactionTtl,
		SecureCookie:            secureCookie,
		RedisAddr:               redisAddr,
		DiscoveryTimeoutSeconds: discoveryTimeoutSeconds,
		AllowedReturnToOrigins:  allowedReturnToOrigins,
	}, nil
}
