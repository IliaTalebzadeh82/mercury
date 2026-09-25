//go:build integration

package budget

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IliaTalebzadeh82/mercury/internal/advertiser"
	"github.com/IliaTalebzadeh82/mercury/internal/campaign"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var integrationSequence atomic.Int64

func TestExactBudgetOneUnitOverAndMaxInt64(t *testing.T) {
	pool, campaigns, budgets, advertiserID := integrationStores(t)

	exact := activeCampaign(t, campaigns, advertiserID, 100, unique("exact"))
	approved := consume(t, budgets, exact.ID, 100, "EUR", unique("exact-key"))
	if approved.Outcome != Approved || approved.ResultingCommitted != 100 || approved.ResultingRemaining != 0 {
		t.Fatalf("exact-budget result: %+v", approved)
	}
	afterConsumption, err := campaigns.Get(context.Background(), exact.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterConsumption.Version != exact.Version || !afterConsumption.UpdatedAt.Equal(exact.UpdatedAt) {
		t.Fatalf("accounting contaminated configuration concurrency fields: before=%+v after=%+v", exact, afterConsumption)
	}
	rejected := consume(t, budgets, exact.ID, 1, "EUR", unique("over-key"))
	if rejected.Outcome != InsufficientBudget || rejected.ResultingCommitted != 100 || rejected.ResultingRemaining != 0 {
		t.Fatalf("one-unit-over result: %+v", rejected)
	}
	assertAccounting(t, pool, exact.ID, 100, 100)

	maximum := activeCampaign(t, campaigns, advertiserID, math.MaxInt64, unique("maximum"))
	first := consume(t, budgets, maximum.ID, math.MaxInt64-1, "EUR", unique("maximum-a"))
	last := consume(t, budgets, maximum.ID, 1, "EUR", unique("maximum-b"))
	over := consume(t, budgets, maximum.ID, 1, "EUR", unique("maximum-c"))
	if first.Outcome != Approved || last.Outcome != Approved || over.Outcome != InsufficientBudget {
		t.Fatalf("MaxInt64 outcomes: %s %s %s", first.Outcome, last.Outcome, over.Outcome)
	}
	assertAccounting(t, pool, maximum.ID, math.MaxInt64, math.MaxInt64)
}

func TestInactiveOutcomesArePersistedAndReplayStable(t *testing.T) {
	pool, campaigns, budgets, advertiserID := integrationStores(t)
	draft := draftCampaign(t, campaigns, advertiserID, 100, unique("inactive"))
	key := unique("inactive-key")
	first := consume(t, budgets, draft.ID, 10, "EUR", key)
	if first.Outcome != CampaignNotActive || first.Replayed {
		t.Fatalf("first result: %+v", first)
	}
	active, err := campaigns.Activate(context.Background(), draft.ID, draft.Version)
	if err != nil {
		t.Fatal(err)
	}
	replay := consume(t, budgets, active.ID, 10, "EUR", key)
	if replay.Outcome != CampaignNotActive || !replay.Replayed || replay.ConsumptionID != first.ConsumptionID {
		t.Fatalf("replay changed inactive outcome: %+v", replay)
	}
	assertAccounting(t, pool, active.ID, 100, 0)
}

func TestRepeatedTwentyBudgetRace(t *testing.T) {
	pool, campaigns, budgets, advertiserID := integrationStores(t)
	for iteration := 0; iteration < 20; iteration++ {
		value := activeCampaign(t, campaigns, advertiserID, 20, unique("twenty"))
		commands := []int64{8, 9, 10}
		start := make(chan struct{})
		results := make(chan Result, len(commands))
		errs := make(chan error, len(commands))
		var ready sync.WaitGroup
		ready.Add(len(commands))
		for index, amount := range commands {
			index, amount := index, amount
			go func() {
				ready.Done()
				<-start
				result, err := budgets.Consume(context.Background(), Command{
					CampaignID: value.ID, AmountMinor: amount, Currency: "EUR",
					IdempotencyKey: fmt.Sprintf("race-%d-%d-%s", iteration, index, unique("key")),
				})
				results <- result
				errs <- err
			}()
		}
		ready.Wait()
		close(start)
		var approvedCount, rejectedCount int
		var approvedSum int64
		for range commands {
			if err := <-errs; err != nil {
				t.Fatal(err)
			}
			result := <-results
			switch result.Outcome {
			case Approved:
				approvedCount++
				approvedSum += result.AmountMinor
			case InsufficientBudget:
				rejectedCount++
			default:
				t.Fatalf("unexpected outcome %s", result.Outcome)
			}
		}
		if approvedCount != 2 || rejectedCount != 1 || approvedSum > 20 {
			t.Fatalf("iteration %d approved=%d rejected=%d sum=%d", iteration, approvedCount, rejectedCount, approvedSum)
		}
		assertAccounting(t, pool, value.ID, 20, approvedSum)
	}
}

func TestManyHotConsumersAndIndependentCampaigns(t *testing.T) {
	pool, campaigns, budgets, advertiserID := integrationStores(t)
	firstConnection, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer firstConnection.Release()
	secondConnection, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer secondConnection.Release()
	if firstConnection.Conn().PgConn().PID() == secondConnection.Conn().PgConn().PID() {
		t.Fatal("expected independent PostgreSQL connections")
	}

	hot := activeCampaign(t, campaigns, advertiserID, 10, unique("hot"))
	results, errs := concurrentConsumes(budgets, 20, func(index int) Command {
		return Command{CampaignID: hot.ID, AmountMinor: 1, Currency: "EUR", IdempotencyKey: fmt.Sprintf("hot-%d-%s", index, unique("key"))}
	})
	assertOutcomeCounts(t, results, errs, 10, 10)
	assertAccounting(t, pool, hot.ID, 10, 10)

	values := make([]campaign.Campaign, 10)
	for index := range values {
		values[index] = activeCampaign(t, campaigns, advertiserID, 5, unique("spread"))
	}
	results, errs = concurrentConsumes(budgets, 50, func(index int) Command {
		return Command{CampaignID: values[index%len(values)].ID, AmountMinor: 1, Currency: "EUR", IdempotencyKey: fmt.Sprintf("spread-%d-%s", index, unique("key"))}
	})
	assertOutcomeCounts(t, results, errs, 50, 0)
	for _, value := range values {
		assertAccounting(t, pool, value.ID, 5, 5)
	}
}

func TestConcurrentSameKeyReplayAndConflict(t *testing.T) {
	pool, campaigns, budgets, advertiserID := integrationStores(t)
	value := activeCampaign(t, campaigns, advertiserID, 100, unique("same-key"))
	key := unique("shared")
	results, errs := concurrentConsumes(budgets, 12, func(int) Command {
		return Command{CampaignID: value.ID, AmountMinor: 7, Currency: "EUR", IdempotencyKey: key}
	})
	var originalID string
	var originals, replays int
	for index, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
		if originalID == "" {
			originalID = results[index].ConsumptionID
		}
		if results[index].ConsumptionID != originalID || results[index].Outcome != Approved {
			t.Fatalf("unstable replay: %+v", results[index])
		}
		if results[index].Replayed {
			replays++
		} else {
			originals++
		}
	}
	if originals != 1 || replays != 11 {
		t.Fatalf("originals=%d replays=%d", originals, replays)
	}
	assertAccounting(t, pool, value.ID, 100, 7)

	conflictKey := unique("conflict")
	results, errs = concurrentCommands(budgets, []Command{
		{CampaignID: value.ID, AmountMinor: 8, Currency: "EUR", IdempotencyKey: conflictKey},
		{CampaignID: value.ID, AmountMinor: 9, Currency: "EUR", IdempotencyKey: conflictKey},
	})
	var success, conflict int
	var added int64
	for index, err := range errs {
		if err == nil {
			success++
			added += results[index].AmountMinor
		} else if errors.Is(err, ErrIdempotencyConflict) {
			conflict++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflict)
	}
	assertAccounting(t, pool, value.ID, 100, 7+added)
}

