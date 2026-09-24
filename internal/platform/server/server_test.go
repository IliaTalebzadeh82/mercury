//go:build integration

package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/IliaTalebzadeh82/mercury/internal/platform/database"
)

func TestOperationalEndpointsWithPostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("MERCURY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("MERCURY_TEST_DATABASE_URL is required for PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("database.Open() unexpected error: %v", err)
	}
	t.Cleanup(pool.Close)

	application := New(time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)), pool)

	t.Run("liveness does not depend on database", func(t *testing.T) {
		response := performRequest(t, application.Handler(), "/healthz")
		assertResponse(t, response, http.StatusOK, `{"status":"ok"}`)
	})

	t.Run("readiness reports healthy database", func(t *testing.T) {
		response := performRequest(t, application.Handler(), "/readyz")
		assertResponse(t, response, http.StatusOK, `{"status":"ready"}`)
	})

	pool.Close()

	t.Run("readiness reports a closed pool", func(t *testing.T) {
		response := performRequest(t, application.Handler(), "/readyz")
		assertResponse(t, response, http.StatusServiceUnavailable, `{"status":"database_unavailable"}`)
	})

	t.Run("liveness remains healthy after database loss", func(t *testing.T) {
		response := performRequest(t, application.Handler(), "/healthz")
		assertResponse(t, response, http.StatusOK, `{"status":"ok"}`)
	})
}

func TestShutdownWithdrawsReadinessAndStopsListener(t *testing.T) {
	databaseURL := os.Getenv("MERCURY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("MERCURY_TEST_DATABASE_URL is required for PostgreSQL integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("database.Open() unexpected error: %v", err)
	}
	t.Cleanup(pool.Close)

	application := New(time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)), pool)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() unexpected error: %v", err)
	}
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- application.Serve(listener) }()

	endpoint := "http://" + listener.Addr().String() + "/healthz"
	httpResponse, err := http.Get(endpoint)
	if err != nil {
		t.Fatalf("GET before shutdown failed: %v", err)
	}
	_ = httpResponse.Body.Close()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), time.Second)
	defer cancelShutdown()
	startedAt := time.Now()
	if err := application.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() unexpected error: %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed >= time.Second {
		t.Fatalf("Shutdown() exceeded its bound: %s", elapsed)
	}
	if err := <-serveErrors; err != http.ErrServerClosed {
		t.Fatalf("Serve() error = %v, want http.ErrServerClosed", err)
	}
	if _, err := http.Get(endpoint); err == nil {
		t.Fatal("listener still accepted traffic after shutdown")
	}

	response := performRequest(t, application.Handler(), "/readyz")
	assertResponse(t, response, http.StatusServiceUnavailable, `{"status":"not_ready"}`)
}

func assertResponse(t *testing.T, response *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
	}
	if got := response.Body.String(); got != body+"\n" {
		t.Fatalf("body = %q, want %q", got, body+"\n")
	}
}

func performRequest(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
