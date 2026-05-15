package broadcast

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const slidingWindowLua = `
local key = KEYS[1]
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
local count = redis.call('ZCARD', key)
if count >= limit then
    return 0
end
redis.call('ZADD', key, now, now)
redis.call('EXPIRE', key, window)
return 1
`

// RedisRateLimiter implements a sliding-window rate limiter in Redis.
type RedisRateLimiter struct {
	client *redis.Client
	window time.Duration
	limit  int
}

// NewRedisRateLimiter creates a rate limiter with the given window and limit.
func NewRedisRateLimiter(client *redis.Client, window time.Duration, limit int) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		window: window,
		limit:  limit,
	}
}

// Allow checks if a request is allowed under the rate limit.
func (r *RedisRateLimiter) Allow(ctx context.Context, tenantID, channelID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("rate_limit:broadcast:%s:%s", tenantID, channelID)
	now := time.Now().UTC().UnixMilli()
	windowMs := r.window.Milliseconds()

	res, err := r.client.Eval(ctx, slidingWindowLua, []string{key}, windowMs, r.limit, now).Result()
	if err != nil {
		return false, fmt.Errorf("redis eval: %w", err)
	}
	allowed, ok := res.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected redis response type")
	}
	return allowed == 1, nil
}

// NoopRateLimiter always allows requests. Useful for tests.
type NoopRateLimiter struct{}

func (n *NoopRateLimiter) Allow(ctx context.Context, tenantID, channelID uuid.UUID) (bool, error) {
	return true, nil
}
