package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/IliaTalebzadeh82/mercury/internal/advertiser"
	"github.com/IliaTalebzadeh82/mercury/internal/campaign"
	"github.com/go-chi/chi/v5"
)

const maxRequestBody = 1 << 20

var (
	etagPattern = regexp.MustCompile(`^"([1-9][0-9]*)"$`)
	uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

type API struct {
	logger      *slog.Logger
	advertisers *advertiser.Store
	campaigns   *campaign.Store
}

func New(logger *slog.Logger, advertisers *advertiser.Store, campaigns *campaign.Store) http.Handler {
	api := &API{logger: logger, advertisers: advertisers, campaigns: campaigns}
	router := chi.NewRouter()
	router.Use(api.requestLog)
	router.NotFound(func(writer http.ResponseWriter, request *http.Request) {
		writeError(writer, request, http.StatusNotFound, "route_not_found", "API route was not found", nil)
	})
	router.MethodNotAllowed(func(writer http.ResponseWriter, request *http.Request) {
		writeError(writer, request, http.StatusMethodNotAllowed, "method_not_allowed", "HTTP method is not allowed for this route", nil)
	})
	router.Post("/advertisers", api.createAdvertiser)
	router.Get("/advertisers", api.listAdvertisers)
	router.Get("/advertisers/{advertiserID}", api.getAdvertiser)
	router.Get("/placements", api.listPlacements)
	router.Post("/advertisers/{advertiserID}/campaigns", api.createCampaign)
	router.Get("/advertisers/{advertiserID}/campaigns", api.listCampaigns)
	router.Get("/campaigns/{campaignID}", api.getCampaign)
	router.Put("/campaigns/{campaignID}/name", api.updateName)
	router.Put("/campaigns/{campaignID}/budget", api.updateBudget)
	router.Put("/campaigns/{campaignID}/placement", api.updatePlacement)
	router.Put("/campaigns/{campaignID}/targeting", api.updateTargeting)
	router.Post("/campaigns/{campaignID}/activate", api.activate)
	router.Post("/campaigns/{campaignID}/pause", api.pause)
	router.Post("/campaigns/{campaignID}/resume", api.resume)
	router.Post("/campaigns/{campaignID}/end", api.end)
	return router
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (writer *statusRecorder) WriteHeader(status int) {
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func (a *API) requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		requestID := newRequestID()
		writer.Header().Set("X-Request-ID", requestID)
		recorder := &statusRecorder{ResponseWriter: writer, status: http.StatusOK}
		next.ServeHTTP(recorder, request)
		a.logger.Info("api request completed",
			"request_id", requestID,
			"method", request.Method,
			"route", chi.RouteContext(request.Context()).RoutePattern(),
			"status", recorder.status,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}

func newRequestID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "request-id-unavailable"
	}
	return hex.EncodeToString(value)
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

func writeError(writer http.ResponseWriter, request *http.Request, status int, code, message string, fields map[string]string) {
	writeJSON(writer, status, errorEnvelope{Error: errorBody{
		Code: code, Message: message, Fields: fields, RequestID: writer.Header().Get("X-Request-ID"),
	}})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, destination any) bool {
	request.Body = http.MaxBytesReader(writer, request.Body, maxRequestBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeError(writer, request, http.StatusBadRequest, "malformed_request", "Request body must be valid JSON", nil)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(writer, request, http.StatusBadRequest, "malformed_request", "Request body must contain one JSON object", nil)
		return false
	}
	return true
}

func idempotencyKey(writer http.ResponseWriter, request *http.Request) (string, bool) {
	values := request.Header.Values("Idempotency-Key")
	if len(values) != 1 || strings.Contains(values[0], ",") || len(values[0]) < 1 || len(values[0]) > 128 {
		writeError(writer, request, http.StatusBadRequest, "invalid_idempotency_key", "Exactly one Idempotency-Key of 1 to 128 bytes is required", nil)
		return "", false
	}
	return values[0], true
}

func expectedVersion(writer http.ResponseWriter, request *http.Request) (int64, bool) {
	values := request.Header.Values("If-Match")
	if len(values) == 0 {
		writeError(writer, request, http.StatusPreconditionRequired, "precondition_required", "If-Match is required", nil)
		return 0, false
	}
	if len(values) != 1 || strings.Contains(values[0], ",") {
		writeError(writer, request, http.StatusBadRequest, "malformed_precondition", "If-Match must contain one strong quoted version", nil)
		return 0, false
	}
	match := etagPattern.FindStringSubmatch(values[0])
	if match == nil {
		writeError(writer, request, http.StatusBadRequest, "malformed_precondition", "If-Match must contain one strong quoted version", nil)
		return 0, false
	}
	value, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		writeError(writer, request, http.StatusBadRequest, "malformed_precondition", "If-Match version is outside the supported range", nil)
		return 0, false
	}
	return value, true
}

func pathID(writer http.ResponseWriter, request *http.Request, name string) (string, bool) {
	raw := chi.URLParam(request, name)
	if !uuidPattern.MatchString(raw) {
		writeError(writer, request, http.StatusBadRequest, "invalid_identifier", "Resource identifier must be a UUID", nil)
		return "", false
	}
	return strings.ToLower(raw), true
}

func parseMinorUnits(raw string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
}

func setCampaignETag(writer http.ResponseWriter, value campaign.Campaign) {
	writer.Header().Set("ETag", fmt.Sprintf(`"%d"`, value.Version))
}

type campaignResponse struct {
	ID            string         `json:"id"`
	AdvertiserID  string         `json:"advertiser_id"`
	Name          string         `json:"name"`
	State         campaign.State `json:"state"`
	PlacementCode string         `json:"placement_code"`
	Budget        struct {
		ConfiguredAmountMinor string `json:"configured_amount_minor"`
		Currency              string `json:"currency"`
	} `json:"budget"`
	Targeting struct {
		Countries []string `json:"countries"`
	} `json:"targeting"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func presentCampaign(value campaign.Campaign) campaignResponse {
	response := campaignResponse{
		ID: value.ID, AdvertiserID: value.AdvertiserID, Name: value.Name, State: value.State,
		PlacementCode: value.PlacementCode, Version: value.Version, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
	response.Budget.ConfiguredAmountMinor = strconv.FormatInt(value.BudgetAmountMinor, 10)
	response.Budget.Currency = value.Currency
	response.Targeting.Countries = value.Countries
	return response
}

func (a *API) handleAdvertiserError(writer http.ResponseWriter, request *http.Request, err error) {
	var validation *advertiser.ValidationError
	switch {
	case errors.As(err, &validation):
		writeError(writer, request, http.StatusUnprocessableEntity, "validation_failed", "Advertiser validation failed", validation.Fields)
	case errors.Is(err, advertiser.ErrNotFound):
		writeError(writer, request, http.StatusNotFound, "advertiser_not_found", "Advertiser was not found", nil)
	case errors.Is(err, advertiser.ErrIdempotencyConflict):
		writeError(writer, request, http.StatusConflict, "idempotency_conflict", "Idempotency-Key was already used for a different advertiser", nil)
	default:
		a.logger.Error("advertiser operation failed", "request_id", writer.Header().Get("X-Request-ID"), "error_kind", "database")
		writeError(writer, request, http.StatusServiceUnavailable, "database_unavailable", "Database operation could not be completed", nil)
	}
}

func (a *API) handleCampaignError(writer http.ResponseWriter, request *http.Request, err error) {
	var validation *campaign.ValidationError
	var conflict *campaign.ConflictError
	switch {
	case errors.As(err, &validation):
		writeError(writer, request, http.StatusUnprocessableEntity, "validation_failed", "Campaign validation failed", validation.Fields)
	case errors.Is(err, campaign.ErrNotFound):
		writeError(writer, request, http.StatusNotFound, "campaign_not_found", "Campaign was not found", nil)
	case errors.Is(err, campaign.ErrAdvertiserNotFound):
		writeError(writer, request, http.StatusNotFound, "advertiser_not_found", "Advertiser was not found", nil)
	case errors.Is(err, campaign.ErrPreconditionFailed):
		writeError(writer, request, http.StatusPreconditionFailed, "precondition_failed", "Campaign changed since it was loaded", nil)
	case errors.Is(err, campaign.ErrIdempotencyConflict):
		writeError(writer, request, http.StatusConflict, "idempotency_conflict", "Idempotency-Key was already used for a different campaign", nil)
	case errors.As(err, &conflict):
		writeError(writer, request, http.StatusConflict, conflict.Code, "Campaign command conflicts with its current state", nil)
	default:
		a.logger.Error("campaign operation failed", "request_id", writer.Header().Get("X-Request-ID"), "error_kind", "database")
		writeError(writer, request, http.StatusServiceUnavailable, "database_unavailable", "Database operation could not be completed", nil)
	}
}
