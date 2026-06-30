package config

import (
	"testing"
)

func TestConfig_Load(t *testing.T) {
	// Setup required env vars
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/db")
	t.Setenv("DIRECT_DATABASE_URL", "postgres://localhost:5433/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("ADMIN_TOKEN", "super-secret")
	t.Setenv("PORT", "9090")
	t.Setenv("APP_ENV", "testing")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("expected Port to be 9090, got %s", cfg.Port)
	}
	if cfg.AppEnv != "testing" {
		t.Errorf("expected AppEnv to be testing, got %s", cfg.AppEnv)
	}
	if cfg.DatabaseURL != "postgres://localhost:5432/db" {
		t.Errorf("expected DatabaseURL to be set correctly")
	}
	if cfg.IsProduction() {
		t.Errorf("expected IsProduction to be false")
	}
	if cfg.IsDevelopment() {
		t.Errorf("expected IsDevelopment to be true for testing env (considered dev-like)")
	}
}

func TestConfig_Validation(t *testing.T) {
	// Clear environment to test validation failure
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DIRECT_DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("ADMIN_TOKEN", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error due to missing required env vars, got nil")
	}
}
