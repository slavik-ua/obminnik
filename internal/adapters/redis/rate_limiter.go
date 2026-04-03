package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const luaScript = `
local current = redis.call("INCR", KEYS[1])
if current == 1 then
	redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return current
`

type FixedWindowRateLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
}

func NewFixedWindowRateLimiter(client *redis.Client, limit int, window time.Duration) *FixedWindowRateLimiter {
	return &FixedWindowRateLimiter{
		client: client,
		limit: limit,
		window: window,
	}
}

func (r *FixedWindowRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	windowID := time.Now().Truncate(r.window).Unix()
	redisKey := fmt.Sprintf("ratelimit:%s:%d", key, windowID)
	windowSecs := int(r.window.Seconds())

	count, err := r.client.Eval(ctx, luaScript, []string{redisKey}, windowSecs).Int64()
	if err != nil {
		// fail-open
		return true, fmt.Errorf("rate limiter unavailable: %w", err)
	}

	return count <= int64(r.limit), nil
}