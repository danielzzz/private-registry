package config_test

import (
	"testing"
	"time"

	"github.com/danielzelisko/private-registry/internal/config"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_PATH", "/data/registry.db")
	t.Setenv("REGISTRY_SERVICE", "registry.example.test")
	t.Setenv("TOKEN_ISSUER", "https://auth.example.test")
	t.Setenv("TOKEN_CERT_PATH", "/data/cert.pem")
	t.Setenv("TOKEN_KEY_PATH", "/data/key.pem")
}

func TestLoadDefaults(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("TOKEN_TTL", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr=%q", cfg.HTTPAddr)
	}
	if cfg.TokenTTL != 300*time.Second {
		t.Fatalf("TokenTTL=%v", cfg.TokenTTL)
	}
	if cfg.DatabasePath != "/data/registry.db" {
		t.Fatalf("DatabasePath=%q", cfg.DatabasePath)
	}
}

func TestLoadRequiresDatabasePath(t *testing.T) {
	t.Setenv("DATABASE_PATH", "")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_PATH")
	}
}

func TestLoadTokenTTLOverride(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("TOKEN_TTL", "600")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TokenTTL != 600*time.Second {
		t.Fatalf("TokenTTL=%v", cfg.TokenTTL)
	}
}

func TestLoadSessionSecret(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("SESSION_SECRET", "super-secret")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SessionSecret != "super-secret" {
		t.Fatalf("SessionSecret=%q", cfg.SessionSecret)
	}
}
