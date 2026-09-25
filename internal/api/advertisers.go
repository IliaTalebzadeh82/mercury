package api

import (
	"net/http"

	"github.com/IliaTalebzadeh82/mercury/internal/advertiser"
)

func (a *API) createAdvertiser(writer http.ResponseWriter, request *http.Request) {
	key, ok := idempotencyKey(writer, request)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if !decodeJSON(writer, request, &body) {
		return
	}
	result, err := a.advertisers.Create(request.Context(), advertiser.CreateCommand{Name: body.Name, IdempotencyKey: key})
	if err != nil {
		a.handleAdvertiserError(writer, request, err)
		return
	}
	writer.Header().Set("Location", "/v1/advertisers/"+result.Advertiser.ID)
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
		writer.Header().Set("Idempotency-Replayed", "true")
	}
	writeJSON(writer, status, result.Advertiser)
}

func (a *API) listAdvertisers(writer http.ResponseWriter, request *http.Request) {
	values, err := a.advertisers.List(request.Context())
	if err != nil {
		a.handleAdvertiserError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"advertisers": values})
}

func (a *API) getAdvertiser(writer http.ResponseWriter, request *http.Request) {
	id, ok := pathID(writer, request, "advertiserID")
	if !ok {
		return
	}
	value, err := a.advertisers.Get(request.Context(), id)
	if err != nil {
		a.handleAdvertiserError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, value)
}