func TestResponseLossAndTerminatedTransactionRetry(t *testing.T) {
	pool, campaigns, budgets, advertiserID := integrationStores(t)
	value := activeCampaign(t, campaigns, advertiserID, 100, unique("response-loss"))
	key := unique("response-key")
	first := consume(t, budgets, value.ID, 11, "EUR", key)
	replay := consume(t, budgets, value.ID, 11, "EUR", key)
	if !replay.Replayed || replay.ConsumptionID != first.ConsumptionID {
		t.Fatalf("lost response was not replayed: %+v %+v", first, replay)
	}

	url := os.Getenv("MERCURY_TEST_DATABASE_URL")
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := conn.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	terminatedKey := unique("terminated")
	command := Command{CampaignID: value.ID, AmountMinor: 13, Currency: "EUR", IdempotencyKey: terminatedKey}
	fingerprint := commandFingerprint(command)
	if _, err := tx.Exec(context.Background(), `SELECT 1 FROM campaigns WHERE id=$1 FOR UPDATE`, value.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(context.Background(), `UPDATE campaigns SET committed_spend_minor=committed_spend_minor+13 WHERE id=$1`, value.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(context.Background(), `
		INSERT INTO budget_consumption_commands
		(campaign_id,idempotency_key,request_fingerprint,amount_minor,currency,outcome,campaign_state_at_evaluation,
		 configured_budget_minor,committed_spend_before_minor,resulting_committed_spend_minor,resulting_remaining_budget_minor)
		VALUES ($1,$2,$3,13,'EUR','APPROVED','ACTIVE',100,11,24,76)`, value.ID, terminatedKey, fingerprint[:]); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	retried := consume(t, budgets, value.ID, 13, "EUR", terminatedKey)
	if retried.Outcome != Approved || retried.Replayed {
		t.Fatalf("terminated transaction left a receipt: %+v", retried)
	}
	assertAccounting(t, pool, value.ID, 100, 24)
}

func TestConsumptionSerializesWithLifecycleAndBudgetEdits(t *testing.T) {
	t.Run("pause", func(t *testing.T) {
		pool, campaigns, budgets, advertiserID := integrationStores(t)
		value := activeCampaign(t, campaigns, advertiserID, 100, unique("pause-race"))
		result, transitionErr := raceConsumptionAndCommand(budgets, value.ID, func() error {
			_, err := campaigns.Pause(context.Background(), value.ID, value.Version)
			return err
		})
		if transitionErr != nil || result.err != nil {
			t.Fatalf("pause=%v consume=%v", transitionErr, result.err)
		}
		if result.value.Outcome != Approved && result.value.Outcome != CampaignNotActive {
			t.Fatalf("outcome=%s", result.value.Outcome)
		}
		assertAccounting(t, pool, value.ID, 100, result.value.ResultingCommitted)
	})

	t.Run("end", func(t *testing.T) {
		pool, campaigns, budgets, advertiserID := integrationStores(t)
		value := activeCampaign(t, campaigns, advertiserID, 100, unique("end-race"))
		result, transitionErr := raceConsumptionAndCommand(budgets, value.ID, func() error {
			_, err := campaigns.End(context.Background(), value.ID, value.Version)
			return err
		})
		if transitionErr != nil || result.err != nil {
			t.Fatalf("end=%v consume=%v", transitionErr, result.err)
		}
		if result.value.Outcome != Approved && result.value.Outcome != CampaignNotActive {
			t.Fatalf("outcome=%s", result.value.Outcome)
		}
		assertAccounting(t, pool, value.ID, 100, result.value.ResultingCommitted)
	})

	t.Run("active increase", func(t *testing.T) {
		pool, campaigns, budgets, advertiserID := integrationStores(t)
		value := activeCampaign(t, campaigns, advertiserID, 10, unique("increase-race"))
		result, updateErr := raceConsumptionAndCommand(budgets, value.ID, func() error {
			_, err := campaigns.UpdateBudget(context.Background(), value.ID, value.Version, 20)
			return err
		})
		if updateErr != nil || result.err != nil {
			t.Fatalf("update=%v consume=%v", updateErr, result.err)
		}
		if result.value.Outcome != Approved && result.value.Outcome != InsufficientBudget {
			t.Fatalf("outcome=%s", result.value.Outcome)
		}
		assertAccounting(t, pool, value.ID, 20, result.value.ResultingCommitted)
	})

	t.Run("paused decrease", func(t *testing.T) {
		pool, campaigns, budgets, advertiserID := integrationStores(t)
		value := activeCampaign(t, campaigns, advertiserID, 100, unique("decrease-race"))
		consume(t, budgets, value.ID, 40, "EUR", unique("seed-spend"))
		paused, err := campaigns.Pause(context.Background(), value.ID, value.Version)
		if err != nil {
			t.Fatal(err)
		}
		result, updateErr := raceConsumptionAndCommand(budgets, value.ID, func() error {
			_, err := campaigns.UpdateBudget(context.Background(), value.ID, paused.Version, 50)
			return err
		})
		if updateErr != nil || result.err != nil || result.value.Outcome != CampaignNotActive {
			t.Fatalf("update=%v consume=%+v", updateErr, result)
		}
		assertAccounting(t, pool, value.ID, 50, 40)
	})
}

func TestLockWaitTimeoutAppendOnlyAndDatabaseConstraint(t *testing.T) {
	pool, campaigns, budgets, advertiserID := integrationStores(t)
	value := activeCampaign(t, campaigns, advertiserID, 100, unique("constraints"))

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(context.Background(), `SELECT 1 FROM campaigns WHERE id=$1 FOR UPDATE`, value.ID); err != nil {
		t.Fatal(err)
	}
	shortStore := NewStore(pool, 100*time.Millisecond)
	_, err = shortStore.Consume(context.Background(), Command{CampaignID: value.ID, AmountMinor: 1, Currency: "EUR", IdempotencyKey: unique("timeout")})
	_ = tx.Rollback(context.Background())
	if !errors.Is(err, ErrDatabase) {
		t.Fatalf("lock wait error=%v, want ErrDatabase", err)
	}
	assertAccounting(t, pool, value.ID, 100, 0)

	approved := consume(t, budgets, value.ID, 10, "EUR", unique("append-only"))
	for _, statement := range []string{
		`UPDATE budget_consumption_commands SET amount_minor=11 WHERE id=$1`,
		`DELETE FROM budget_consumption_commands WHERE id=$1`,
	} {
		_, err := pool.Exec(context.Background(), statement, approved.ConsumptionID)
		var pgError *pgconn.PgError
		if !errors.As(err, &pgError) || pgError.Code != "55000" {
			t.Fatalf("append-only error=%v", err)
		}
	}

	_, err = pool.Exec(context.Background(), `UPDATE campaigns SET committed_spend_minor=101 WHERE id=$1`, value.ID)
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) || pgError.Code != "23514" {
		t.Fatalf("counter constraint error=%v", err)
	}
	assertAccounting(t, pool, value.ID, 100, 10)
}

func TestContentionTimingEvidence(t *testing.T) {
	pool, campaigns, budgets, advertiserID := integrationStoresWithMaxConnections(t, 4)
	hot := activeCampaign(t, campaigns, advertiserID, 10_000, unique("timing-hot"))
	spread := make([]campaign.Campaign, 100)
	for index := range spread {
		spread[index] = activeCampaign(t, campaigns, advertiserID, 100, unique("timing-spread"))
	}

	for _, scenario := range []struct {
		name    string
		command func(int) Command
	}{
		{name: "hot", command: func(index int) Command {
			return Command{CampaignID: hot.ID, AmountMinor: 1, Currency: "EUR", IdempotencyKey: fmt.Sprintf("timing-hot-%d-%s", index, unique("key"))}
		}},
		{name: "spread", command: func(index int) Command {
			return Command{CampaignID: spread[index%len(spread)].ID, AmountMinor: 1, Currency: "EUR", IdempotencyKey: fmt.Sprintf("timing-spread-%d-%s", index, unique("key"))}
		}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			results, errs := concurrentConsumes(budgets, 2_000, scenario.command)
			locks := make([]time.Duration, len(results))
			transactions := make([]time.Duration, len(results))
			for index, err := range errs {
				if err != nil {
					t.Fatal(err)
				}
				locks[index] = results[index].CampaignLockWait
				transactions[index] = results[index].TransactionDuration
			}
			sort.Slice(locks, func(i, j int) bool { return locks[i] < locks[j] })
			sort.Slice(transactions, func(i, j int) bool { return transactions[i] < transactions[j] })
			t.Logf("requests=%d pool=4 lock_ms[p50=%.3f p95=%.3f p99=%.3f] transaction_ms[p50=%.3f p95=%.3f p99=%.3f]",
				len(results), milliseconds(percentile(locks, 50)), milliseconds(percentile(locks, 95)), milliseconds(percentile(locks, 99)),
				milliseconds(percentile(transactions, 50)), milliseconds(percentile(transactions, 95)), milliseconds(percentile(transactions, 99)))
		})
	}

	assertAccounting(t, pool, hot.ID, 10_000, 2_000)
	for _, value := range spread {
		assertAccounting(t, pool, value.ID, 100, 20)
	}
}

