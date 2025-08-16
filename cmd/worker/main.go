package main

import (
	"context"
	"database/sql"
	"fmt"
	"knock-fm/internal/config"
	"knock-fm/internal/pkg/logger"
	"knock-fm/internal/repository/postgres"
	"knock-fm/internal/repository/redis"
	"knock-fm/internal/service/worker"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Validate worker-specific configuration
	if err := cfg.ValidateForWorker(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	log := logger.New(cfg.LogLevel)
	log.Info("Starting worker service...")

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
	if err := redisClient.Ping(context.Background()); err != nil {
		log.Error("Failed to ping Redis", "error", err)
		os.Exit(1)
	}

	// Create repositories
	queueRepo := redis.NewQueueRepository(redisClient, log)
	knokRepo := postgres.NewKnokRepository(db, log)
	serverRepo := postgres.NewServerRepository(db, log)

	// Create worker service
	workerService, err := worker.New(cfg, log, knokRepo, serverRepo, queueRepo)
	if err != nil {
		log.Error("Failed to create worker service", "error", err)
		os.Exit(1)
	}

	// Create a channel to track shutdown completion
	done := make(chan struct{})

	// Start worker service in a goroutine
	go func() {
		defer close(done)
		if err := workerService.Start(); err != nil {
			log.Error("Worker service failed", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either shutdown signal or service completion
	select {
	case <-quit:
		log.Info("Shutdown signal received, stopping worker service...")
	case <-done:
		log.Info("Worker service completed")
	}

	// Graceful shutdown with timeout
	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop worker service
	if err := workerService.Stop(); err != nil {
		log.Error("Error stopping worker service", "error", err)
	}

	log.Info("Worker service shutdown complete")
}
