package repository

import (
	"context"
	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/db"
)

type PostgresAuthRepository struct {
	store *db.Store
}

func NewPostgresAuthRepository(store *db.Store) *PostgresAuthRepository {
	return &PostgresAuthRepository{store: store}
}

func (r *PostgresAuthRepository) CreateUser(ctx context.Context, user *domain.User) error {
	return r.store.CreateUser(ctx, db.CreateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
	})
}

func (r *PostgresAuthRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:           row.ID,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		CreatedAt:    row.CreatedAt.Time,
	}, nil
}
