package redis

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewClient creates a new Redis client from a Redis URL
func NewClient(redisURL string, logger *slog.Logger) (*redis.Client, error) {
	// Parse Redis URL
	parsedURL, err := url.Parse(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Redis URL: %w", err)
	}

	// Extract connection details
	addr := parsedURL.Host
	if addr == "" {
		addr = "localhost:6379"
	}

	password := ""
	if parsedURL.User != nil {
		password, _ = parsedURL.User.Password()
	}

	db := 0
	if parsedURL.Path != "" && len(parsedURL.Path) > 1 {
		if dbNum, err := strconv.Atoi(parsedURL.Path[1:]); err == nil {
			db = dbNum
		}
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		PoolTimeout:  30 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Connected to Redis",
		"addr", addr,
		"db", db,
	)

	return client, nil
}

// HealthCheck performs a health check on the Redis connection
func HealthCheck(ctx context.Context, client *redis.Client) error {
	return client.Ping(ctx).Err()
}
