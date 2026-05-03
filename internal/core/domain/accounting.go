package domain

import (
	"errors"
	"github.com/google/uuid"
)

const (
	Decimals = 100_000_000
)

var (
	ErrInsufficientFunds = errors.New("accounting: insufficient available balance")
)

type BalanceRecord struct {
	UserID      uuid.UUID `json:"user_id"`
	AssetSymbol string    `json:"asset_symbol"`
	Available   int64     `json:"available"`
	Locked      int64     `json:"locked"`
}

func ToFixedPoint(val int64) int64 {
	return val * Decimals
}
