package main

import (
	"log"
	"net/http"

	"github.com/autotraka/go-gateway/internal/config"
	"github.com/autotraka/go-gateway/internal/health"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.Load()

	r := chi.NewRouter()

	// Health endpoints
	r.Get("/health", health.Handler)
	r.Get("/ping", health.Ping)

	log.Printf("Starting go-gateway on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
