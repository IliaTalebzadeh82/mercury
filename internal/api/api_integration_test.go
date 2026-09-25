//go:build integration

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IliaTalebzadeh82/mercury/internal/advertiser"
	"github.com/IliaTalebzadeh82/mercury/internal/budget"
	"github.com/IliaTalebzadeh82/mercury/internal/campaign"
	"github.com/IliaTalebzadeh82/mercury/internal/decision"
	"github.com/IliaTalebzadeh82/mercury/internal/platform/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

var apiSequence atomic.Int64

func TestPhaseOneAPIContracts(t *testing.T) {
	application, pool := integrationAPI(t)
	defer pool.Close()

	advertiserKey := apiKey("advertiser")
	response := apiRequest(t, application, http.MethodPost, "/advertisers", map[string]any{"name": " API advertiser "}, map[string]string{"Idempotency-Key": advertiserKey})
	assertStatus(t, response, http.StatusCreated)
	var createdAdvertiser advertiser.Advertiser
	decodeResponse(t, response, &createdAdvertiser)

	replay := apiRequest(t, application, http.MethodPost, "/advertisers", map[string]any{"name": "API advertiser"}, map[string]string{"Idempotency-Key": advertiserKey})
	assertStatus(t, replay, http.StatusOK)
	if replay.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatal("advertiser replay header missing")
	}

	conflictingAdvertiser := apiRequest(t, application, http.MethodPost, "/advertisers", map[string]any{"name": "Different"}, map[string]string{"Idempotency-Key": advertiserKey})
	assertStatus(t, conflictingAdvertiser, http.StatusConflict)

	missingKey := apiRequest(t, application, http.MethodPost, "/advertisers", map[string]any{"name": "Missing key"}, nil)
	assertStatus(t, missingKey, http.StatusBadRequest)

	malformed := httptest.NewRequest(http.MethodPost, "/advertisers", strings.NewReader(`{"name":`))
	malformed.Header.Set("Idempotency-Key", apiKey("malformed"))
	malformedResponse := httptest.NewRecorder()
	application.ServeHTTP(malformedResponse, malformed)
	assertStatus(t, malformedResponse, http.StatusBadRequest)

	campaignKey := apiKey("campaign")
	createBody := map[string]any{
		"name": " Lunch campaign ", "placement_code": "HOME_FEED",
		"budget":    map[string]any{"configured_amount_minor": "001250", "currency": "eur"},
		"targeting": map[string]any{"countries": []string{"fr", "DE", "FR"}},
	}
	created := apiRequest(t, application, http.MethodPost, "/advertisers/"+createdAdvertiser.ID+"/campaigns", createBody, map[string]string{"Idempotency-Key": campaignKey})
	assertStatus(t, created, http.StatusCreated)
	if created.Header().Get("ETag") != `"1"` {
		t.Fatalf("creation ETag = %q", created.Header().Get("ETag"))
	}
	var value campaignResponse
	decodeResponse(t, created, &value)
	if value.Budget.ConfiguredAmountMinor != "1250" || strings.Join(value.Targeting.Countries, ",") != "DE,FR" {
		t.Fatalf("campaign was not normalized: %+v", value)
	}

	replayBody := map[string]any{
		"targeting":      map[string]any{"countries": []string{"FR", "DE"}},
		"budget":         map[string]any{"currency": "EUR", "configured_amount_minor": "+1250"},
		"placement_code": "home_feed", "name": "Lunch campaign",
	}
	replayed := apiRequest(t, application, http.MethodPost, "/advertisers/"+createdAdvertiser.ID+"/campaigns", replayBody, map[string]string{"Idempotency-Key": campaignKey})
	assertStatus(t, replayed, http.StatusOK)
	conflictingCampaign := apiRequest(t, application, http.MethodPost, "/advertisers/"+createdAdvertiser.ID+"/campaigns", map[string]any{
		"name": "Different campaign", "placement_code": "home_feed",
		"budget":    map[string]any{"configured_amount_minor": "1250", "currency": "EUR"},
		"targeting": map[string]any{"countries": []string{"DE", "FR"}},
	}, map[string]string{"Idempotency-Key": campaignKey})
	assertStatus(t, conflictingCampaign, http.StatusConflict)

	invalidCampaign := apiRequest(t, application, http.MethodPost, "/advertisers/"+createdAdvertiser.ID+"/campaigns", map[string]any{
		"name": "Invalid", "placement_code": "home_feed", "budget": map[string]any{"configured_amount_minor": "0", "currency": "EUR"}, "targeting": map[string]any{"countries": []string{}},
	}, map[string]string{"Idempotency-Key": apiKey("invalid")})
	assertStatus(t, invalidCampaign, http.StatusUnprocessableEntity)

	path := "/campaigns/" + value.ID
	missingPrecondition := apiRequest(t, application, http.MethodPut, path+"/name", map[string]any{"name": "Renamed"}, nil)
	assertStatus(t, missingPrecondition, http.StatusPreconditionRequired)
	malformedPrecondition := apiRequest(t, application, http.MethodPut, path+"/name", map[string]any{"name": "Renamed"}, map[string]string{"If-Match": `W/"1"`})
	assertStatus(t, malformedPrecondition, http.StatusBadRequest)
	stale := apiRequest(t, application, http.MethodPut, path+"/name", map[string]any{"name": "Renamed"}, map[string]string{"If-Match": `"9"`})
	assertStatus(t, stale, http.StatusPreconditionFailed)

	renamed := apiRequest(t, application, http.MethodPut, path+"/name", map[string]any{"name": "Renamed"}, map[string]string{"If-Match": `"1"`})
	assertStatus(t, renamed, http.StatusOK)
	if renamed.Header().Get("ETag") != `"2"` {
		t.Fatalf("renamed ETag = %q", renamed.Header().Get("ETag"))
	}
	noOp := apiRequest(t, application, http.MethodPut, path+"/name", map[string]any{"name": "Renamed"}, map[string]string{"If-Match": `"2"`})
	assertStatus(t, noOp, http.StatusOK)
	if noOp.Header().Get("ETag") != `"2"` {
		t.Fatal("semantic no-op incremented version")
	}

	active := apiRequest(t, application, http.MethodPost, path+"/activate", nil, map[string]string{"If-Match": `"2"`})
	assertStatus(t, active, http.StatusOK)
	illegalActivate := apiRequest(t, application, http.MethodPost, path+"/activate", nil, map[string]string{"If-Match": `"3"`})
	assertStatus(t, illegalActivate, http.StatusConflict)
	activePlacement := apiRequest(t, application, http.MethodPut, path+"/placement", map[string]any{"placement_code": "search_results"}, map[string]string{"If-Match": `"3"`})
	assertStatus(t, activePlacement, http.StatusConflict)
	activeDecrease := apiRequest(t, application, http.MethodPut, path+"/budget", map[string]any{"configured_amount_minor": "100"}, map[string]string{"If-Match": `"3"`})
	assertStatus(t, activeDecrease, http.StatusConflict)

	paused := apiRequest(t, application, http.MethodPost, path+"/pause", nil, map[string]string{"If-Match": `"3"`})
	assertStatus(t, paused, http.StatusOK)
	emptyTargeting := apiRequest(t, application, http.MethodPut, path+"/targeting", map[string]any{"countries": []string{}}, map[string]string{"If-Match": `"4"`})
	assertStatus(t, emptyTargeting, http.StatusUnprocessableEntity)
	placed := apiRequest(t, application, http.MethodPut, path+"/placement", map[string]any{"placement_code": "search_results"}, map[string]string{"If-Match": `"4"`})
	assertStatus(t, placed, http.StatusOK)
	targeted := apiRequest(t, application, http.MethodPut, path+"/targeting", map[string]any{"countries": []string{"GB"}}, map[string]string{"If-Match": `"5"`})
	assertStatus(t, targeted, http.StatusOK)
	resumed := apiRequest(t, application, http.MethodPost, path+"/resume", nil, map[string]string{"If-Match": `"6"`})
	assertStatus(t, resumed, http.StatusOK)
	ended := apiRequest(t, application, http.MethodPost, path+"/end", nil, map[string]string{"If-Match": `"7"`})
	assertStatus(t, ended, http.StatusOK)
	terminal := apiRequest(t, application, http.MethodPut, path+"/name", map[string]any{"name": "Forbidden"}, map[string]string{"If-Match": `"8"`})
	assertStatus(t, terminal, http.StatusConflict)
}

