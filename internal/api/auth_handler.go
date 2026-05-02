package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"simple-orderbook/internal/core/ports"
	"simple-orderbook/internal/core/services"
)

type AuthHandler struct {
	service ports.AuthService
}

func NewAuthHandler(service ports.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*10)

	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, "invalid-json", "Bad Request", "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		WriteError(w, "validation-error", "Missing Credentials", "Email and password are required", http.StatusBadRequest)
		return
	}

	if err := h.service.Register(r.Context(), req.Email, req.Password); err != nil {
		if errors.Is(err, services.ErrUserAlreadyExists) {
			WriteError(w, "conflict", "Registration Failed", "Email already in use", http.StatusConflict)
			return
		}
		slog.Error("registration failed", "error", err, "email", req.Email)
		WriteError(w, "internal-error", "Server Error", "Could not create account", http.StatusInternalServerError)
		return
	}

	slog.Info("user registered", "email", req.Email)
	w.WriteHeader(http.StatusCreated)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*10)

	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, "invalid-json", "Bad Request", "Invalid JSON body", http.StatusBadRequest)
		return
	}

	token, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			WriteError(w, "unauthorized", "Login Failed", "Invalid email or password", http.StatusUnauthorized)
			return
		}
		slog.Error("login-error", "error", err, "email", req.Email)
		WriteError(w, "internal-error", "Server Error", "Could not create account", http.StatusInternalServerError)
		return
	}

	slog.Info("user logged in", "email", req.Email)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(loginResponse{Token: token})
}