func percentile(values []time.Duration, percent int) time.Duration {
	index := (len(values)*percent + 99) / 100
	if index < 1 {
		index = 1
	}
	return values[index-1]
}

func milliseconds(value time.Duration) float64 {
	return float64(value) / float64(time.Millisecond)
}

type consumptionCall struct {
	value Result
	err   error
}

func raceConsumptionAndCommand(budgets *Store, campaignID string, command func() error) (consumptionCall, error) {
	start := make(chan struct{})
	consumeResult := make(chan consumptionCall, 1)
	commandResult := make(chan error, 1)
	var ready sync.WaitGroup
	ready.Add(2)
	go func() {
		ready.Done()
		<-start
		value, err := budgets.Consume(context.Background(), Command{CampaignID: campaignID, AmountMinor: 15, Currency: "EUR", IdempotencyKey: unique("state-race")})
		consumeResult <- consumptionCall{value: value, err: err}
	}()
	go func() {
		ready.Done()
		<-start
		commandResult <- command()
	}()
	ready.Wait()
	close(start)
	return <-consumeResult, <-commandResult
}

func concurrentConsumes(store *Store, count int, command func(int) Command) ([]Result, []error) {
	commands := make([]Command, count)
	for index := range commands {
		commands[index] = command(index)
	}
	return concurrentCommands(store, commands)
}

