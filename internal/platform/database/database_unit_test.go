package database

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestInvalidConfigurationDoesNotExposeSecrets(t *testing.T) {
	const (
		passwordSentinel = "user-info-secret-sentinel"
		querySentinel    = "query-secret-sentinel"
	)
	databaseURL := "postgres://mercury:" + passwordSentinel + "@localhost/mercury?connect_timeout=invalid&sslpassword=" + querySentinel

	_, err := Open(context.Background(), databaseURL)
	if !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("Open() error = %v, want ErrInvalidConfiguration", err)
	}
	for _, secret := range []string{passwordSentinel, querySentinel} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("Open() error exposed sentinel secret %q", secret)
		}
	}

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	logger.Error("database startup failed", "error", err)
	for _, secret := range []string{passwordSentinel, querySentinel} {
		if strings.Contains(output.String(), secret) {
			t.Fatalf("structured log exposed sentinel secret %q", secret)
		}
	}
}
