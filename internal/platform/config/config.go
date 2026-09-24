package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

const (
	defaultHTTPAddress        = ":8080"
	defaultDatabasePing       = 2 * time.Second
	defaultShutdownTimeout    = 10 * time.Second
	defaultLogLevel           = "info"
	minimumOperationalTimeout = 100 * time.Millisecond
)

type Config struct {
	HTTPAddress         string
	DatabaseURL         string
	DatabasePingTimeout time.Duration
	ShutdownTimeout     time.Duration
	LogLevel            slog.Level
}

func Load() (Config, error) {
	databaseURL := strings.TrimSpace(os.Getenv("MERCURY_DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, fmt.Errorf("MERCURY_DATABASE_URL is required")
	}

	httpAddress := valueOrDefault("MERCURY_HTTP_ADDRESS", defaultHTTPAddress)
	databasePingTimeout, err := duration("MERCURY_DATABASE_PING_TIMEOUT", defaultDatabasePing)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := duration("MERCURY_SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	if err != nil {
		return Config{}, err
	}
	logLevel, err := parseLogLevel(valueOrDefault("MERCURY_LOG_LEVEL", defaultLogLevel))
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddress:         httpAddress,
		DatabaseURL:         databaseURL,
		DatabasePingTimeout: databasePingTimeout,
		ShutdownTimeout:     shutdownTimeout,
		LogLevel:            logLevel,
	}, nil
}

func valueOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func duration(name string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration: %w", name, err)
	}
	if value < minimumOperationalTimeout {
		return 0, fmt.Errorf("%s must be at least %s", name, minimumOperationalTimeout)
	}
	return value, nil
}

func parseLogLevel(raw string) (slog.Level, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.ToLower(raw))); err != nil {
		return 0, fmt.Errorf("MERCURY_LOG_LEVEL is invalid: %w", err)
	}
	return level, nil
}