func TestAPIReturnsSafeDatabaseFailure(t *testing.T) {
	application, pool := integrationAPI(t)
	pool.Close()
	response := apiRequest(t, application, http.MethodGet, "/placements", nil, nil)
	assertStatus(t, response, http.StatusServiceUnavailable)
	if strings.Contains(response.Body.String(), "postgres://") || strings.Contains(response.Body.String(), "campaigns_") {
		t.Fatalf("unsafe database detail in response: %s", response.Body.String())
	}
	decisionResponse := apiRequest(t, application, http.MethodPost, "/ad-decisions", map[string]any{
		"opportunity_id": "41000000-0000-4000-8000-000000000001", "placement": "home_feed", "country": "US",
	}, nil)
	assertStatus(t, decisionResponse, http.StatusServiceUnavailable)
	if strings.Contains(decisionResponse.Body.String(), "postgres://") || strings.Contains(decisionResponse.Body.String(), "campaign") {
		t.Fatalf("unsafe database detail in decision response: %s", decisionResponse.Body.String())
	}
}

func TestPhaseTwoDecisionAPIContracts(t *testing.T) {
	application, pool := integrationAPI(t)
	defer pool.Close()
	seedDecisionCampaign(t, pool)
	body := map[string]any{
		"opportunity_id": "41000000-0000-4000-8000-000000000101", "placement": " SEARCH_RESULTS ", "country": " rz ",
	}
	first := apiRequest(t, application, http.MethodPost, "/ad-decisions", body, nil)
	assertStatus(t, first, http.StatusOK)
	var firstDecision decision.Decision
	decodeResponse(t, first, &firstDecision)
	if firstDecision.Outcome != "FILL" || firstDecision.Selection == nil || firstDecision.Placement != "search_results" || firstDecision.Country != "RZ" {
		t.Fatalf("unexpected fill: %+v", firstDecision)
	}
	if strings.Contains(first.Body.String(), "eligible_candidate_count") || strings.Contains(first.Body.String(), "ranking") {
		t.Fatalf("normal response leaks diagnostics: %s", first.Body.String())
	}
	second := apiRequest(t, application, http.MethodPost, "/ad-decisions", body, nil)
	assertStatus(t, second, http.StatusOK)
	var secondDecision decision.Decision
	decodeResponse(t, second, &secondDecision)
	if secondDecision.Selection.CampaignID != firstDecision.Selection.CampaignID || secondDecision.DecisionID == firstDecision.DecisionID {
		t.Fatalf("retry identity contract violated: %+v %+v", firstDecision, secondDecision)
	}

	noFill := apiRequest(t, application, http.MethodPost, "/ad-decisions", map[string]any{
		"opportunity_id": "41000000-0000-4000-8000-000000000102", "placement": "search_results", "country": "RY",
	}, nil)
	assertStatus(t, noFill, http.StatusOK)
	var noFillDecision decision.Decision
	decodeResponse(t, noFill, &noFillDecision)
	if noFillDecision.Outcome != "NO_FILL" || noFillDecision.Selection != nil || noFillDecision.DecisionID == "" {
		t.Fatalf("unexpected no-fill: %+v", noFillDecision)
	}

	cases := []struct {
		name string
		body string
		want int
	}{
		{"malformed", `{"opportunity_id":`, http.StatusBadRequest},
		{"multiple", `{"opportunity_id":"41000000-0000-4000-8000-000000000101","placement":"home_feed","country":"US"}{}`, http.StatusBadRequest},
		{"unknown field", `{"opportunity_id":"41000000-0000-4000-8000-000000000101","placement":"home_feed","country":"US","user_id":"forbidden"}`, http.StatusBadRequest},
		{"uppercase UUID", `{"opportunity_id":"41000000-0000-4000-8000-00000000010A","placement":"home_feed","country":"US"}`, http.StatusUnprocessableEntity},
		{"nil UUID", `{"opportunity_id":"00000000-0000-0000-0000-000000000000","placement":"home_feed","country":"US"}`, http.StatusUnprocessableEntity},
		{"unknown placement", `{"opportunity_id":"41000000-0000-4000-8000-000000000103","placement":"not_real","country":"US"}`, http.StatusUnprocessableEntity},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/ad-decisions", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			application.ServeHTTP(response, request)
			assertStatus(t, response, test.want)
		})
	}

	disabled := apiRequest(t, application, http.MethodPost, "/ad-decisions/explain", body, nil)
	assertStatus(t, disabled, http.StatusNotFound)
	enabled := New(slog.New(slog.NewTextHandler(io.Discard, nil)), advertiser.NewStore(pool, 5*time.Second), campaign.NewStore(pool, 5*time.Second), budget.NewStore(pool, 5*time.Second), decision.NewEngine(pool, 5*time.Second), true)
	explained := apiRequest(t, enabled, http.MethodPost, "/ad-decisions/explain", body, nil)
	assertStatus(t, explained, http.StatusOK)
	if !strings.Contains(explained.Body.String(), `"explanations"`) || !strings.Contains(explained.Body.String(), `"eligible":true`) {
		t.Fatalf("diagnostic response missing explanation: %s", explained.Body.String())
	}
}

