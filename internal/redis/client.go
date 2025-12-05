package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
	"bank-aml-system/internal/config"
	"bank-aml-system/internal/models"
)

type Client struct {
	rdb *redisv9.Client
}

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

func (c *Client) Close() error {
	return c.rdb.Close()
}

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

// IncrementRiskStats увеличивает счетчик статистики рисков
func (c *Client) IncrementRiskStats(riskLevel string) error {
	ctx := context.Background()
	key := fmt.Sprintf("risk_stats:%s", riskLevel)
	return c.rdb.Incr(ctx, key).Err()
}

// IncrementAccountDailyCount увеличивает счетчик транзакций по счету за день
func (c *Client) IncrementAccountDailyCount(accountNumber string) error {
	ctx := context.Background()
	key := fmt.Sprintf("limits:account:%s:daily:count", accountNumber)
	pipe := c.rdb.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

// GetAccountDailyCount получает количество транзакций по счету за день
func (c *Client) GetAccountDailyCount(accountNumber string) (int64, error) {
	ctx := context.Background()
	key := fmt.Sprintf("limits:account:%s:daily:count", accountNumber)
	count, err := c.rdb.Get(ctx, key).Int64()
	if err == redisv9.Nil {
		return 0, nil
	}
	return count, err
}

// IsAccountBlacklisted проверяет, находится ли счет в черном списке
func (c *Client) IsAccountBlacklisted(accountNumber string) (bool, error) {
	ctx := context.Background()
	key := "blacklist:accounts"
	return c.rdb.SIsMember(ctx, key, accountNumber).Result()
}

// IsHighRiskCountry проверяет, является ли страна высокорисковой
func (c *Client) IsHighRiskCountry(countryCode string) (bool, error) {
	ctx := context.Background()
	key := "high_risk_countries"
	return c.rdb.SIsMember(ctx, key, countryCode).Result()
}

// InitializeBlacklists инициализирует черные списки (можно расширить)
func (c *Client) InitializeBlacklists() error {
	ctx := context.Background()
	
	// Высокорисковые страны (офшорные зоны)
	highRiskCountries := []string{"VG", "KY", "BS", "PA", "SC", "MU"}
	if len(highRiskCountries) > 0 {
		key := "high_risk_countries"
		for _, country := range highRiskCountries {
			c.rdb.SAdd(ctx, key, country)
		}
	}

	return nil
}

