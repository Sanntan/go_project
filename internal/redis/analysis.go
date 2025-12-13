package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bank-aml-system/internal/models"

	redisv9 "github.com/redis/go-redis/v9"
)

// SaveAnalysis сохраняет результаты анализа в Redis с TTL 1 час
func (c *Client) SaveAnalysis(processingID string, analysis *models.RiskAnalysis) error {
	ctx := context.Background()
	key := fmt.Sprintf("transaction:%s:analysis", processingID)

	data, err := json.Marshal(analysis)
	if err != nil {
		return fmt.Errorf("failed to marshal analysis: %w", err)
	}

	return c.rdb.Set(ctx, key, data, time.Hour).Err()
}

// GetAnalysis получает результаты анализа из Redis
func (c *Client) GetAnalysis(processingID string) (*models.RiskAnalysis, error) {
	ctx := context.Background()
	key := fmt.Sprintf("transaction:%s:analysis", processingID)

	data, err := c.rdb.Get(ctx, key).Result()
	if err == redisv9.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis: %w", err)
	}

	var analysis models.RiskAnalysis
	if err := json.Unmarshal([]byte(data), &analysis); err != nil {
		return nil, fmt.Errorf("failed to unmarshal analysis: %w", err)
	}

	return &analysis, nil
}