func TestDecisionAPITimeoutIsServiceUnavailable(t *testing.T) {
	_, pool := integrationAPI(t)
	defer pool.Close()
	application := New(slog.New(slog.NewTextHandler(io.Discard, nil)), advertiser.NewStore(pool, time.Nanosecond), campaign.NewStore(pool, time.Nanosecond), budget.NewStore(pool, time.Nanosecond), decision.NewEngine(pool, time.Nanosecond), false)
	response := apiRequest(t, application, http.MethodPost, "/ad-decisions", map[string]any{
		"opportunity_id": "41000000-0000-4000-8000-000000000104", "placement": "home_feed", "country": "US",
	}, nil)
	assertStatus(t, response, http.StatusServiceUnavailable)
}

func TestSuccessfulDecisionLogIsStructuredAndBounded(t *testing.T) {
	_, pool := integrationAPI(t)
	defer pool.Close()
	seedDecisionCampaign(t, pool)
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	application := New(logger, advertiser.NewStore(pool, 5*time.Second), campaign.NewStore(pool, 5*time.Second), budget.NewStore(pool, 5*time.Second), decision.NewEngine(pool, 5*time.Second), false)
	response := apiRequest(t, application, http.MethodPost, "/ad-decisions", map[string]any{
		"opportunity_id": "41000000-0000-4000-8000-000000000105", "placement": "search_results", "country": "RZ",
	}, nil)
	assertStatus(t, response, http.StatusOK)
	if strings.Contains(logs.String(), "campaign_id") || strings.Contains(logs.String(), "ranking") || strings.Contains(logs.String(), "budget") || strings.Contains(logs.String(), "postgres://") {
		t.Fatalf("decision log contains forbidden data: %s", logs.String())
	}
	var found bool
	for _, line := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("log is not structured JSON: %v: %s", err, line)
		}
		if entry["event"] == "ad_decision_completed" {
			found = true
			for _, key := range []string{"request_id", "opportunity_id", "decision_id", "outcome", "placement", "country", "eligible_candidate_count", "query_duration_ms", "decision_duration_ms"} {
				if _, exists := entry[key]; !exists {
					t.Fatalf("decision log missing %s: %v", key, entry)
				}
			}
		}
	}
	if !found {
		t.Fatalf("structured decision summary not found: %s", logs.String())
	}
}

