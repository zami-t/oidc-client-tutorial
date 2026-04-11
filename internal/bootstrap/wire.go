package bootstrap

import (
	"context"
	"errors"
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

type sessionCookieConfig struct {
	secure   bool
	sameSite http.SameSite
}

// config holds application configuration loaded from environment variables.
type config struct {
	port                    string
	version                 string
	googleClientId          string
	googleClientSecret      string
	redirectUri             string
	authMethod              model.AuthMethod
	sessionTtl              time.Duration
	transactionTtl          time.Duration
	sessionCookie           sessionCookieConfig
	redisAddr               string
	discoveryTimeoutSeconds int
	allowedReturnToOrigins  []string
}

// Server wraps the HTTP server and external connections, providing lifecycle management.
type Server struct {
	httpServer  *http.Server
	redisClient *redis.Client
}

// Addr returns the listening address of the server.
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

// GracefulShutdown drains active HTTP connections then closes external connections.
// Both errors are returned if multiple failures occur.
func (s *Server) GracefulShutdown(ctx context.Context) error {
	var httpErr, redisErr error
	if err := s.httpServer.Shutdown(ctx); err != nil {
		httpErr = fmt.Errorf("http shutdown: %w", err)
	}
	if err := s.redisClient.Close(); err != nil {
		redisErr = fmt.Errorf("redis close: %w", err)
	}
	return errors.Join(httpErr, redisErr)
}

// CloseExternalConnections closes external connections without draining HTTP.
// Use this when the HTTP server has already stopped.
func (s *Server) CloseExternalConnections() error {
	if err := s.redisClient.Close(); err != nil {
		return fmt.Errorf("redis close: %w", err)
	}
	return nil
}

// InitializeServer wires all dependencies and returns a ready-to-run Server.
func InitializeServer() (*Server, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	log := logger.New("oidc-client", cfg.version)

	// Infrastructure
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.redisAddr,
	})
	transactionRepo := repository.NewRedisTransactionRepository(redisClient)
	sessionRepo := repository.NewRedisSessionRepository(redisClient)
	discoveryClient := infraClient.NewHttpDiscoveryClient(time.Duration(cfg.discoveryTimeoutSeconds) * time.Second)
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
			model.NewClient(cfg.googleClientId, cfg.googleClientSecret, model.NewRedirectUri(cfg.redirectUri)),
			[]string{"openid", "email", "profile"},
			cfg.authMethod,
		),
	}

	// Usecases
	loginUC := usecase.NewLoginUsecase(
		transactionRepo,
		orchestrator,
		randomGen,
		cfg.transactionTtl,
		cfg.allowedReturnToOrigins,
		log,
	)
	callbackUC := usecase.NewCallbackUsecase(
		transactionRepo,
		sessionRepo,
		orchestrator,
		tokenClient,
		tokenVerifier,
		randomGen,
		cfg.sessionTtl,
		log,
	)
	logoutUC := usecase.NewLogoutUsecase(sessionRepo, log)
	meUC := usecase.NewMeUsecase(sessionRepo, log)

	// Handlers
	cookieManager := handler.NewCookieManager(cfg.sessionCookie.secure, cfg.sessionCookie.sameSite)
	loginH := handler.NewLoginHandler(loginUC, log)
	callbackH := handler.NewCallbackHandler(callbackUC, log, cookieManager)
	logoutH := handler.NewLogoutHandler(logoutUC, log, cookieManager)
	meH := handler.NewMeHandler(meUC, log)
	healthH := &handler.HealthHandler{}

	// Router (Go 1.22+ method+path routing)
	mux := http.NewServeMux()
	mux.Handle("GET /login", loginH)
	mux.Handle("GET /callback", callbackH)
	mux.Handle("POST /logout", logoutH)
	mux.Handle("GET /me", meH)
	mux.Handle("GET /health", healthH)

	router := handler.NewTraceMiddleware(log, mux)
	return &Server{
		httpServer: &http.Server{
			Addr:    ":" + cfg.port,
			Handler: router,
		},
		redisClient: redisClient,
	}, nil
}

func loadConfig() (*config, error) {
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

	cookieSecure := false
	if v := os.Getenv("COOKIE_SECURE"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			cookieSecure = parsed
		}
	}

	cookieSameSite := http.SameSiteLaxMode
	switch os.Getenv("COOKIE_SAME_SITE") {
	case "strict":
		cookieSameSite = http.SameSiteStrictMode
	case "none":
		cookieSameSite = http.SameSiteNoneMode
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

	return &config{
		port:               port,
		version:            version,
		googleClientId:     clientId,
		googleClientSecret: clientSecret,
		redirectUri:        redirectUri,
		authMethod:         authMethod,
		sessionTtl:         sessionTtl,
		transactionTtl:     transactionTtl,
		sessionCookie: sessionCookieConfig{
			secure:   cookieSecure,
			sameSite: cookieSameSite,
		},
		redisAddr:               redisAddr,
		discoveryTimeoutSeconds: discoveryTimeoutSeconds,
		allowedReturnToOrigins:  allowedReturnToOrigins,
	}, nil
}
