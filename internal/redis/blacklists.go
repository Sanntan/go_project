package redis

import (
	"context"
)

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

// AddToBlacklist добавляет счет в черный список (используется для тестирования и администрирования)
func (c *Client) AddToBlacklist(accountNumber string) error {
	ctx := context.Background()
	key := "blacklist:accounts"
	return c.rdb.SAdd(ctx, key, accountNumber).Err()
}
