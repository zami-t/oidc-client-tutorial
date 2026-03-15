package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"oidc-tutorial/internal/bootstrap"
	"oidc-tutorial/internal/logger"
)

func main() {
	log := logger.New("oidc-client", "0.1.0")
	ctx := context.Background()

	app, err := bootstrap.InitializeApp()
	if err != nil {
		log.Error(ctx, fmt.Sprintf("failed to initialize app: %v", err), "INIT_FAILED", err)
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Info(ctx, fmt.Sprintf("starting server on :%s", port))
	if err := http.ListenAndServe(":"+port, app.Router); err != nil {
		log.Error(ctx, fmt.Sprintf("server error: %v", err), "SERVER_ERROR", err)
		os.Exit(1)
	}
}
