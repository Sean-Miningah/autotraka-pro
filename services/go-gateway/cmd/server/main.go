package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/autotraka/go-gateway/internal/analytics"
	"github.com/autotraka/go-gateway/internal/auth"
	"github.com/autotraka/go-gateway/internal/automation"
	"github.com/autotraka/go-gateway/internal/broadcast"
	"github.com/autotraka/go-gateway/internal/channel"
	"github.com/autotraka/go-gateway/internal/config"
	"github.com/autotraka/go-gateway/internal/contact"
	"github.com/autotraka/go-gateway/internal/conversation"
	"github.com/autotraka/go-gateway/internal/db"
	"github.com/autotraka/go-gateway/internal/eventbus"
	"github.com/autotraka/go-gateway/internal/health"
	migratepkg "github.com/autotraka/go-gateway/internal/migrate"
	applog "github.com/autotraka/go-gateway/internal/log"
	redisclient "github.com/autotraka/go-gateway/internal/redis"
	"github.com/autotraka/go-gateway/internal/scheduler"
	"github.com/autotraka/go-gateway/internal/sqlcgen"
	"github.com/autotraka/go-gateway/internal/template"
	"github.com/autotraka/go-gateway/internal/telemetry"
	"github.com/autotraka/go-gateway/internal/webhook"
	"github.com/autotraka/go-gateway/internal/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			direction := "up"
			if len(os.Args) > 2 {
				direction = os.Args[2]
			}
			if err := migratepkg.Run(cfg.DatabaseURL, direction); err != nil {
				logger.Error("migration failed", "error", err)
				os.Exit(1)
			}
			logger.Info("migration complete", "direction", direction)
			return
		default:
			logger.Error("unknown command", "command", os.Args[1])
			os.Exit(1)
		}
	}

	logger.Info("starting go-gateway", "env", cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tp, err := telemetry.InitTracer(ctx)
	if err != nil {
		logger.Error("failed to init tracer", "error", err)
		os.Exit(1)
	}

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	eb, err := eventbus.New(cfg.NATSURL, logger)
	if err != nil {
		logger.Error("failed to connect to nats", "error", err)
		os.Exit(1)
	}

	queries := sqlcgen.New(pool)
	authSvc := auth.NewService(queries, []byte(cfg.JWTSecret))
	authHandler := auth.NewHandler(authSvc)

	contactSvc := contact.NewService(queries)
	contactHandler := contact.NewHandler(contactSvc)

	// Default channels (temporary — per-tenant DB lookup will replace these).
	wa := channel.NewWhatsApp(cfg.MetaBaseURL, cfg.WhatsAppAccessToken, cfg.WhatsAppPhoneNumberID, cfg.WhatsAppAppSecret, cfg.WhatsAppVerifyToken)
	ig := channel.NewInstagram(cfg.MetaBaseURL, cfg.InstagramAccessToken, cfg.InstagramAccountID, cfg.InstagramAppSecret, cfg.InstagramVerifyToken)
	fb := channel.NewFacebook(cfg.MetaBaseURL, cfg.FacebookAccessToken, cfg.FacebookPageID, cfg.FacebookAppSecret, cfg.FacebookVerifyToken)

	metaTemplateClient := template.NewMetaTemplateAPI(cfg.MetaBaseURL, cfg.WhatsAppAccessToken)
	templateSvc := template.NewService(queries, metaTemplateClient)
	templateHandler := template.NewHandler(templateSvc)

	autoSvc := automation.NewService(queries)
	autoHandler := automation.NewHandler(autoSvc)

	convSvc := conversation.NewService(queries, contactSvc, templateSvc, wa, eb)
	convHandler := conversation.NewHandler(convSvc)
	if err := convSvc.StartAIConsumers(ctx); err != nil {
		logger.Error("failed to start AI consumers", "error", err)
	}

	// Redis client for rate limiting
	var rateLimiter broadcast.RateLimiter = &broadcast.NoopRateLimiter{}
	if cfg.RedisURL != "" {
		redisClient, err := redisclient.New(cfg.RedisURL)
		if err != nil {
			logger.Warn("failed to connect to redis, using noop rate limiter", "error", err)
		} else {
			if err := redisClient.Ping(ctx); err != nil {
				logger.Warn("redis ping failed, using noop rate limiter", "error", err)
			} else {
				rateLimiter = broadcast.NewRedisRateLimiter(redisClient.Client, time.Minute, 80) // 80 per minute (Meta limit)
			}
		}
	}

	broadcastSvc := broadcast.NewService(queries, wa, rateLimiter)
	broadcastHandler := broadcast.NewHandler(broadcastSvc)

	// Analytics
	analyticsSvc := analytics.NewService(queries)
	analyticsHandler := analytics.NewHandler(analyticsSvc)

	wsHub := websocket.NewHub(eb)
	wsHub.Run()
	wsHandler := websocket.NewHandler(wsHub, []byte(cfg.JWTSecret))

	webhookHandler := webhook.NewHandler(queries, eb, wa, uuid.Nil, uuid.Nil)
	instagramWebhookHandler := webhook.NewHandler(queries, eb, ig, uuid.Nil, uuid.Nil)
	facebookWebhookHandler := webhook.NewHandler(queries, eb, fb, uuid.Nil, uuid.Nil)

	// Background worker for unprocessed webhook events.
	worker := webhook.NewWorker(queries, eb, wa)
	go worker.Run(ctx, 30*time.Second)

	// Scheduler with distributed locking.
	sched := scheduler.New(queries)
	hc := scheduler.NewHealthChecker(queries)
	broadcastSched := broadcast.NewSchedulerTask(broadcastSvc)
	sched.RegisterTask("channel-health", 5*time.Minute, hc.CheckAllChannels)
	sched.RegisterTask("template-status-sync", 10*time.Minute, templateSvc.SyncPendingStatuses)
	sched.RegisterTask("broadcast-scheduler", 30*time.Second, broadcastSched.Run)
	analyticsAggregator := analytics.NewAggregatorTask(queries)
	sched.RegisterTask("analytics-aggregate", 24*time.Hour, analyticsAggregator.Run)
	sched.Start(ctx)

	r := chi.NewRouter()
	r.Use(otelhttp.NewMiddleware("go-gateway"))

	r.Get("/health", health.Handler)
	r.Get("/ping", health.Ping)

	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Post("/api/v1/auth/refresh", authHandler.Refresh)

	r.Route("/webhook", func(r chi.Router) {
		r.Get("/whatsapp", webhookHandler.WhatsApp)
		r.Post("/whatsapp", webhookHandler.WhatsApp)
		r.Get("/instagram", instagramWebhookHandler.Instagram)
		r.Post("/instagram", instagramWebhookHandler.Instagram)
		r.Get("/facebook", facebookWebhookHandler.Facebook)
		r.Post("/facebook", facebookWebhookHandler.Facebook)
	})

	r.Route("/internal", func(r chi.Router) {
		r.Use(auth.ServiceTokenMiddleware(cfg.ServiceToken))
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			auth.WriteJSON(w, http.StatusOK, auth.Envelope{Data: "ok"})
		})
	})

	channelHealthHandler := channel.NewHealthHandler(queries)

	r.Group(func(r chi.Router) {
		r.Use(auth.JWTMiddleware([]byte(cfg.JWTSecret)))
		contactHandler.RegisterRoutes(r)
		convHandler.RegisterRoutes(r)
		templateHandler.RegisterRoutes(r)
		autoHandler.RegisterRoutes(r)
		channelHealthHandler.RegisterRoutes(r)
		broadcastHandler.RegisterRoutes(r)
		analyticsHandler.RegisterRoutes(r)
		r.Get("/api/v1/me", func(w http.ResponseWriter, r *http.Request) {
			auth.WriteJSON(w, http.StatusOK, auth.Envelope{
				Data: map[string]interface{}{
					"tenant_id": auth.GetTenantID(r.Context()),
					"member_id": auth.GetMemberID(r.Context()),
					"role":       auth.GetRole(r.Context()),
				},
			})
		})
	})

	r.Get("/api/v1/ws", wsHandler.ServeHTTP)

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

	sched.Stop()

	pool.Close()

	if err := eb.Close(); err != nil {
		logger.Error("eventbus close failed", "error", err)
	}

	if err := telemetry.ShutdownTracer(shutdownCtx, tp); err != nil {
		logger.Error("tracer shutdown failed", "error", err)
	}

	logger.Info("shutdown complete")
}