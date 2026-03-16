package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrderSide int8

const (
	SideBuy OrderSide = iota
	SideSell
)

type OrderStatus int8

const (
	StatusNew OrderStatus = iota
	StatusFilled
	StatusCancelled
)

// ID (16 bytes)
// Price (8 bytes)
// Quantity (8 bytes)
// RemainingQuantity (8 bytes)
// CreatedAt (24 bytes)
// Side (1 byte)
// Status (1 byte)
// Total = 66 bytes
// With padding = 72 bytes
type Order struct {
	ID                uuid.UUID   `json:"id"`
	Price             int64       `json:"price"`
	Quantity          int64       `json:"quantity"`
	RemainingQuantity int64       `json:"remaining_quantity"`
	CreatedAt         time.Time   `json:"created_at"`
	Side              OrderSide   `json:"side"`
	Status            OrderStatus `json:"status"`
}
