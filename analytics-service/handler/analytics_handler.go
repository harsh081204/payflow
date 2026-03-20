package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"analytics-service/repository"
)

type AnalyticsHandler struct {
	repo repository.AnalyticsRepository
}

func NewAnalyticsHandler(repo repository.AnalyticsRepository) *AnalyticsHandler {
	return &AnalyticsHandler{repo: repo}
}

func (h *AnalyticsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /metrics/system", h.GetMetrics) // using /metrics/system to avoid prometheus /metrics collision if added later
}

func (h *AnalyticsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.repo.GetMetrics(r.Context())
	if err != nil {
		slog.Error("Failed to get metrics", "error", err)
		http.Error(w, "Failed to retrieve metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}
