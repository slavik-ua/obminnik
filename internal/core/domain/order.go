package domain

import (
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
	StatusPartial
	StatusFilled
	StatusCancelled
)

type Order struct {
	ID                uuid.UUID   `json:"id"`
	UserID            uuid.UUID   `json:"user_id"`
	CreatedAt         int64       `json:"created_at"`
	Price             int64       `json:"price"`
	Quantity          int64       `json:"quantity"`
	RemainingQuantity int64       `json:"remaining_quantity"`
	Side              OrderSide   `json:"side"`
	Status            OrderStatus `json:"status"`

	// internal doubly-linked list pointers
	parent            *PriceLevel
	next              *Order
	prev              *Order
}

// Trade records a single matched execution between a taker and a maker order
type Trade struct {
	Price        int64     `json:"price"`
	Quantity     int64     `json:"quantity"`
	TakerOrderID uuid.UUID `json:"taker_order_id"`
	MakerOrderID uuid.UUID `json:"maker_order_id"`
	
	TakerUserID  uuid.UUID `json:"taker_user_id"`
	MakerUserID  uuid.UUID `json:"maker_user_id"`
}
