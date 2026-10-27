// Package config reads the process environment once, at startup. It is the
// only package that calls os.Getenv; everything else receives values from
// the composition root (cmd/wire.go).
package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// Config holds everything read from the environment. Secrets live in the
// environment (or a secret manager), never in code or in git.
type Config struct {
	AppEnv         string
	LogLevel       string
	LogFormat      string
	HTTPAddr       string
	DBDriver       string
	DBDSN          string
	RequestTimeout time.Duration
}

// Load reads the environment, applies defaults and reports every missing or
// invalid value at once.
func Load() (Config, error) {
	var errs []error

	cfg := Config{
		AppEnv:    getEnv("APP_ENV", "development"),
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "text"),
		HTTPAddr:  getEnv("HTTP_ADDR", ":8080"),
		DBDriver:  getEnv("DB_DRIVER", "sqlite3"),
		DBDSN:     os.Getenv("DB_DSN"),
	}

	timeout, err := time.ParseDuration(getEnv("REQUEST_TIMEOUT", "30s"))
	if err != nil {
		errs = append(errs, fmt.Errorf("REQUEST_TIMEOUT: %w", err))
	}
	cfg.RequestTimeout = timeout

	if cfg.DBDSN == "" {
		errs = append(errs, errors.New("DB_DSN is not set"))
	}

	return cfg, errors.Join(errs...)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
