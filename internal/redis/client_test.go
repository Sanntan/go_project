package redis

import (
	"context"
	"testing"
	"time"

	"bank-aml-system/internal/config"
	"bank-aml-system/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*Client, func()) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "127.0.0.1", // Используем IPv4 вместо localhost
			Port:     "6379",
			Password: "",
		},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return nil, nil
	}

	// Очищаем тестовые данные перед тестом
	ctx := context.Background()
	client.rdb.FlushDB(ctx)

	cleanup := func() {
		// Очищаем тестовые данные после теста
		ctx := context.Background()
		client.rdb.FlushDB(ctx)
		client.Close()
	}

	return client, cleanup
}

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "127.0.0.1", // Используем IPv4 вместо localhost
			Port:     "6379",
			Password: "",
		},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return
	}
	defer client.Close()

	assert.NotNil(t, client)
	assert.NotNil(t, client.rdb)
}

func TestClient_SaveAnalysis(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	if client == nil {
		return
	}
	defer cleanup()

	processingID := "PROC-001"
	analysis := &models.RiskAnalysis{
		RiskScore:     50,
		RiskLevel:     "medium",
		Flags:         []string{"large_amount", "offshore_counterparty"},
		Recommendation: "log_only",
		AnalyzedAt:    time.Now(),
	}

	err := client.SaveAnalysis(processingID, analysis)
	require.NoError(t, err)

	// Проверяем, что данные сохранены
	saved, err := client.GetAnalysis(processingID)
	require.NoError(t, err)
	require.NotNil(t, saved)

	assert.Equal(t, analysis.RiskScore, saved.RiskScore)
	assert.Equal(t, analysis.RiskLevel, saved.RiskLevel)
	assert.Equal(t, analysis.Flags, saved.Flags)
	assert.Equal(t, analysis.Recommendation, saved.Recommendation)
}

