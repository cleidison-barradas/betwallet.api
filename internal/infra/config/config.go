package config

import (
	"os"
	"time"

	"go.uber.org/fx"
)

type Config struct {
	Port           string
	DatabaseURL    string
	OIDCIssuer     string
	DatabaseConfig DatabaseConfig
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
}

type DatabaseConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	PingTimeout     time.Duration
}

func Load() Config {
	return Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", ""),
		OIDCIssuer:   getEnv("OIDC_ISSUER", ""),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var Module = fx.Module("config", fx.Provide(Load))