func TestPhaseThreeBudgetAPIContracts(t *testing.T) {
	application, pool := integrationAPI(t)
	defer pool.Close()
	campaignID := seedBudgetCampaign(t, pool)

	account := apiRequest(t, application, http.MethodGet, "/campaigns/"+campaignID+"/budget", nil, nil)
	assertStatus(t, account, http.StatusOK)
	if !strings.Contains(account.Body.String(), `"configured_amount_minor":"100"`) ||
		!strings.Contains(account.Body.String(), `"committed_spend_minor":"0"`) ||
		!strings.Contains(account.Body.String(), `"remaining_amount_minor":"100"`) {
		t.Fatalf("unexpected budget response: %s", account.Body.String())
	}

	body := map[string]any{"amount_minor": "60", "currency": "eur"}
	approved := apiRequest(t, application, http.MethodPost, "/campaigns/"+campaignID+"/budget-consumptions", body, map[string]string{"Idempotency-Key": "api-budget-approved"})
	assertStatus(t, approved, http.StatusOK)
	if !strings.Contains(approved.Body.String(), `"outcome":"APPROVED"`) || !strings.Contains(approved.Body.String(), `"remaining_amount_minor":"40"`) {
		t.Fatalf("unexpected approval: %s", approved.Body.String())
	}

	replay := apiRequest(t, application, http.MethodPost, "/campaigns/"+campaignID+"/budget-consumptions", body, map[string]string{"Idempotency-Key": "api-budget-approved"})
	assertStatus(t, replay, http.StatusOK)
	if replay.Header().Get("Idempotency-Replayed") != "true" || replay.Body.String() != approved.Body.String() {
		t.Fatalf("replay differs: header=%q body=%s", replay.Header().Get("Idempotency-Replayed"), replay.Body.String())
	}

	conflict := apiRequest(t, application, http.MethodPost, "/campaigns/"+campaignID+"/budget-consumptions", map[string]any{"amount_minor": "61", "currency": "EUR"}, map[string]string{"Idempotency-Key": "api-budget-approved"})
	assertStatus(t, conflict, http.StatusConflict)
	insufficient := apiRequest(t, application, http.MethodPost, "/campaigns/"+campaignID+"/budget-consumptions", map[string]any{"amount_minor": "41", "currency": "EUR"}, map[string]string{"Idempotency-Key": "api-budget-insufficient"})
	assertStatus(t, insufficient, http.StatusOK)
	if !strings.Contains(insufficient.Body.String(), `"outcome":"INSUFFICIENT_BUDGET"`) {
		t.Fatalf("unexpected rejection: %s", insufficient.Body.String())
	}
	currency := apiRequest(t, application, http.MethodPost, "/campaigns/"+campaignID+"/budget-consumptions", map[string]any{"amount_minor": "1", "currency": "USD"}, map[string]string{"Idempotency-Key": "api-budget-currency"})
	assertStatus(t, currency, http.StatusUnprocessableEntity)
	missingKey := apiRequest(t, application, http.MethodPost, "/campaigns/"+campaignID+"/budget-consumptions", body, nil)
	assertStatus(t, missingKey, http.StatusBadRequest)

	var receipts int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM budget_consumption_commands WHERE campaign_id=$1`, campaignID).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if receipts != 2 {
		t.Fatalf("receipts=%d, want approved and insufficient only", receipts)
	}
}

func TestBudgetCompletionLogIsStructuredAndDoesNotLeakFinancialKeys(t *testing.T) {
	_, pool := integrationAPI(t)
	defer pool.Close()
	campaignID := seedBudgetCampaign(t, pool)
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	application := New(logger, advertiser.NewStore(pool, 5*time.Second), campaign.NewStore(pool, 5*time.Second), budget.NewStore(pool, 5*time.Second), decision.NewEngine(pool, 5*time.Second), false)
	response := apiRequest(t, application, http.MethodPost, "/campaigns/"+campaignID+"/budget-consumptions", map[string]any{"amount_minor": "7", "currency": "EUR"}, map[string]string{"Idempotency-Key": "never-log-this-key"})
	assertStatus(t, response, http.StatusOK)
	if strings.Contains(logs.String(), "never-log-this-key") || strings.Contains(logs.String(), `"amount_minor"`) || strings.Contains(logs.String(), "postgres://") {
		t.Fatalf("budget log leaked forbidden data: %s", logs.String())
	}
	if !strings.Contains(logs.String(), `"event":"budget_consumption_completed"`) || !strings.Contains(logs.String(), `"campaign_lock_wait_ms"`) {
		t.Fatalf("completion log missing bounded fields: %s", logs.String())
	}
}

func seedBudgetCampaign(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	sequence := time.Now().UnixNano()
	advertiserKey := fmt.Sprintf("api-budget-advertiser-%d", sequence)
	campaignKey := fmt.Sprintf("api-budget-campaign-%d", sequence)
	var campaignID string
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var advertiserID string
	if err := tx.QueryRow(ctx, `INSERT INTO advertisers (name,creation_idempotency_key,creation_request_fingerprint) VALUES ('Budget API fixture',$1,decode(repeat('31',32),'hex')) RETURNING id::text`, advertiserKey).Scan(&advertiserID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO campaigns (advertiser_id,name,state,placement_code,budget_amount_minor,currency,creation_idempotency_key,creation_request_fingerprint) VALUES ($1,'Budget API campaign','ACTIVE','home_feed',100,'EUR',$2,decode(repeat('32',32),'hex')) RETURNING id::text`, advertiserID, campaignKey).Scan(&campaignID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO campaign_target_countries (campaign_id,country_code) VALUES ($1,'DE')`, campaignID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return campaignID
}

func seedDecisionCampaign(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, `INSERT INTO advertisers (id,name,creation_idempotency_key,creation_request_fingerprint) VALUES ('41000000-0000-4000-8000-000000000001','API decision fixture','api-decision-fixture',decode(repeat('01',32),'hex')) ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO campaigns (id,advertiser_id,name,state,placement_code,budget_amount_minor,currency,creation_idempotency_key,creation_request_fingerprint) VALUES ('41000000-0000-4000-8000-000000000011','41000000-0000-4000-8000-000000000001','API decision fixture','ACTIVE','search_results',1,'USD','api-decision-campaign',decode(repeat('02',32),'hex')) ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO campaign_target_countries (campaign_id,country_code) VALUES ('41000000-0000-4000-8000-000000000011','RZ') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func integrationAPI(t *testing.T) (http.Handler, *pgxpool.Pool) {
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
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(logger, advertiser.NewStore(pool, 5*time.Second), campaign.NewStore(pool, 5*time.Second), budget.NewStore(pool, 5*time.Second), decision.NewEngine(pool, 5*time.Second), false), pool
}

func apiRequest(t *testing.T, handler http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var encoded io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		encoded = bytes.NewReader(data)
	}
	request := httptest.NewRequest(method, path, encoded)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertStatus(t *testing.T, response *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if response.Code != expected {
		t.Fatalf("status=%d want=%d body=%s", response.Code, expected, response.Body.String())
	}
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, value any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), value); err != nil {
		t.Fatalf("decode response: %v body=%s", err, response.Body.String())
	}
}
func apiKey(prefix string) string {
	return prefix + "-" + strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + strconv.FormatInt(apiSequence.Add(1), 10)
}
