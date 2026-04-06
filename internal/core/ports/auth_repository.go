package ports

import (
	"context"
	"simple-orderbook/internal/core/domain"
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
}