func concurrentCommands(store *Store, commands []Command) ([]Result, []error) {
	start := make(chan struct{})
	results := make([]Result, len(commands))
	errs := make([]error, len(commands))
	var ready sync.WaitGroup
	var done sync.WaitGroup
	ready.Add(len(commands))
	done.Add(len(commands))
	for index, command := range commands {
		index, command := index, command
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			results[index], errs[index] = store.Consume(context.Background(), command)
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
	return results, errs
}

func assertOutcomeCounts(t *testing.T, results []Result, errs []error, approved, insufficient int) {
	t.Helper()
	var approvedSeen, insufficientSeen int
	for index, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
		switch results[index].Outcome {
		case Approved:
			approvedSeen++
		case InsufficientBudget:
			insufficientSeen++
		default:
			t.Fatalf("unexpected outcome %s", results[index].Outcome)
		}
	}
	if approvedSeen != approved || insufficientSeen != insufficient {
		t.Fatalf("approved=%d insufficient=%d, want %d/%d", approvedSeen, insufficientSeen, approved, insufficient)
	}
}

func assertAccounting(t *testing.T, pool *pgxpool.Pool, campaignID string, configured, committed int64) {
	t.Helper()
	var storedConfigured, storedCommitted, remaining, ledger int64
	err := pool.QueryRow(context.Background(), `
		SELECT c.budget_amount_minor,c.committed_spend_minor,
			c.budget_amount_minor-c.committed_spend_minor,
			COALESCE(sum(b.amount_minor) FILTER (WHERE b.outcome='APPROVED'),0)::bigint
		FROM campaigns c LEFT JOIN budget_consumption_commands b ON b.campaign_id=c.id
		WHERE c.id=$1 GROUP BY c.id`, campaignID).
		Scan(&storedConfigured, &storedCommitted, &remaining, &ledger)
	if err != nil {
		t.Fatal(err)
	}
	if storedConfigured != configured || storedCommitted != committed || remaining != configured-committed || ledger != committed {
		t.Fatalf("accounting configured=%d committed=%d remaining=%d ledger=%d, want %d/%d/%d/%d",
			storedConfigured, storedCommitted, remaining, ledger, configured, committed, configured-committed, committed)
	}
}

