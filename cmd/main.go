package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"oidc-tutorial/internal/bootstrap"
	"oidc-tutorial/internal/logger"
)

// Service and Version are set once at startup (in main) and included in every log entry.
const (
	service = "oidc-client"
	version = "1.0.0"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log := logger.New(service, version)

	srv, err := bootstrap.InitializeServer()
	if err != nil {
		log.Error(ctx, fmt.Sprintf("failed to initialize app: %v", err), "INIT_FAILED", err)
		os.Exit(1)
	}

	log.Info(ctx, fmt.Sprintf("starting server on %s", srv.Addr()))
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		stop()
		log.Info(ctx, "shutting down server gracefully")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.GracefulShutdown(shutdownCtx); err != nil {
			log.Error(ctx, fmt.Sprintf("error during server shutdown: %v", err), "SHUTDOWN_ERROR", err)
		}
	case err := <-errCh:
		log.Error(ctx, fmt.Sprintf("unexpected server error: %v", err), "SERVER_ERROR", err)
		if err := srv.CloseExternalConnections(); err != nil {
			log.Error(ctx, fmt.Sprintf("error closing resources: %v", err), "CLOSE_ERROR", err)
		}
	}

	log.Info(ctx, "server shutdown complete")
}
