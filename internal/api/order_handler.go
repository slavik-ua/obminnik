package api

import (
	"encoding/json"
	"fmt"
	"log"
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
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
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
		http.Error(w, fmt.Sprintf("Request body too large or invalid: %v", err), http.StatusBadRequest)
		return
	}

	if req.Quantity <= 0 || req.Price <= 0 {
		http.Error(w, "Price and Quantity must be positive", http.StatusBadRequest)
		return
	}

	if req.Quantity > math.MaxInt32 {
		http.Error(w, "Quantity is too large", http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
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
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	log.Printf("NEW ORDER: ID: %s, PRICE: %d, SIDE: %d", newOrder.ID, newOrder.Price, newOrder.Side)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newOrder)
}