func consume(t *testing.T, store *Store, campaignID string, amount int64, currency, key string) Result {
	t.Helper()
	result, err := store.Consume(context.Background(), Command{CampaignID: campaignID, AmountMinor: amount, Currency: currency, IdempotencyKey: key})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func integrationStores(t *testing.T) (*pgxpool.Pool, *campaign.Store, *Store, string) {
	return integrationStoresWithMaxConnections(t, 32)
}

func integrationStoresWithMaxConnections(t *testing.T, maxConnections int32) (*pgxpool.Pool, *campaign.Store, *Store, string) {
	t.Helper()
	url := os.Getenv("MERCURY_TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("MERCURY_TEST_DATABASE_URL is required")
	}
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns = maxConnections
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	advertisers := advertiser.NewStore(pool, 5*time.Second)
	advertiserResult, err := advertisers.Create(context.Background(), advertiser.CreateCommand{Name: "Budget integration advertiser", IdempotencyKey: unique("advertiser")})
	if err != nil {
		t.Fatal(err)
	}
	return pool, campaign.NewStore(pool, 5*time.Second), NewStore(pool, 5*time.Second), advertiserResult.Advertiser.ID
}

func draftCampaign(t *testing.T, store *campaign.Store, advertiserID string, amount int64, key string) campaign.Campaign {
	t.Helper()
	result, err := store.Create(context.Background(), campaign.CreateCommand{
		AdvertiserID: advertiserID, Name: "Budget campaign", PlacementCode: "home_feed",
		AmountMinor: amount, Currency: "EUR", Countries: []string{"DE"}, IdempotencyKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.Campaign
}

func activeCampaign(t *testing.T, store *campaign.Store, advertiserID string, amount int64, key string) campaign.Campaign {
	t.Helper()
	value := draftCampaign(t, store, advertiserID, amount, key)
	active, err := store.Activate(context.Background(), value.ID, value.Version)
	if err != nil {
		t.Fatal(err)
	}
	return active
}

func unique(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), integrationSequence.Add(1))
}
