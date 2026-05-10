package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/autotraka/go-gateway/internal/config"
	"github.com/autotraka/go-gateway/internal/db"
	"github.com/autotraka/go-gateway/internal/health"
	applog "github.com/autotraka/go-gateway/internal/log"
	"github.com/autotraka/go-gateway/internal/telemetry"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Default().Error("failed to load config", "error", err)
		os.Exit(1)
	}

	applog.Init(cfg.Env)
	logger := slog.Default()
	logger.Info("starting go-gateway", "env", cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tp, err := telemetry.InitTracer(ctx)
	if err != nil {
		logger.Error("failed to init tracer", "error", err)
		os.Exit(1)
	}

	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	r := chi.NewRouter()
	r.Use(otelhttp.NewMiddleware("go-gateway"))

	r.Get("/health", health.Handler)
	r.Get("/ping", health.Ping)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		logger.Info("server listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "error", err)
	}

	if err := database.Close(); err != nil {
		logger.Error("database close failed", "error", err)
	}

	if err := telemetry.ShutdownTracer(shutdownCtx, tp); err != nil {
		logger.Error("tracer shutdown failed", "error", err)
	}

	logger.Info("shutdown complete")
}
