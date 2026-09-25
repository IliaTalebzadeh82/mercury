package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	cleanMercuryEnvironment(t)

	_, err := Load()

	if err == nil || err.Error() != "MERCURY_DATABASE_URL is required" {
		t.Fatalf("Load() error = %v, want required database URL error", err)
	}
}

func TestLoadDefaults(t *testing.T) {
	cleanMercuryEnvironment(t)
	t.Setenv("MERCURY_DATABASE_URL", "postgres://mercury:secret@localhost/mercury")

	cfg, err := Load()

	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.HTTPAddress != ":8080" || cfg.DatabasePingTimeout != 2*time.Second || cfg.DatabaseOperationTimeout != 3*time.Second || cfg.ShutdownTimeout != 10*time.Second || cfg.LogLevel != slog.LevelInfo || cfg.DiagnosticAPIEnabled {
		t.Fatalf("Load() did not return the expected non-secret defaults")
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "malformed database timeout", key: "MERCURY_DATABASE_PING_TIMEOUT", value: "soon"},
		{name: "database timeout too short", key: "MERCURY_DATABASE_PING_TIMEOUT", value: "10ms"},
		{name: "malformed operation timeout", key: "MERCURY_DATABASE_OPERATION_TIMEOUT", value: "eventually"},
		{name: "operation timeout too short", key: "MERCURY_DATABASE_OPERATION_TIMEOUT", value: "10ms"},
		{name: "malformed shutdown timeout", key: "MERCURY_SHUTDOWN_TIMEOUT", value: "later"},
		{name: "unknown log level", key: "MERCURY_LOG_LEVEL", value: "verbose"},
		{name: "invalid diagnostic API flag", key: "MERCURY_DIAGNOSTIC_API_ENABLED", value: "sometimes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanMercuryEnvironment(t)
			t.Setenv("MERCURY_DATABASE_URL", "postgres://example")
			t.Setenv(tt.key, tt.value)
			_, err := Load()
			if err == nil {
				t.Fatal("Load() succeeded, want validation error")
			}
		})
	}
}

func cleanMercuryEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"MERCURY_DATABASE_URL",
		"MERCURY_HTTP_ADDRESS",
		"MERCURY_DATABASE_PING_TIMEOUT",
		"MERCURY_DATABASE_OPERATION_TIMEOUT",
		"MERCURY_SHUTDOWN_TIMEOUT",
		"MERCURY_LOG_LEVEL",
		"MERCURY_DIAGNOSTIC_API_ENABLED",
	} {
		t.Setenv(name, "")
	}
}
