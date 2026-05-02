package ports

import (
	"context"

	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/db"

	"github.com/google/uuid"
)

type AccountRepository interface {
	LockFunds(ctx context.Context, userID uuid.UUID, asset string, amount int64, refID uuid.UUID) error
	UnlockFunds(ctx context.Context, userID uuid.UUID, asset string, amount int64, refID uuid.UUID) error
	ListAllBalances(ctx context.Context) ([]domain.BalanceRecord, error)
	Deposit(ctx context.Context, q *db.Queries, userID uuid.UUID, asset string, amount int64) error
}
