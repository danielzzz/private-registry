package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultHTTPAddr = ":8080"
	defaultTokenTTL = 300 * time.Second
)

type Config struct {
	DatabaseDriver  string
	DatabasePath    string
	DatabaseDSN     string
	HTTPAddr        string
	RegistryService string
	TokenIssuer     string
	TokenCertPath   string
	TokenKeyPath    string
	TokenTTL        time.Duration
	AdminUser       string
	AdminPassword   string
	SessionSecret   string
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseDriver:  envOrDefault("DATABASE_DRIVER", "sqlite"),
		DatabasePath:    os.Getenv("DATABASE_PATH"),
		DatabaseDSN:     os.Getenv("DATABASE_DSN"),
		HTTPAddr:        envOrDefault("HTTP_ADDR", defaultHTTPAddr),
		RegistryService: os.Getenv("REGISTRY_SERVICE"),
		TokenIssuer:     os.Getenv("TOKEN_ISSUER"),
		TokenCertPath:   os.Getenv("TOKEN_CERT_PATH"),
		TokenKeyPath:    os.Getenv("TOKEN_KEY_PATH"),
		AdminUser:       os.Getenv("ADMIN_USER"),
		AdminPassword:   os.Getenv("ADMIN_PASSWORD"),
		SessionSecret:   os.Getenv("SESSION_SECRET"),
	}

	switch cfg.DatabaseDriver {
	case "sqlite":
		if cfg.DatabasePath == "" {
			return Config{}, fmt.Errorf("DATABASE_PATH is required when DATABASE_DRIVER=sqlite")
		}
	case "mysql":
		if cfg.DatabaseDSN == "" {
			return Config{}, fmt.Errorf("DATABASE_DSN is required when DATABASE_DRIVER=mysql")
		}
	default:
		return Config{}, fmt.Errorf("DATABASE_DRIVER must be sqlite or mysql, got %q", cfg.DatabaseDriver)
	}
	if cfg.RegistryService == "" {
		return Config{}, fmt.Errorf("REGISTRY_SERVICE is required")
	}
	if cfg.TokenIssuer == "" {
		return Config{}, fmt.Errorf("TOKEN_ISSUER is required")
	}
	if cfg.TokenCertPath == "" {
		return Config{}, fmt.Errorf("TOKEN_CERT_PATH is required")
	}
	if cfg.TokenKeyPath == "" {
		return Config{}, fmt.Errorf("TOKEN_KEY_PATH is required")
	}

	ttlStr := os.Getenv("TOKEN_TTL")
	if ttlStr == "" {
		cfg.TokenTTL = defaultTokenTTL
	} else {
		seconds, err := strconv.Atoi(ttlStr)
		if err != nil || seconds <= 0 {
			return Config{}, fmt.Errorf("TOKEN_TTL must be a positive integer of seconds")
		}
		cfg.TokenTTL = time.Duration(seconds) * time.Second
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
