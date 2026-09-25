package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/IliaTalebzadeh82/mercury/internal/campaign"
)

type campaignCreationRequest struct {
	Name          string `json:"name"`
	PlacementCode string `json:"placement_code"`
	Budget        struct {
		ConfiguredAmountMinor string `json:"configured_amount_minor"`
		Currency              string `json:"currency"`
	} `json:"budget"`
	Targeting struct {
		Countries []string `json:"countries"`
	} `json:"targeting"`
}

func (a *API) createCampaign(writer http.ResponseWriter, request *http.Request) {
	advertiserID, ok := pathID(writer, request, "advertiserID")
	if !ok {
		return
	}
	key, ok := idempotencyKey(writer, request)
	if !ok {
		return
	}
	var body campaignCreationRequest
	if !decodeJSON(writer, request, &body) {
		return
	}
	amount, err := parseMinorUnits(body.Budget.ConfiguredAmountMinor)
	if err != nil {
		writeError(writer, request, http.StatusUnprocessableEntity, "validation_failed", "Campaign validation failed", map[string]string{"configured_amount_minor": "must be an integer minor-unit string"})
		return
	}
	result, err := a.campaigns.Create(request.Context(), campaign.CreateCommand{
		AdvertiserID: advertiserID, Name: body.Name, PlacementCode: body.PlacementCode,
		AmountMinor: amount, Currency: body.Budget.Currency, Countries: body.Targeting.Countries, IdempotencyKey: key,
	})
	if err != nil {
		a.handleCampaignError(writer, request, err)
		return
	}
	setCampaignETag(writer, result.Campaign)
	writer.Header().Set("Location", "/v1/campaigns/"+result.Campaign.ID)
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
		writer.Header().Set("Idempotency-Replayed", "true")
	}
	writeJSON(writer, status, presentCampaign(result.Campaign))
}

func (a *API) listCampaigns(writer http.ResponseWriter, request *http.Request) {
	advertiserID, ok := pathID(writer, request, "advertiserID")
	if !ok {
		return
	}
	if _, err := a.advertisers.Get(request.Context(), advertiserID); err != nil {
		a.handleAdvertiserError(writer, request, err)
		return
	}
	limit, offset := 50, 0
	if raw := request.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			writeError(writer, request, 400, "invalid_pagination", "limit must be between 1 and 100", nil)
			return
		}
		limit = value
	}
	if raw := request.URL.Query().Get("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			writeError(writer, request, 400, "invalid_pagination", "offset must be non-negative", nil)
			return
		}
		offset = value
	}
	values, err := a.campaigns.List(request.Context(), advertiserID, limit, offset)
	if err != nil {
		a.handleCampaignError(writer, request, err)
		return
	}
	responses := make([]campaignResponse, len(values))
	for index := range values {
		responses[index] = presentCampaign(values[index])
	}
	writeJSON(writer, http.StatusOK, map[string]any{"campaigns": responses, "limit": limit, "offset": offset})
}

func (a *API) getCampaign(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathID(writer, request, "campaignID")
	if !ok {
		return
	}
	value, err := a.campaigns.Get(request.Context(), id)
	if err != nil {
		a.handleCampaignError(writer, request, err)
		return
	}
	setCampaignETag(writer, value)
	writeJSON(writer, http.StatusOK, presentCampaign(value))
}

func (a *API) updateName(writer http.ResponseWriter, request *http.Request) {
	a.mutateCampaign(writer, request, func(id string, version int64) (campaign.Campaign, error) {
		var body struct {
			Name string `json:"name"`
		}
		if !decodeJSON(writer, request, &body) {
			return campaign.Campaign{}, errAlreadyWritten
		}
		return a.campaigns.UpdateName(request.Context(), id, version, body.Name)
	})
}

func (a *API) updateBudget(writer http.ResponseWriter, request *http.Request) {
	a.mutateCampaign(writer, request, func(id string, version int64) (campaign.Campaign, error) {
		var body struct {
			ConfiguredAmountMinor string `json:"configured_amount_minor"`
		}
		if !decodeJSON(writer, request, &body) {
			return campaign.Campaign{}, errAlreadyWritten
		}
		amount, err := parseMinorUnits(body.ConfiguredAmountMinor)
		if err != nil {
			return campaign.Campaign{}, &campaign.ValidationError{Fields: map[string]string{"configured_amount_minor": "must be an integer minor-unit string"}}
		}
		return a.campaigns.UpdateBudget(request.Context(), id, version, amount)
	})
}

func (a *API) updatePlacement(writer http.ResponseWriter, request *http.Request) {
	a.mutateCampaign(writer, request, func(id string, version int64) (campaign.Campaign, error) {
		var body struct {
			PlacementCode string `json:"placement_code"`
		}
		if !decodeJSON(writer, request, &body) {
			return campaign.Campaign{}, errAlreadyWritten
		}
		return a.campaigns.UpdatePlacement(request.Context(), id, version, body.PlacementCode)
	})
}

func (a *API) updateTargeting(writer http.ResponseWriter, request *http.Request) {
	a.mutateCampaign(writer, request, func(id string, version int64) (campaign.Campaign, error) {
		var body struct {
			Countries []string `json:"countries"`
		}
		if !decodeJSON(writer, request, &body) {
			return campaign.Campaign{}, errAlreadyWritten
		}
		return a.campaigns.UpdateTargeting(request.Context(), id, version, body.Countries)
	})
}

var errAlreadyWritten = &campaign.ConflictError{Code: "response_already_written"}

func (a *API) mutateCampaign(writer http.ResponseWriter, request *http.Request, command func(string, int64) (campaign.Campaign, error)) {
	id, ok := pathID(writer, request, "campaignID")
	if !ok {
		return
	}
	version, ok := expectedVersion(writer, request)
	if !ok {
		return
	}
	value, err := command(id, version)
	if err == errAlreadyWritten {
		return
	}
	if err != nil {
		a.handleCampaignError(writer, request, err)
		return
	}
	setCampaignETag(writer, value)
	writeJSON(writer, http.StatusOK, presentCampaign(value))
}

func (a *API) activate(writer http.ResponseWriter, request *http.Request) {
	a.lifecycle(writer, request, a.campaigns.Activate)
}
func (a *API) pause(writer http.ResponseWriter, request *http.Request) {
	a.lifecycle(writer, request, a.campaigns.Pause)
}
func (a *API) resume(writer http.ResponseWriter, request *http.Request) {
	a.lifecycle(writer, request, a.campaigns.Resume)
}
func (a *API) end(writer http.ResponseWriter, request *http.Request) {
	a.lifecycle(writer, request, a.campaigns.End)
}

func (a *API) lifecycle(writer http.ResponseWriter, request *http.Request, command func(context.Context, string, int64) (campaign.Campaign, error)) {
	id, ok := pathID(writer, request, "campaignID")
	if !ok {
		return
	}
	version, ok := expectedVersion(writer, request)
	if !ok {
		return
	}
	value, err := command(request.Context(), id, version)
	if err != nil {
		a.handleCampaignError(writer, request, err)
		return
	}
	setCampaignETag(writer, value)
	writeJSON(writer, http.StatusOK, presentCampaign(value))
}

func (a *API) listPlacements(writer http.ResponseWriter, request *http.Request) {
	values, err := a.campaigns.ListPlacements(request.Context())
	if err != nil {
		a.handleCampaignError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"placements": values})
}
