package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"whatsapp-api/internal/api"
	"whatsapp-api/internal/auth"
	"whatsapp-api/internal/config"
	"whatsapp-api/internal/db"
	"whatsapp-api/internal/session"
	"whatsapp-api/internal/webhook"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load .env if present
	_ = godotenv.Load()

	// Logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Config
	cfg := config.Load()
	log.Info().Str("port", cfg.Port).Str("db_driver", cfg.DBDriver).Msg("starting WhatsApp API")

	// Ensure data directory exists
	dataDir := "/app/data"
	if err := os.MkdirAll(dataDir+"/sessions", 0755); err != nil {
		log.Fatal().Err(err).Msg("create data directory")
	}
	if err := os.MkdirAll(cfg.MediaStoragePath, 0755); err != nil {
		log.Fatal().Err(err).Msg("create media directory")
	}

	// Database
	database, err := db.Open(cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("open database")
	}
	defer database.Close()

	// Seed initial admin user (no-op if an admin already exists)
	if err := auth.SeedAdminUser(database, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		log.Fatal().Err(err).Msg("seed admin user")
	}
	log.Info().Str("username", cfg.AdminUsername).Msg("admin account ready")

	// Webhook dispatcher
	dispatcher := webhook.NewDispatcher(database, cfg.WebhookTimeout, cfg.WebhookMaxRetries)

	// SSE hub for real-time agent notifications
	hub := api.NewSSEHub()

	// Session manager (restores persisted sessions)
	mgr, err := session.NewManager(database, cfg, dispatcher, hub)
	if err != nil {
		log.Fatal().Err(err).Msg("init session manager")
	}

	// HTTP Router
	router := api.NewRouter(cfg, database, mgr, dispatcher, hub)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info().Str("addr", srv.Addr).Msg("server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	<-quit
	log.Info().Msg("shutting down...")

	// Shutdown all session queues
	for _, sess := range mgr.List() {
		if sess.Queue != nil {
			sess.Queue.Stop()
		}
	}

	log.Info().Msg("server stopped")
}
