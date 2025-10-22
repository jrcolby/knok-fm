package main

import (
	"context"
	"database/sql"
	"fmt"
	"knock-fm/internal/config"
	"knock-fm/internal/pkg/logger"
	"knock-fm/internal/repository/postgres"
	"knock-fm/internal/repository/redis"
	"knock-fm/internal/service/bot"
	"knock-fm/internal/service/platforms"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Validate bot-specific configuration
	if err := cfg.ValidateForBot(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	log := logger.New(cfg.LogLevel)
	log.Info("Starting Discord bot service...")

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}

	// Run database migrations
	if err := postgres.RunMigrations(db, log); err != nil {
		log.Error("Failed to run database migrations", "error", err)
		os.Exit(1)
	}

	// Connect to Redis
	redisClient, err := redis.NewClient(cfg.RedisURL, log)
	if err != nil {
		log.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// Test Redis connection
	if err := redis.HealthCheck(context.Background(), redisClient); err != nil {
		log.Error("Failed to ping Redis", "error", err)
		os.Exit(1)
	}

	// Create repositories
	queueRepo := redis.NewQueueRepository(redisClient, log)
	knokRepo := postgres.NewKnokRepository(db, log)
	serverRepo := postgres.NewServerRepository(db, log)
	platformRepo := postgres.NewPlatformRepository(db, log)

	// Create and load platform loader
	platformLoader := platforms.NewLoader(platformRepo, log)
	ctx := context.Background()
	if err := platformLoader.Load(ctx); err != nil {
		log.Error("Failed to load platforms", "error", err)
		os.Exit(1)
	}
	log.Info("Platform loader initialized",
		"platform_count", platformLoader.Count(),
	)

	// Create bot service
	botService, err := bot.New(cfg, log, queueRepo, knokRepo, serverRepo, platformLoader)
	if err != nil {
		log.Error("Failed to create bot service", "error", err)
		os.Exit(1)
	}

	// Create a channel to track shutdown completion
	done := make(chan struct{})

	// Start bot service in a goroutine
	go func() {
		defer close(done)
		if err := botService.Start(); err != nil {
			log.Error("Bot service failed", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either shutdown signal or service completion
	select {
	case <-quit:
		log.Info("Shutdown signal received, stopping bot service...")
	case <-done:
		log.Info("Bot service completed")
	}

	// Graceful shutdown with timeout
	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop bot service
	if err := botService.Stop(); err != nil {
		log.Error("Error stopping bot service", "error", err)
	}

	log.Info("Bot service shutdown complete")
}
