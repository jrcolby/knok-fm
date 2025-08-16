package config

import (
	"flag"
	"log"
	"os"
)

type Config struct {
	Port         string
	DatabaseURL  string
	RedisURL     string
	DiscordToken string
	LogLevel     string
	StaticDir    string
}

func Load() *Config {
	config := &Config{
		Port:      getEnvWithDefault("PORT", "8080"),
		LogLevel:  getEnvWithDefault("LOG_LEVEL", "info"),
		StaticDir: getEnvWithDefault("STATIC_DIR", "./web/build"),
	}

	// Required environment variables (for database/redis services)
	config.DatabaseURL = mustGetEnv("DATABASE_URL")
	config.RedisURL = mustGetEnv("REDIS_URL")

	// Optional Discord token (only required for bot service)
	config.DiscordToken = getEnvWithDefault("DISCORD_TOKEN", "")

	// Command line flags override environment
	flag.StringVar(&config.Port, "port", config.Port, "Server port")
	flag.StringVar(&config.LogLevel, "log-level", config.LogLevel, "Log level")
	flag.Parse()

	return config
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is required", key)
	}
	return value
}

// ValidateForBot ensures all required fields for bot service are present
func (c *Config) ValidateForBot() error {
	if c.DiscordToken == "" {
		log.Fatalf("Environment variable DISCORD_TOKEN is required for bot service")
	}
	return nil
}

// ValidateForWorker ensures all required fields for worker service are present
func (c *Config) ValidateForWorker() error {
	// Worker only needs database and Redis, no Discord token required
	return nil
}

// ValidateForAPI ensures all required fields for API service are present
func (c *Config) ValidateForAPI() error {
	// API only needs basic config, no Discord token required
	return nil
}
