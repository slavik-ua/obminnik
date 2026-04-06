package api

import (
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/core/ports"
)

type OrderHandler struct {
	service ports.OrderService
}

func NewOrderHandler(service ports.OrderService) *OrderHandler {
	return &OrderHandler{
		service: service,
	}
}

// DTO (Data Transfer Object).
type CreateOrderRequest struct {
	Price    int64            `json:"price"`
	Quantity int64            `json:"quantity"`
	Side     domain.OrderSide `json:"side"`
}

func (h *OrderHandler) GetOrderBook(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.GetOrderBook(r.Context())
	if err != nil {
		WriteError(w, "about:blank", "Internal Server Error", "Could not fetch orderbook", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	var req CreateOrderRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		WriteError(w, "invalid-json", "Bad Request", "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Quantity <= 0 || req.Price <= 0 {
		WriteError(w, "validation-error", "Invalid Values", "Price and Quantity must be positive", http.StatusBadRequest)
		return
	}

	if req.Quantity > math.MaxInt32 {
		WriteError(w, "validation-error", "Quantity Too Large", "Quantity exceeds maximum allowed", http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	if !ok {
		WriteError(w, "unauthorized", "Unauthorized", "User ID not found in session", http.StatusUnauthorized)
		return
	}

	newOrder := domain.Order{
		ID:                uuid.New(),
		UserID:            userID,
		Price:             req.Price,
		Quantity:          req.Quantity,
		RemainingQuantity: req.Quantity,
		CreatedAt:         time.Now().Unix(),
		Side:              req.Side,
		Status:            domain.StatusNew,
	}

	_, err := h.service.PlaceOrder(r.Context(), &newOrder)
	if err != nil {
		slog.Error("failed to place order", "error", err, "user_id", userID)
		WriteError(w, "internal-error", "Placement Failed", "Order could not be processed", http.StatusInternalServerError)
		return
	}

	slog.Info("order created",
		"id", newOrder.ID,
		"price", newOrder.Price,
		"side", newOrder.Side,
		"user_id", userID,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newOrder)
}
