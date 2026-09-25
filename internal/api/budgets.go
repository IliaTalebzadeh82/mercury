package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/IliaTalebzadeh82/mercury/internal/budget"
)

type budgetResponse struct {
	CampaignID            string `json:"campaign_id"`
	ConfiguredAmountMinor string `json:"configured_amount_minor"`
	CommittedSpendMinor   string `json:"committed_spend_minor"`
	RemainingAmountMinor  string `json:"remaining_amount_minor"`
	Currency              string `json:"currency"`
}

type consumptionResponse struct {
	ConsumptionID string         `json:"consumption_id"`
	CampaignID    string         `json:"campaign_id"`
	Outcome       budget.Outcome `json:"outcome"`
	Amount        struct {
		AmountMinor string `json:"amount_minor"`
		Currency    string `json:"currency"`
	} `json:"amount"`
	Budget budgetResponse `json:"budget"`
}

func presentBudget(value budget.Account) budgetResponse {
	return budgetResponse{
		CampaignID: value.CampaignID, ConfiguredAmountMinor: strconv.FormatInt(value.ConfiguredAmountMinor, 10),
		CommittedSpendMinor:  strconv.FormatInt(value.CommittedSpendMinor, 10),
		RemainingAmountMinor: strconv.FormatInt(value.RemainingAmountMinor, 10), Currency: value.Currency,
	}
}

func presentConsumption(value budget.Result) consumptionResponse {
	response := consumptionResponse{ConsumptionID: value.ConsumptionID, CampaignID: value.CampaignID, Outcome: value.Outcome}
	response.Amount.AmountMinor = strconv.FormatInt(value.AmountMinor, 10)
	response.Amount.Currency = value.Currency
	response.Budget = budgetResponse{
		CampaignID: value.CampaignID, ConfiguredAmountMinor: strconv.FormatInt(value.ConfiguredMinor, 10),
		CommittedSpendMinor:  strconv.FormatInt(value.ResultingCommitted, 10),
		RemainingAmountMinor: strconv.FormatInt(value.ResultingRemaining, 10), Currency: value.Currency,
	}
	return response
}

func (a *API) getBudget(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathID(writer, request, "campaignID")
	if !ok {
		return
	}
	value, err := a.budgets.GetAccount(request.Context(), id)
	if err != nil {
		a.handleBudgetError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, presentBudget(value))
}

func (a *API) consumeBudget(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathID(writer, request, "campaignID")
	if !ok {
		return
	}
	key, ok := idempotencyKey(writer, request)
	if !ok {
		return
	}
	var body struct {
		AmountMinor string `json:"amount_minor"`
		Currency    string `json:"currency"`
	}
	if !decodeJSON(writer, request, &body) {
		return
	}
	amount, err := parseMinorUnits(body.AmountMinor)
	if err != nil {
		writeError(writer, request, http.StatusUnprocessableEntity, "validation_failed", "Budget consumption validation failed", map[string]string{"amount_minor": "must be an integer minor-unit string"})
		return
	}
	result, err := a.budgets.Consume(request.Context(), budget.Command{
		CampaignID: id, AmountMinor: amount, Currency: body.Currency, IdempotencyKey: key,
	})
	if err != nil {
		a.handleBudgetError(writer, request, err)
		return
	}
	if result.Replayed {
		writer.Header().Set("Idempotency-Replayed", "true")
	}
	a.logger.Info("budget consumption completed",
		"event", "budget_consumption_completed",
		"request_id", writer.Header().Get("X-Request-ID"),
		"consumption_id", result.ConsumptionID,
		"outcome", result.Outcome,
		"replayed", result.Replayed,
		"currency", result.Currency,
		"campaign_lock_wait_ms", result.CampaignLockWait.Milliseconds(),
		"transaction_duration_ms", result.TransactionDuration.Milliseconds(),
	)
	writeJSON(writer, http.StatusOK, presentConsumption(result))
}

func (a *API) handleBudgetError(writer http.ResponseWriter, request *http.Request, err error) {
	var validation *budget.ValidationError
	switch {
	case errors.As(err, &validation):
		writeError(writer, request, http.StatusUnprocessableEntity, "validation_failed", "Budget consumption validation failed", validation.Fields)
	case errors.Is(err, budget.ErrNotFound):
		writeError(writer, request, http.StatusNotFound, "campaign_not_found", "Campaign was not found", nil)
	case errors.Is(err, budget.ErrCurrencyMismatch):
		writeError(writer, request, http.StatusUnprocessableEntity, "currency_mismatch", "Currency must match the campaign currency", map[string]string{"currency": "must match the campaign currency"})
	case errors.Is(err, budget.ErrIdempotencyConflict):
		writeError(writer, request, http.StatusConflict, "idempotency_conflict", "Idempotency-Key was already used for a different budget consumption", nil)
	case errors.Is(err, budget.ErrOutcomeUnknown):
		a.logger.Error("budget consumption outcome unknown", "request_id", writer.Header().Get("X-Request-ID"), "error_kind", "commit_outcome_unknown")
		writeError(writer, request, http.StatusServiceUnavailable, "accounting_outcome_unknown", "Accounting outcome is unknown; retry with the same Idempotency-Key", nil)
	default:
		a.logger.Error("budget operation failed", "request_id", writer.Header().Get("X-Request-ID"), "error_kind", "database")
		writeError(writer, request, http.StatusServiceUnavailable, "accounting_unavailable", "Accounting operation could not be completed; retry consumption with the same Idempotency-Key", nil)
	}
}
