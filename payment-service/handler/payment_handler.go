package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"payment-service/models"
	"payment-service/service"

	"github.com/google/uuid"
)

type PaymentHandler struct {
	svc service.PaymentService
}

func NewPaymentHandler(svc service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

func (h *PaymentHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /payments/charge", h.Charge)
}

func (h *PaymentHandler) Charge(w http.ResponseWriter, r *http.Request) {
	// Parse Idempotency-Key
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		http.Error(w, "Idempotency-Key header is required", http.StatusBadRequest)
		return
	}

	// Read X-User-Id added by Gateway middleware
	userIDStr := r.Header.Get("X-User-Id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid or missing X-User-Id header", http.StatusUnauthorized)
		return
	}

	var req models.ChargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.svc.Charge(r.Context(), userID, &req, idempotencyKey)
	if err != nil {
		if err == service.ErrDuplicateRequest {
			// Idempotency constraint triggered
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error(), "transaction": resp})
			return
		}
		if err == service.ErrInsufficientFund {
			http.Error(w, err.Error(), http.StatusPaymentRequired)
			return
		}
		if err == service.ErrAccountNotFound || err.Error() == "merchant account missing" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		slog.Error("Failed to process payment", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
