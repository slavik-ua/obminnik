package repository

import (
	"context"
	"github.com/google/uuid"
	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/db"
)

type PostgresAccountRepository struct {
	store *db.Store
	idGen domain.IDGenerator
}

func NewPostgresAccountRepository(store *db.Store, idGen domain.IDGenerator) *PostgresAccountRepository {
	return &PostgresAccountRepository{
		store: store,
		idGen: idGen,
	}
}

func (r *PostgresAccountRepository) LockFunds(ctx context.Context, userID uuid.UUID, asset string, amount int64, refID uuid.UUID) error {
	return r.store.ExecTx(ctx, func(q *db.Queries) error {
		_, err := q.UpdateBalanceLock(ctx, db.UpdateBalanceLockParams{
			UserID:      userID,
			AssetSymbol: asset,
			Amount:      amount,
		})
		if err != nil {
			return domain.ErrInsufficientFunds
		}

		_, err = q.CreateLedgerEntries(ctx, []db.CreateLedgerEntriesParams{
			{ID: r.idGen.Next(), UserID: userID, AssetSymbol: asset, Amount: -amount, BalanceType: "AVAILABLE", ReferenceType: "ORDER_LOCK", ReferenceID: refID},
			{ID: r.idGen.Next(), UserID: userID, AssetSymbol: asset, Amount: amount, BalanceType: "LOCKED", ReferenceType: "ORDER_LOCK", ReferenceID: refID},
		})

		return err
	})
}

func (r *PostgresAccountRepository) UnlockFunds(ctx context.Context, userID uuid.UUID, asset string, amount int64, refID uuid.UUID) error {
	return r.store.ExecTx(ctx, func(q *db.Queries) error {
		_, err := q.UpdateBalanceUnlock(ctx, db.UpdateBalanceUnlockParams{
			UserID:      userID,
			AssetSymbol: asset,
			Amount:      amount,
		})
		if err != nil {
			return domain.ErrInsufficientFunds
		}

		_, err = q.CreateLedgerEntries(ctx, []db.CreateLedgerEntriesParams{
			{ID: r.idGen.Next(), UserID: userID, AssetSymbol: asset, Amount: amount, BalanceType: "AVAILABLE", ReferenceType: "ORDER_UNLOCK", ReferenceID: refID},
			{ID: r.idGen.Next(), UserID: userID, AssetSymbol: asset, Amount: -amount, BalanceType: "LOCKED", ReferenceType: "ORDER_UNLOCK", ReferenceID: refID},
		})

		return err
	})
}

func (r *PostgresAccountRepository) ListAllBalances(ctx context.Context) ([]domain.BalanceRecord, error) {
	rows, err := r.store.ListAllBalances(ctx)
	if err != nil {
		return nil, err
	}

	records := make([]domain.BalanceRecord, len(rows))

	for i, row := range rows {
		records[i] = domain.BalanceRecord{
			UserID:      row.UserID,
			AssetSymbol: row.AssetSymbol,
			Available:   row.Available,
			Locked:      row.Locked,
		}
	}

	return records, nil
}

func (r *PostgresAccountRepository) Deposit(ctx context.Context, q *db.Queries, userID uuid.UUID, asset string, amount int64) error {
	_, err := q.UpsertBalance(ctx, db.UpsertBalanceParams{
		UserID:      userID,
		AssetSymbol: asset,
		Available:   amount,
	})

	return err
}
