package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"user-service/models"
	"user-service/service"

	"github.com/google/uuid"
)

type UserHandler struct {
	svc       service.UserService
	jwtSecret string
}

func NewUserHandler(svc service.UserService, jwtSecret string) *UserHandler {
	return &UserHandler{
		svc:       svc,
		jwtSecret: jwtSecret,
	}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /users/register", h.Register)
	mux.HandleFunc("POST /users/login", h.Login)
	mux.HandleFunc("GET /users/{id}", h.GetUser)
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.svc.Register(r.Context(), &req)
	if err != nil {
		if err == service.ErrUserExists {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		slog.Error("Failed to register user", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.RegisterResponse{User: *user})
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, user, err := h.svc.Login(r.Context(), &req, h.jwtSecret)
	if err != nil {
		if err == service.ErrInvalidCreds {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		slog.Error("Failed to login user", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(models.LoginResponse{
		Token: token,
		User:  *user,
	})
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		// Fallback if context based router wasn't used or id not found
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) > 2 {
			idStr = parts[len(parts)-1]
		}
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := h.svc.GetUser(r.Context(), id)
	if err != nil {
		if err == service.ErrUserNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		slog.Error("Failed to fetch user", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(user)
}
