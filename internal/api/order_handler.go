package api

import (
	"encoding/json"
	"math"
	"net/http"
	"time"
	"log"
	"fmt"

	"github.com/google/uuid"
	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/core/ports"
)

type OrderHandler struct {
	store ports.OrderRepository
}

func NewOrderHandler(store ports.OrderRepository) *OrderHandler {
	return &OrderHandler{
		store: store,
	}
}

// DTO (Data Transfer Object).
type CreateOrderRequest struct {
	Price    int64            `json:"price"`
	Quantity int64            `json:"quantity"`
	Side     domain.OrderSide `json:"side"`
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	var req CreateOrderRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Request body too large or invalid: %v", err), http.StatusRequestEntityTooLarge)
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

	newOrder := domain.Order{
		ID:                uuid.New(),
		Price:             req.Price,
		Quantity:          req.Quantity,
		RemainingQuantity: req.Quantity,
		CreatedAt:         time.Now(),
		Side:              req.Side,
		Status:            domain.StatusNew,
	}

	err := h.store.Create(r.Context(), &newOrder)
	if err != nil {
		http.Error(w, "Side should be 0, 1, BUY, SELL", http.StatusInternalServerError)
		return
	}

	log.Printf("NEW ORDER: ID: %s, PRICE: %d, SIDE: %d", newOrder.ID, newOrder.Price, newOrder.Side)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newOrder)
}
