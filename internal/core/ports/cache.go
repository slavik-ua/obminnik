package ports

import (
	"context"
)

type OrderBookCache interface {
	Get(ctx context.Context) ([]byte, bool, error)
	Set(ctx context.Context, data []byte) error
	Invalidate(ctx context.Context) error
}
