package redis

import (
	"context"
	"fmt"
)

// ClearTransactionData очищает все данные транзакций из Redis (но сохраняет черные списки)
func (c *Client) ClearTransactionData() error {
	ctx := context.Background()

	// Удаляем все ключи, связанные с транзакциями
	patterns := []string{
		"transaction:*",
		"risk_stats:*",
		"limits:account:*",
	}

	for _, pattern := range patterns {
		iter := c.rdb.Scan(ctx, 0, pattern, 0).Iterator()
		for iter.Next(ctx) {
			c.rdb.Del(ctx, iter.Val())
		}
		if err := iter.Err(); err != nil {
			return fmt.Errorf("failed to clear pattern %s: %w", pattern, err)
		}
	}

	return nil
}
