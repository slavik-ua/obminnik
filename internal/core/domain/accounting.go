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
	UserID      uuid.UUID
	AssetSymbol string
	Available   int64
	Locked      int64
}

func ToFixedPoint(val int64) int64 {
	return val * Decimals
}
