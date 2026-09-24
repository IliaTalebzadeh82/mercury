//go:build integration

package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestOpenConnectsToPostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("MERCURY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("MERCURY_TEST_DATABASE_URL is required for PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Open() unexpected error: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Ping() unexpected error: %v", err)
	}
	if got := pool.Config().MaxConns; got != maxConnections {
		t.Fatalf("MaxConns = %d, want %d", got, maxConnections)
	}
}
