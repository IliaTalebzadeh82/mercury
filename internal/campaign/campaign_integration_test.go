//go:build integration

package campaign

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IliaTalebzadeh82/mercury/internal/advertiser"
	"github.com/IliaTalebzadeh82/mercury/internal/platform/database"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var integrationSequence atomic.Int64

func TestDatabaseEnforcesCountryCardinalityAtCommit(t *testing.T) {
	pool, campaigns, advertiserID := integrationStores(t)
	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO campaigns (advertiser_id,name,placement_code,budget_amount_minor,currency,creation_idempotency_key,creation_request_fingerprint)
		VALUES ($1,'No targets','home_feed',100,'EUR',$2,decode(repeat('01',32),'hex'))`, advertiserID, unique("missing-target"))
	if err != nil {
		t.Fatal(err)
	}
	err = tx.Commit(ctx)
	assertConstraintViolation(t, err)

	created := createCampaign(t, campaigns, advertiserID, unique("valid-target"))
	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM campaign_target_countries WHERE campaign_id=$1`, created.ID); err != nil {
		t.Fatal(err)
	}
	err = tx.Commit(ctx)
	assertConstraintViolation(t, err)

	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM campaign_target_countries WHERE campaign_id=$1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO campaign_target_countries (campaign_id,country_code) VALUES ($1,'GB')`, created.ID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("atomic target replacement failed: %v", err)
	}
}

func TestConcurrentCountryRemovalPreservesMinimum(t *testing.T) {
	pool, campaigns, advertiserID := integrationStores(t)
	created := createCampaign(t, campaigns, advertiserID, unique("country-removal"))
	if _, err := pool.Exec(context.Background(), `INSERT INTO campaign_target_countries (campaign_id,country_code) VALUES ($1,'FR')`, created.ID); err != nil {
		t.Fatalf("insert second country: %v", err)
	}

	start := make(chan struct{})
	errorsSeen := make(chan error, 2)
	for _, country := range []string{"DE", "FR"} {
		country := country
		go func() {
			tx, err := pool.Begin(context.Background())
			if err != nil {
				errorsSeen <- err
				return
			}
			defer rollback(tx)
			<-start
			if _, err := tx.Exec(context.Background(), `DELETE FROM campaign_target_countries WHERE campaign_id=$1 AND country_code=$2`, created.ID, country); err != nil {
				errorsSeen <- err
				return
			}
			errorsSeen <- tx.Commit(context.Background())
		}()
	}
	close(start)

	var succeeded, failed int
	for range 2 {
		if err := <-errorsSeen; err == nil {
			succeeded++
		} else {
			failed++
		}
	}
	if succeeded != 1 || failed != 1 {
		t.Fatalf("concurrent removals: succeeded=%d failed=%d, want one each", succeeded, failed)
	}
	var remaining int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM campaign_target_countries WHERE campaign_id=$1`, created.ID).Scan(&remaining); err != nil {
		t.Fatalf("count remaining countries: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("remaining countries=%d, want 1", remaining)
	}
}

func TestConcurrentCommandsWithSameVersionPermitOneMutation(t *testing.T) {
	_, campaigns, advertiserID := integrationStores(t)
	created := createCampaign(t, campaigns, advertiserID, unique("same-version"))
	errorsSeen := concurrentErrors(
		func() error {
			_, err := campaigns.UpdateName(context.Background(), created.ID, created.Version, "First writer")
			return err
		},
		func() error {
			_, err := campaigns.UpdateName(context.Background(), created.ID, created.Version, "Second writer")
			return err
		},
	)
	assertOneSuccessOnePrecondition(t, errorsSeen)
}

func TestLifecycleTransitionRacingConfigurationUpdate(t *testing.T) {
	_, campaigns, advertiserID := integrationStores(t)
	created := createCampaign(t, campaigns, advertiserID, unique("lifecycle-config"))
	errorsSeen := concurrentErrors(
		func() error {
			_, err := campaigns.Activate(context.Background(), created.ID, created.Version)
			return err
		},
		func() error {
			_, err := campaigns.UpdatePlacement(context.Background(), created.ID, created.Version, "search_results")
			return err
		},
	)
	assertOneSuccessOnePrecondition(t, errorsSeen)
}

func TestConcurrentLifecycleCommandsPermitOneTransition(t *testing.T) {
	_, campaigns, advertiserID := integrationStores(t)
	created := createCampaign(t, campaigns, advertiserID, unique("lifecycle-race"))
	active, err := campaigns.Activate(context.Background(), created.ID, created.Version)
	if err != nil {
		t.Fatal(err)
	}
	errorsSeen := concurrentErrors(
		func() error { _, err := campaigns.Pause(context.Background(), active.ID, active.Version); return err },
		func() error { _, err := campaigns.End(context.Background(), active.ID, active.Version); return err },
	)
	assertOneSuccessOnePrecondition(t, errorsSeen)
}

