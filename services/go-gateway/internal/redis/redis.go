package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Client wraps go-redis with application-specific helpers.
type Client struct {
	*redis.Client
}

// New creates a Redis client from a URL string.
func New(redisURL string) (*Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return &Client{Client: redis.NewClient(opt)}, nil
}

// Ping verifies the connection.
func (c *Client) Ping(ctx context.Context) error {
	return c.Client.Ping(ctx).Err()
}
