package redis

import (
	"context"
	"fmt"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

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
