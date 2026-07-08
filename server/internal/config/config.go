// Package config loads coordinator configuration from the environment.
// Credentials are never hardcoded (FR-030) — they must come from the
// environment/secret store at deploy time.
package config

import (
	"fmt"
	"os"
)

// Config holds all environment-derived settings for the coordinator.
type Config struct {
	// DBPath is the filesystem path to the SQLite database file.
	DBPath string
	// MigrationsDir is the directory containing *.sql migration files.
	MigrationsDir string
	// ListenAddr is the address the HTTP server listens on, e.g. ":8080".
	ListenAddr string
	// OperatorToken is the bearer token required to trigger rounds (FR-029).
	OperatorToken string
	// FCMServiceAccountJSON is the path to the Firebase service-account
	// credentials file used to send FCM messages.
	FCMServiceAccountJSON string
}

// Load reads configuration from environment variables, applying sensible
// defaults for local development and returning an error for anything that
// is required but missing.
func Load() (Config, error) {
	cfg := Config{
		DBPath:                getEnvDefault("DB_PATH", "./coordinator.db"),
		MigrationsDir:         getEnvDefault("MIGRATIONS_DIR", "migrations"),
		ListenAddr:            getEnvDefault("LISTEN_ADDR", ":8080"),
		OperatorToken:         os.Getenv("OPERATOR_TOKEN"),
		FCMServiceAccountJSON: os.Getenv("FCM_SERVICE_ACCOUNT_JSON"),
	}

	if cfg.OperatorToken == "" {
		return Config{}, fmt.Errorf("OPERATOR_TOKEN environment variable is required")
	}
	if cfg.FCMServiceAccountJSON == "" {
		return Config{}, fmt.Errorf("FCM_SERVICE_ACCOUNT_JSON environment variable is required")
	}

	return cfg, nil
}

func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
