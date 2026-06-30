package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds the application configuration.
type Config struct {
	Port              string
	AppEnv            string
	GinMode           string
	DatabaseURL       string
	DirectDatabaseURL string
	RedisURL          string
	AdminToken        string
}

// Load loads the configuration from environment variables.
// It optionally loads from a .env file if it exists.
func Load() (*Config, error) {
	// Load .env file if it exists (useful for local development)
	_ = godotenv.Load()

	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		AppEnv:            getEnv("APP_ENV", "development"),
		GinMode:           getEnv("GIN_MODE", "debug"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		DirectDatabaseURL: os.Getenv("DIRECT_DATABASE_URL"),
		RedisURL:          os.Getenv("REDIS_URL"),
		AdminToken:        os.Getenv("ADMIN_TOKEN"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks if all required configuration options are set.
func (c *Config) Validate() error {
	var missing []string

	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if c.DirectDatabaseURL == "" {
		missing = append(missing, "DIRECT_DATABASE_URL")
	}
	if c.RedisURL == "" {
		missing = append(missing, "REDIS_URL")
	}
	if c.AdminToken == "" {
		missing = append(missing, "ADMIN_TOKEN")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// IsProduction returns true if the application environment is production.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// IsDevelopment returns true if the application environment is development.
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development" || c.AppEnv == ""
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
