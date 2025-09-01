package handlers

import (
	"log/slog"
	"net/http"
	"time"
)

type StatsHandler struct {
	logger *slog.Logger
}

func NewStatsHandler(logger *slog.Logger) *StatsHandler {
	return &StatsHandler{
		logger: logger,
	}
}

func (h *StatsHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","timestamp":"` + time.Now().Format(time.RFC3339) + `","message":"Stats endpoint coming soon"}`))
}
