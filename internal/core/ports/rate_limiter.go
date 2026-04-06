package ports

import (
	"context"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}
