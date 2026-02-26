package main

import (
	"log"
	"net/http"
	"os"

	"oidc-tutorial/internal/bootstrap"
)

func main() {
	app, err := bootstrap.InitializeApp()
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, app.Router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
