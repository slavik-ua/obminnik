package domain

import (
	"time"
	"strings"
	"bytes"
	"errors"
	"encoding/json"

	"github.com/google/uuid"
)

type OrderSide int8

const (
	SideBuy OrderSide = iota
	SideSell
)

func (s *OrderSide) UnmarshalJSON(b []byte) error {
	str := strings.ToUpper(string(bytes.Trim(b, `"`)))

	switch str {
	case "BUY":
		*s = SideBuy
	case "SELL":
		*s = SideSell
	default:
		return errors.New("invalid order side: must be BUY or SELL")
	}

	return nil
}

func (s OrderSide) MarshalJSON() ([]byte, error) {
	var str string

	switch s {
	case SideBuy:
		str = "BUY"
	case SideSell:
		str = "SELL"
	default:
		return nil, errors.New("invalid order side value")
	}

	return json.Marshal(str)
}

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
// CreatedAt (8 bytes)
// Side (1 byte)
// Status (1 byte)
// Total = 74 bytes
// With padding = 80 bytes
type Order struct {
	ID                uuid.UUID   `json:"id"`
	CreatedAt         int64       `json:"created_at"`
	Price             int64       `json:"price"`
	Quantity          int64       `json:"quantity"`
	RemainingQuantity int64       `json:"remaining_quantity"`
	parent            *PriceLevel
	next              *Order
	prev              *Order
	Side              OrderSide   `json:"side"`
	Status            OrderStatus `json:"status"`
}
