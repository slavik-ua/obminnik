package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const orderBookKey = "orderbook:l2"
const orderBookTTL = 60 * time.Second

type OrderBookRedisCache struct {
	client *redis.Client
}

func NewOrderBookRedisCache(client *redis.Client) *OrderBookRedisCache {
	return &OrderBookRedisCache{client: client}
}

func (r *OrderBookRedisCache) Get(ctx context.Context) ([]byte, bool, error) {
	data, err := r.client.Get(ctx, orderBookKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	return data, true, nil
}

func (r *OrderBookRedisCache) Set(ctx context.Context, data []byte) error {
	return r.client.Set(ctx, orderBookKey, data, orderBookTTL).Err()
}

func (r *OrderBookRedisCache) Invalidate(ctx context.Context) error {
	return r.client.Del(ctx, orderBookKey).Err()
}
