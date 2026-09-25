//go:build integration

package decision

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/IliaTalebzadeh82/mercury/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDecisionFillNoFillAndUnsupportedPlacement(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	seedCampaigns(t, pool, "31000000-0000-4000-8000-000000000001", []seedCampaign{
		{ID: "31000000-0000-4000-8000-000000000011", State: "ACTIVE", Placement: "search_results", Countries: []string{"QZ"}, Budget: 1},
		{ID: "31000000-0000-4000-8000-000000000012", State: "ACTIVE", Placement: "search_results", Countries: []string{"QZ"}, Budget: 9_999_999},
	})
	engine := NewEngine(pool, 5*time.Second)
	opportunity := Opportunity{ID: "31000000-0000-4000-8000-000000000101", Placement: " SEARCH_RESULTS ", Country: " qz "}
	first, err := engine.Decide(context.Background(), opportunity)
	if err != nil {
		t.Fatalf("Decide(): %v", err)
	}
	second, err := engine.Decide(context.Background(), opportunity)
	if err != nil {
		t.Fatalf("Decide() repeated: %v", err)
	}
	if first.Outcome != "FILL" || first.Selection == nil || first.EligibleCandidateCount != 2 {
		t.Fatalf("unexpected fill: %+v", first)
	}
	if first.Selection.CampaignID != second.Selection.CampaignID || first.DecisionID == second.DecisionID {
		t.Fatalf("winner must be stable and decision IDs distinct: %+v %+v", first, second)
	}
	if first.Placement != "search_results" || first.Country != "QZ" {
		t.Fatalf("normalization not reflected: %+v", first)
	}

	noFill, err := engine.Decide(context.Background(), Opportunity{ID: "31000000-0000-4000-8000-000000000102", Placement: "search_results", Country: "QY"})
	if err != nil {
		t.Fatalf("no-fill Decide(): %v", err)
	}
	if noFill.Outcome != "NO_FILL" || noFill.Selection != nil || noFill.EligibleCandidateCount != 0 || noFill.DecisionID == "" {
		t.Fatalf("unexpected no-fill: %+v", noFill)
	}
	if _, err := engine.Decide(context.Background(), Opportunity{ID: "31000000-0000-4000-8000-000000000103", Placement: "missing_placement", Country: "QZ"}); err != ErrUnsupportedPlacement {
		t.Fatalf("unsupported placement error=%v", err)
	}
}

func TestPostgreSQLAndGoRankingOrderAgree(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	ids := []string{
		"32000000-0000-4000-8000-000000000011",
		"32000000-0000-4000-8000-000000000012",
		"32000000-0000-4000-8000-000000000013",
		"32000000-0000-4000-8000-000000000014",
	}
	fixtures := make([]seedCampaign, len(ids))
	for index, id := range ids {
		fixtures[index] = seedCampaign{ID: id, State: "ACTIVE", Placement: "home_feed", Countries: []string{"QX"}, Budget: int64(index + 1)}
	}
	seedCampaigns(t, pool, "32000000-0000-4000-8000-000000000001", fixtures)
	engine := NewEngine(pool, 5*time.Second)
	for index := 101; index <= 125; index++ {
		opportunityID := fmt.Sprintf("32000000-0000-4000-8000-%012x", index)
		want, _ := choose(opportunityID, candidates(ids))
		result, err := engine.Decide(context.Background(), Opportunity{ID: opportunityID, Placement: "home_feed", Country: "QX"})
		if err != nil {
			t.Fatal(err)
		}
		if result.Selection == nil || result.Selection.CampaignID != want.ID {
			t.Fatalf("opportunity %s PostgreSQL winner=%+v Go winner=%s", opportunityID, result.Selection, want.ID)
		}
		for _, id := range ids {
			var digest []byte
			if err := pool.QueryRow(context.Background(), `SELECT decode(md5($1::uuid::text || ':' || $2::uuid::text),'hex')`, opportunityID, id).Scan(&digest); err != nil {
				t.Fatal(err)
			}
			goDigest := score(opportunityID, id)
			if string(digest) != string(goDigest[:]) {
				t.Fatalf("digest mismatch for opportunity %s campaign %s", opportunityID, id)
			}
		}
	}
}

