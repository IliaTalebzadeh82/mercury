//go:build integration

package advertiser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IliaTalebzadeh82/mercury/internal/platform/database"
)

var advertiserSequence atomic.Int64

func TestConcurrentAdvertiserCreationRetriesShareOneResource(t *testing.T) {
	store := integrationStore(t)
	key := advertiserKey("equivalent")
	start := make(chan struct{})
	results := make(chan CreateResult, 2)
	errorsSeen := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for _, name := range []string{" Equivalent advertiser ", "Equivalent advertiser"} {
		name := name
		go func() {
			ready.Done()
			<-start
			result, err := store.Create(context.Background(), CreateCommand{Name: name, IdempotencyKey: key})
			results <- result
			errorsSeen <- err
		}()
	}
	ready.Wait()
	close(start)
	first, second := <-results, <-results
	if err := <-errorsSeen; err != nil {
		t.Fatal(err)
	}
	if err := <-errorsSeen; err != nil {
		t.Fatal(err)
	}
	if first.Advertiser.ID != second.Advertiser.ID {
		t.Fatalf("IDs differ: %s != %s", first.Advertiser.ID, second.Advertiser.ID)
	}
	if first.Replayed == second.Replayed {
		t.Fatal("expected exactly one replay")
	}
}

func TestConcurrentAdvertiserIdempotencyConflictRejectsOneRequest(t *testing.T) {
	store := integrationStore(t)
	key := advertiserKey("conflict")
	start := make(chan struct{})
	errorsSeen := make(chan error, 2)
	for _, name := range []string{"First advertiser", "Second advertiser"} {
		name := name
		go func() {
			<-start
			_, err := store.Create(context.Background(), CreateCommand{Name: name, IdempotencyKey: key})
			errorsSeen <- err
		}()
	}
	close(start)
	var success, conflict int
	for range 2 {
		err := <-errorsSeen
		if err == nil {
			success++
		} else if errors.Is(err, ErrIdempotencyConflict) {
			conflict++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflict)
	}
}

func integrationStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("MERCURY_TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("MERCURY_TEST_DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, url)
	if err != nil {
		t.Fatalf("database.Open(): %v", err)
	}
	t.Cleanup(pool.Close)
	return NewStore(pool, 5*time.Second)
}

func advertiserKey(prefix string) string {
	return fmt.Sprintf("advertiser-%s-%d-%d", prefix, time.Now().UnixNano(), advertiserSequence.Add(1))
}
