package redis

import (
	"context"
	"fmt"

	"bank-aml-system/internal/config"

	redisv9 "github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redisv9.Client
}

// NewClient создает новое подключение к Redis
func NewClient(cfg *config.Config) (*Client, error) {
	rdb := redisv9.NewClient(&redisv9.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       0,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{rdb: rdb}, nil
}

// Close закрывает соединение с Redis
func (c *Client) Close() error {
	return c.rdb.Close()
}
