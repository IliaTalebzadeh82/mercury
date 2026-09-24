package server

import (
	"context"
	"encoding/json"
	"net/http"
)

type statusResponse struct {
	Status string `json:"status"`
}

func (s *Server) health(writer http.ResponseWriter, _ *http.Request) {
	writeStatus(writer, http.StatusOK, "ok")
}

func (s *Server) readiness(writer http.ResponseWriter, request *http.Request) {
	if !s.ready.Load() {
		writeStatus(writer, http.StatusServiceUnavailable, "not_ready")
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), s.pingTimeout)
	defer cancel()
	if err := s.database.Ping(ctx); err != nil {
		writeStatus(writer, http.StatusServiceUnavailable, "database_unavailable")
		return
	}

	writeStatus(writer, http.StatusOK, "ready")
}

func writeStatus(writer http.ResponseWriter, code int, status string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	_ = json.NewEncoder(writer).Encode(statusResponse{Status: status})
}