func TestSQLAndDiagnosticEligibilityAgree(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	seedCampaigns(t, pool, "33000000-0000-4000-8000-000000000001", []seedCampaign{
		{ID: "33000000-0000-4000-8000-000000000011", State: "ACTIVE", Placement: "restaurant_list", Countries: []string{"QW"}, Budget: 1},
		{ID: "33000000-0000-4000-8000-000000000012", State: "PAUSED", Placement: "restaurant_list", Countries: []string{"QW"}, Budget: 1},
		{ID: "33000000-0000-4000-8000-000000000013", State: "ACTIVE", Placement: "home_feed", Countries: []string{"QW"}, Budget: 1},
		{ID: "33000000-0000-4000-8000-000000000014", State: "ACTIVE", Placement: "restaurant_list", Countries: []string{"QV"}, Budget: 1},
	})
	result, err := NewEngine(pool, 5*time.Second).Explain(context.Background(), Opportunity{
		ID: "33000000-0000-4000-8000-000000000101", Placement: "restaurant_list", Country: "QW",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision.EligibleCandidateCount != 1 || result.Decision.Selection == nil || result.Decision.Selection.CampaignID != "33000000-0000-4000-8000-000000000011" {
		t.Fatalf("unexpected SQL eligibility: %+v", result.Decision)
	}
	byID := make(map[string]Explanation)
	for _, explanation := range result.Explanations {
		byID[explanation.CampaignID] = explanation
	}
	checks := map[string][]string{
		"33000000-0000-4000-8000-000000000011": {},
		"33000000-0000-4000-8000-000000000012": {"campaign_not_active"},
		"33000000-0000-4000-8000-000000000013": {"placement_mismatch"},
		"33000000-0000-4000-8000-000000000014": {"country_not_targeted"},
	}
	for id, want := range checks {
		got, exists := byID[id]
		if !exists || fmt.Sprint(got.Reasons) != fmt.Sprint(want) || got.Eligible != (len(want) == 0) {
			t.Fatalf("explanation %s=%+v want reasons=%v", id, got, want)
		}
	}
}

func TestDiagnosticIsBoundedAndIncludesSelectedCampaign(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	fixtures := make([]seedCampaign, 201)
	for index := range fixtures {
		fixtures[index] = seedCampaign{
			ID: fmt.Sprintf("34000000-0000-4000-8000-%012x", index+1), State: "DRAFT",
			Placement: "home_feed", Countries: []string{"QU"}, Budget: 1,
		}
	}
	selectedID := fixtures[200].ID
	fixtures[200].State = "ACTIVE"
	seedCampaigns(t, pool, "34000000-0000-4000-8000-000000000fff", fixtures)
	result, err := NewEngine(pool, 5*time.Second).Explain(context.Background(), Opportunity{
		ID: "34000000-0000-4000-8000-000000001001", Placement: "home_feed", Country: "QU",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Truncated || len(result.Explanations) != explanationLimit || !containsExplanation(result.Explanations, selectedID) {
		t.Fatalf("diagnostic bounds violated: truncated=%t count=%d selected=%t", result.Truncated, len(result.Explanations), containsExplanation(result.Explanations, selectedID))
	}
}

func TestProductionDecisionRanksAllOneThousandEligibleCandidates(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	fixtures := make([]seedCampaign, 1000)
	ids := make([]string, 1000)
	for index := range fixtures {
		id := fmt.Sprintf("39000000-0000-4000-8000-%012x", index+1)
		ids[index] = id
		fixtures[index] = seedCampaign{ID: id, State: "ACTIVE", Placement: "search_results", Countries: []string{"QN"}, Budget: int64(index + 1)}
	}
	seedCampaigns(t, pool, "39000000-0000-4000-8000-000000000fff", fixtures)
	opportunityID := "39000000-0000-4000-8000-000000001001"
	want, _ := choose(opportunityID, candidates(ids))
	result, err := NewEngine(pool, 5*time.Second).Decide(context.Background(), Opportunity{ID: opportunityID, Placement: "search_results", Country: "QN"})
	if err != nil {
		t.Fatal(err)
	}
	if result.EligibleCandidateCount != 1000 || result.Selection == nil || result.Selection.CampaignID != want.ID {
		t.Fatalf("production result=%+v Go winner=%s", result, want.ID)
	}
}

func TestSimultaneousDecisionsHaveStableWinnerAndUniqueIdentities(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	seedCampaigns(t, pool, "35000000-0000-4000-8000-000000000001", []seedCampaign{
		{ID: "35000000-0000-4000-8000-000000000011", State: "ACTIVE", Placement: "search_results", Countries: []string{"QT"}, Budget: 1},
		{ID: "35000000-0000-4000-8000-000000000012", State: "ACTIVE", Placement: "search_results", Countries: []string{"QT"}, Budget: 1},
	})
	engine := NewEngine(pool, 5*time.Second)
	opportunity := Opportunity{ID: "35000000-0000-4000-8000-000000000101", Placement: "search_results", Country: "QT"}
	const requests = 32
	start := make(chan struct{})
	results := make(chan Decision, requests)
	errors := make(chan error, requests)
	var wait sync.WaitGroup
	for range requests {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			value, err := engine.Decide(context.Background(), opportunity)
			if err != nil {
				errors <- err
				return
			}
			results <- value
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	close(errors)
	for err := range errors {
		t.Errorf("Decide(): %v", err)
	}
	ids := make(map[string]struct{}, requests)
	winner := ""
	for value := range results {
		if value.Selection == nil {
			t.Fatalf("simultaneous decision returned no selection: %+v", value)
		}
		if winner == "" {
			winner = value.Selection.CampaignID
		}
		if value.Selection.CampaignID != winner {
			t.Fatalf("unstable winner: %+v want=%s", value.Selection, winner)
		}
		ids[value.DecisionID] = struct{}{}
	}
	if len(ids) != requests {
		t.Fatalf("unique decision IDs=%d want=%d", len(ids), requests)
	}
}

func TestDiagnosticSelectionAndReasonsUseOneRepeatableReadSnapshot(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	id := "37000000-0000-4000-8000-000000000011"
	seedCampaigns(t, pool, "37000000-0000-4000-8000-000000000001", []seedCampaign{
		{ID: id, State: "ACTIVE", Placement: "restaurant_list", Countries: []string{"QP"}, Budget: 1},
	})
	opportunity := Opportunity{ID: "37000000-0000-4000-8000-000000000101", Placement: "restaurant_list", Country: "QP"}
	tx, err := pool.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	engine := NewEngine(pool, 5*time.Second)
	selected, err := engine.queryDecision(context.Background(), tx, opportunity)
	if err != nil || selected.Selection == nil || selected.Selection.CampaignID != id {
		t.Fatalf("snapshot selection=%+v err=%v", selected, err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE campaigns SET state='PAUSED',version=version+1 WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	candidate, err := loadDiagnosticCandidate(context.Background(), tx, id, "QP")
	if err != nil {
		t.Fatal(err)
	}
	if reasons := rejectionReasons(candidate, opportunity); len(reasons) != 0 {
		t.Fatalf("diagnostic mixed snapshots: state=%s reasons=%v", candidate.State, reasons)
	}
}

func TestDecisionObservationsAcrossPauseTargetReplacementAndResume(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	id := "36000000-0000-4000-8000-000000000011"
	seedCampaigns(t, pool, "36000000-0000-4000-8000-000000000001", []seedCampaign{
		{ID: id, State: "ACTIVE", Placement: "home_feed", Countries: []string{"QS"}, Budget: 1},
	})
	engine := NewEngine(pool, 5*time.Second)
	oldOpportunity := Opportunity{ID: "36000000-0000-4000-8000-000000000101", Placement: "home_feed", Country: "QS"}
	newOpportunity := Opportunity{ID: "36000000-0000-4000-8000-000000000102", Placement: "home_feed", Country: "QR"}
	assertOutcome(t, engine, oldOpportunity, "FILL")

	if _, err := pool.Exec(context.Background(), `UPDATE campaigns SET state='PAUSED',version=version+1 WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	assertOutcome(t, engine, oldOpportunity, "NO_FILL")

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(context.Background(), `DELETE FROM campaign_target_countries WHERE campaign_id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(context.Background(), `INSERT INTO campaign_target_countries (campaign_id,country_code) VALUES ($1,'QR')`, id); err != nil {
		t.Fatal(err)
	}
	// The uncommitted replacement is invisible and the committed campaign remains paused.
	assertOutcome(t, engine, oldOpportunity, "NO_FILL")
	assertOutcome(t, engine, newOpportunity, "NO_FILL")
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE campaigns SET state='ACTIVE',version=version+1 WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	assertOutcome(t, engine, oldOpportunity, "NO_FILL")
	assertOutcome(t, engine, newOpportunity, "FILL")
}

func TestDecisionConcurrentWithPauseSatisfiesStatementSnapshotContract(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	id := "38000000-0000-4000-8000-000000000011"
	seedCampaigns(t, pool, "38000000-0000-4000-8000-000000000001", []seedCampaign{
		{ID: id, State: "ACTIVE", Placement: "home_feed", Countries: []string{"QO"}, Budget: 1},
	})
	engine := NewEngine(pool, 5*time.Second)
	opportunity := Opportunity{ID: "38000000-0000-4000-8000-000000000101", Placement: "home_feed", Country: "QO"}
	for iteration := range 25 {
		if _, err := pool.Exec(context.Background(), `UPDATE campaigns SET state='ACTIVE',version=version+1 WHERE id=$1`, id); err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		resultChannel := make(chan Decision, 1)
		errorChannel := make(chan error, 2)
		var wait sync.WaitGroup
		wait.Add(2)
		go func() {
			defer wait.Done()
			<-start
			result, err := engine.Decide(context.Background(), opportunity)
			if err != nil {
				errorChannel <- err
				return
			}
			resultChannel <- result
		}()
		go func() {
			defer wait.Done()
			<-start
			if _, err := pool.Exec(context.Background(), `UPDATE campaigns SET state='PAUSED',version=version+1 WHERE id=$1`, id); err != nil {
				errorChannel <- err
			}
		}()
		close(start)
		wait.Wait()
		close(errorChannel)
		for err := range errorChannel {
			t.Fatalf("iteration %d: %v", iteration, err)
		}
		result := <-resultChannel
		if result.Outcome != "FILL" && result.Outcome != "NO_FILL" {
			t.Fatalf("iteration %d observed invalid outcome %s", iteration, result.Outcome)
		}
		// Once pause has committed, every later statement must exclude the campaign.
		assertOutcome(t, engine, opportunity, "NO_FILL")
	}
}

func integrationPool(t *testing.T) *pgxpool.Pool {
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
	return pool
}

type seedCampaign struct {
	ID        string
	State     string
	Placement string
	Countries []string
	Budget    int64
}

func seedCampaigns(t *testing.T, pool *pgxpool.Pool, advertiserID string, values []seedCampaign) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, `DELETE FROM campaigns WHERE advertiser_id=$1`, advertiserID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM advertisers WHERE id=$1`, advertiserID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO advertisers (id,name,creation_idempotency_key,creation_request_fingerprint) VALUES ($1::uuid,'Decision fixture',$1::text,decode(repeat('01',32),'hex')) ON CONFLICT (id) DO NOTHING`, advertiserID); err != nil {
		t.Fatal(err)
	}
	for _, value := range values {
		if _, err := tx.Exec(ctx, `
			INSERT INTO campaigns (id,advertiser_id,name,state,placement_code,budget_amount_minor,currency,creation_idempotency_key,creation_request_fingerprint)
			VALUES ($1::uuid,$2::uuid,'Decision fixture',$3,$4,$5,'USD',$1::text,decode(repeat('02',32),'hex')) ON CONFLICT (id) DO NOTHING`,
			value.ID, advertiserID, value.State, value.Placement, value.Budget); err != nil {
			t.Fatal(err)
		}
		for _, country := range value.Countries {
			if _, err := tx.Exec(ctx, `INSERT INTO campaign_target_countries (campaign_id,country_code) VALUES ($1,$2) ON CONFLICT DO NOTHING`, value.ID, country); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func containsExplanation(values []Explanation, id string) bool {
	for _, value := range values {
		if value.CampaignID == id {
			return true
		}
	}
	return false
}

func assertOutcome(t *testing.T, engine *Engine, opportunity Opportunity, want string) {
	t.Helper()
	result, err := engine.Decide(context.Background(), opportunity)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != want {
		t.Fatalf("outcome=%s want=%s", result.Outcome, want)
	}
}