func TestConcurrentEquivalentCreationIsIdempotent(t *testing.T) {
	_, campaigns, advertiserID := integrationStores(t)
	key := unique("same-create")
	start := make(chan struct{})
	results := make(chan CreateResult, 2)
	errorsSeen := make(chan error, 2)
	commands := []CreateCommand{
		{AdvertiserID: advertiserID, Name: "Equivalent", PlacementCode: "HOME_FEED", AmountMinor: 100, Currency: "eur", Countries: []string{"FR", "de", "FR"}, IdempotencyKey: key},
		{AdvertiserID: advertiserID, Name: "Equivalent", PlacementCode: "home_feed", AmountMinor: 100, Currency: "EUR", Countries: []string{"DE", "FR"}, IdempotencyKey: key},
	}
	for _, command := range commands {
		command := command
		go func() {
			<-start
			result, err := campaigns.Create(context.Background(), command)
			results <- result
			errorsSeen <- err
		}()
	}
	close(start)
	first, second := <-results, <-results
	if err := <-errorsSeen; err != nil {
		t.Fatal(err)
	}
	if err := <-errorsSeen; err != nil {
		t.Fatal(err)
	}
	if first.Campaign.ID != second.Campaign.ID {
		t.Fatalf("creation IDs differ: %s != %s", first.Campaign.ID, second.Campaign.ID)
	}
	if first.Replayed == second.Replayed {
		t.Fatalf("replay flags = %v and %v, want exactly one replay", first.Replayed, second.Replayed)
	}
}

func TestConcurrentConflictingCreationRejectsOneCommand(t *testing.T) {
	_, campaigns, advertiserID := integrationStores(t)
	key := unique("conflicting-create")
	errorsSeen := concurrentErrors(
		func() error {
			_, err := campaigns.Create(context.Background(), CreateCommand{AdvertiserID: advertiserID, Name: "Alpha", PlacementCode: "home_feed", AmountMinor: 100, Currency: "EUR", Countries: []string{"DE"}, IdempotencyKey: key})
			return err
		},
		func() error {
			_, err := campaigns.Create(context.Background(), CreateCommand{AdvertiserID: advertiserID, Name: "Beta", PlacementCode: "home_feed", AmountMinor: 100, Currency: "EUR", Countries: []string{"DE"}, IdempotencyKey: key})
			return err
		},
	)
	var success, conflict int
	for _, err := range errorsSeen {
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

func TestLifecycleAndEditabilityRules(t *testing.T) {
	_, campaigns, advertiserID := integrationStores(t)
	created := createCampaign(t, campaigns, advertiserID, unique("rules"))
	active, err := campaigns.Activate(context.Background(), created.ID, created.Version)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := campaigns.UpdateBudget(context.Background(), active.ID, active.Version, 99); err == nil {
		t.Fatal("active budget decrease succeeded")
	}
	if _, err := campaigns.UpdatePlacement(context.Background(), active.ID, active.Version, "search_results"); err == nil {
		t.Fatal("active placement update succeeded")
	}
	paused, err := campaigns.Pause(context.Background(), active.ID, active.Version)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := campaigns.UpdateBudget(context.Background(), paused.ID, paused.Version, 50)
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := campaigns.Resume(context.Background(), updated.ID, updated.Version)
	if err != nil {
		t.Fatal(err)
	}
	ended, err := campaigns.End(context.Background(), resumed.ID, resumed.Version)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := campaigns.UpdateName(context.Background(), ended.ID, ended.Version, "Forbidden"); err == nil {
		t.Fatal("ended campaign mutation succeeded")
	}
}

func integrationStores(t *testing.T) (*pgxpool.Pool, *Store, string) {
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
	advertisers := advertiser.NewStore(pool, 5*time.Second)
	result, err := advertisers.Create(context.Background(), advertiser.CreateCommand{Name: "Integration advertiser", IdempotencyKey: unique("advertiser")})
	if err != nil {
		t.Fatalf("create advertiser: %v", err)
	}
	return pool, NewStore(pool, 5*time.Second), result.Advertiser.ID
}

func createCampaign(t *testing.T, store *Store, advertiserID, key string) Campaign {
	t.Helper()
	result, err := store.Create(context.Background(), CreateCommand{AdvertiserID: advertiserID, Name: "Campaign", PlacementCode: "home_feed", AmountMinor: 100, Currency: "EUR", Countries: []string{"DE"}, IdempotencyKey: key})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	return result.Campaign
}

func unique(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), integrationSequence.Add(1))
}

func concurrentErrors(commands ...func() error) []error {
	start := make(chan struct{})
	errorsSeen := make(chan error, len(commands))
	var ready sync.WaitGroup
	ready.Add(len(commands))
	for _, command := range commands {
		command := command
		go func() { ready.Done(); <-start; errorsSeen <- command() }()
	}
	ready.Wait()
	close(start)
	result := make([]error, 0, len(commands))
	for range commands {
		result = append(result, <-errorsSeen)
	}
	return result
}

func assertOneSuccessOnePrecondition(t *testing.T, values []error) {
	t.Helper()
	var success, stale int
	for _, err := range values {
		if err == nil {
			success++
		} else if errors.Is(err, ErrPreconditionFailed) {
			stale++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatalf("success=%d stale=%d", success, stale)
	}
}

func assertConstraintViolation(t *testing.T, err error) {
	t.Helper()
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) || pgError.Code != "23514" {
		t.Fatalf("error = %v, want PostgreSQL check violation", err)
	}
}
