package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/IliaTalebzadeh82/mercury/internal/decision"
)

type opportunityRequest struct {
	OpportunityID string `json:"opportunity_id"`
	Placement     string `json:"placement"`
	Country       string `json:"country"`
}

func (request opportunityRequest) opportunity() decision.Opportunity {
	return decision.Opportunity{ID: request.OpportunityID, Placement: request.Placement, Country: request.Country}
}

func (a *API) createDecision(writer http.ResponseWriter, request *http.Request) {
	started := time.Now()
	var body opportunityRequest
	if !decodeJSON(writer, request, &body) {
		return
	}
	result, err := a.decisions.Decide(request.Context(), body.opportunity())
	if err != nil {
		a.handleDecisionError(writer, request, err)
		return
	}
	a.logger.Info("ad decision completed",
		"event", "ad_decision_completed",
		"request_id", writer.Header().Get("X-Request-ID"),
		"opportunity_id", result.OpportunityID,
		"decision_id", result.DecisionID,
		"outcome", result.Outcome,
		"placement", result.Placement,
		"country", result.Country,
		"eligible_candidate_count", result.EligibleCandidateCount,
		"query_duration_ms", result.QueryDuration.Milliseconds(),
		"decision_duration_ms", time.Since(started).Milliseconds(),
	)
	writeJSON(writer, http.StatusOK, result)
}

func (a *API) explainDecision(writer http.ResponseWriter, request *http.Request) {
	var body opportunityRequest
	if !decodeJSON(writer, request, &body) {
		return
	}
	result, err := a.decisions.Explain(request.Context(), body.opportunity())
	if err != nil {
		a.handleDecisionError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (a *API) handleDecisionError(writer http.ResponseWriter, request *http.Request, err error) {
	var validation *decision.ValidationError
	switch {
	case errors.As(err, &validation):
		writeError(writer, request, http.StatusUnprocessableEntity, "validation_failed", "Ad opportunity validation failed", validation.Fields)
	case errors.Is(err, decision.ErrUnsupportedPlacement):
		writeError(writer, request, http.StatusUnprocessableEntity, "unsupported_placement", "Placement is not supported", map[string]string{"placement": "is not a supported placement"})
	case errors.Is(err, decision.ErrDatabase):
		a.logger.Error("ad decision failed", "request_id", writer.Header().Get("X-Request-ID"), "error_kind", "database")
		writeError(writer, request, http.StatusServiceUnavailable, "database_unavailable", "Database operation could not be completed", nil)
	default:
		a.logger.Error("ad decision failed", "request_id", writer.Header().Get("X-Request-ID"), "error_kind", "internal")
		writeError(writer, request, http.StatusInternalServerError, "internal_error", "Ad decision could not be completed", nil)
	}
}
