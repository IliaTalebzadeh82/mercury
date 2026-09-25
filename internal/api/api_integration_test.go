//go:build integration

package api

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/IliaTalebzadeh82/mercury/internal/campaign"
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
	return New(logger, advertiser.NewStore(pool, 5*time.Second), campaign.NewStore(pool, 5*time.Second)), pool
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