func TestClient_GetAnalysis(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	if client == nil {
		return
	}
	defer cleanup()

	processingID := "PROC-002"
	analysis := &models.RiskAnalysis{
		RiskScore:     75,
		RiskLevel:     "high",
		Flags:         []string{"very_large_amount", "blacklisted_counterparty"},
		Recommendation: "require_verification",
		AnalyzedAt:    time.Now(),
	}

	// Сохраняем анализ
	err := client.SaveAnalysis(processingID, analysis)
	require.NoError(t, err)

	// Получаем анализ
	saved, err := client.GetAnalysis(processingID)
	require.NoError(t, err)
	require.NotNil(t, saved)

	assert.Equal(t, analysis.RiskScore, saved.RiskScore)
	assert.Equal(t, analysis.RiskLevel, saved.RiskLevel)
	assert.Equal(t, analysis.Flags, saved.Flags)
	assert.Equal(t, analysis.Recommendation, saved.Recommendation)

	// Проверяем несуществующий анализ
	notFound, err := client.GetAnalysis("NONEXISTENT")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestClient_IncrementRiskStats(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	if client == nil {
		return
	}
	defer cleanup()

	riskLevel := "high"

	// Увеличиваем счетчик
	err := client.IncrementRiskStats(riskLevel)
	require.NoError(t, err)

	err = client.IncrementRiskStats(riskLevel)
	require.NoError(t, err)

	// Проверяем значение
	ctx := context.Background()
	key := "risk_stats:" + riskLevel
	count, err := client.rdb.Get(ctx, key).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestClient_IncrementAccountDailyCount(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	if client == nil {
		return
	}
	defer cleanup()

	accountNumber := "ACC123456"

	// Увеличиваем счетчик
	err := client.IncrementAccountDailyCount(accountNumber)
	require.NoError(t, err)

	err = client.IncrementAccountDailyCount(accountNumber)
	require.NoError(t, err)

	// Проверяем значение
	count, err := client.GetAccountDailyCount(accountNumber)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestClient_GetAccountDailyCount(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	if client == nil {
		return
	}
	defer cleanup()

	accountNumber := "ACC789012"

	// Для нового счета должно быть 0
	count, err := client.GetAccountDailyCount(accountNumber)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Увеличиваем счетчик
	err = client.IncrementAccountDailyCount(accountNumber)
	require.NoError(t, err)

	// Проверяем значение
	count, err = client.GetAccountDailyCount(accountNumber)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestClient_IsAccountBlacklisted(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	if client == nil {
		return
	}
	defer cleanup()

	accountNumber := "ACC-BLACKLIST-001"

	// Добавляем счет в черный список
	ctx := context.Background()
	key := "blacklist:accounts"
	err := client.rdb.SAdd(ctx, key, accountNumber).Err()
	require.NoError(t, err)

	// Проверяем, что счет в черном списке
	isBlacklisted, err := client.IsAccountBlacklisted(accountNumber)
	require.NoError(t, err)
	assert.True(t, isBlacklisted)

	// Проверяем счет, которого нет в черном списке
	isBlacklisted, err = client.IsAccountBlacklisted("ACC-CLEAN-001")
	require.NoError(t, err)
	assert.False(t, isBlacklisted)
}

func TestClient_IsHighRiskCountry(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	if client == nil {
		return
	}
	defer cleanup()

	// Инициализируем черные списки
	err := client.InitializeBlacklists()
	require.NoError(t, err)

	// Проверяем высокорисковую страну
	isHighRisk, err := client.IsHighRiskCountry("KY")
	require.NoError(t, err)
	assert.True(t, isHighRisk)

	// Проверяем обычную страну
	isHighRisk, err = client.IsHighRiskCountry("RU")
	require.NoError(t, err)
	assert.False(t, isHighRisk)
}

func TestClient_InitializeBlacklists(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	if client == nil {
		return
	}
	defer cleanup()

	err := client.InitializeBlacklists()
	require.NoError(t, err)

	// Проверяем, что высокорисковые страны добавлены
	highRiskCountries := []string{"VG", "KY", "BS", "PA", "SC", "MU"}
	for _, country := range highRiskCountries {
		isHighRisk, err := client.IsHighRiskCountry(country)
		require.NoError(t, err)
		assert.True(t, isHighRisk, "Country %s should be in high risk list", country)
	}
}

func TestClient_ClearTransactionData(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	if client == nil {
		return
	}
	defer cleanup()

	// Создаем тестовые данные
	processingID := "PROC-CLEAR-001"
	analysis := &models.RiskAnalysis{
		RiskScore:     50,
		RiskLevel:     "medium",
		Flags:         []string{"test"},
		Recommendation: "test",
		AnalyzedAt:    time.Now(),
	}

	err := client.SaveAnalysis(processingID, analysis)
	require.NoError(t, err)

	err = client.IncrementRiskStats("high")
	require.NoError(t, err)

	err = client.IncrementAccountDailyCount("ACC123456")
	require.NoError(t, err)

	// Очищаем данные транзакций
	err = client.ClearTransactionData()
	require.NoError(t, err)

	// Проверяем, что данные удалены
	saved, err := client.GetAnalysis(processingID)
	require.NoError(t, err)
	assert.Nil(t, saved)

	// Проверяем, что черные списки сохранены
	err = client.InitializeBlacklists()
	require.NoError(t, err)
	isHighRisk, err := client.IsHighRiskCountry("KY")
	require.NoError(t, err)
	assert.True(t, isHighRisk)
}

func TestClient_Close(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	if client == nil {
		return
	}
	defer cleanup()

	err := client.Close()
	require.NoError(t, err)

	// Проверяем, что после закрытия нельзя выполнить операцию
	err = client.IncrementRiskStats("test")
	assert.Error(t, err)
}

func TestClient_AnalysisTTL(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	if client == nil {
		return
	}
	defer cleanup()

	processingID := "PROC-TTL-001"
	analysis := &models.RiskAnalysis{
		RiskScore:     50,
		RiskLevel:     "medium",
		Flags:         []string{"test"},
		Recommendation: "test",
		AnalyzedAt:    time.Now(),
	}

	err := client.SaveAnalysis(processingID, analysis)
	require.NoError(t, err)

	// Проверяем TTL (должен быть около 1 часа)
	ctx := context.Background()
	key := "transaction:" + processingID + ":analysis"
	ttl, err := client.rdb.TTL(ctx, key).Result()
	require.NoError(t, err)
	
	// TTL должен быть больше 0 и меньше 2 часов (с учетом погрешности)
	assert.Greater(t, ttl, time.Duration(0))
	assert.Less(t, ttl, 2*time.Hour)
}

