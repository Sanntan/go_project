package fraud

import (
	"testing"
	"time"

	"bank-aml-system/internal/config"
	"bank-aml-system/internal/models"
	"bank-aml-system/internal/redis"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "127.0.0.1", // Используем IPv4 вместо localhost
			Port:     "6379",
			Password: "",
		},
	}

	client, err := redis.NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return nil, nil
	}

	// Очищаем тестовые данные перед тестом
	client.ClearTransactionData()

	// Инициализируем черные списки
	client.InitializeBlacklists()

	cleanup := func() {
		// Очищаем тестовые данные после теста
		client.ClearTransactionData()
		client.Close()
	}

	return client, cleanup
}

func TestNewRiskAnalyzer(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	if redisClient == nil {
		return
	}
	defer cleanup()

	analyzer := NewRiskAnalyzer(redisClient)
	
	assert.NotNil(t, analyzer)
	assert.Equal(t, redisClient, analyzer.redisClient)
}

func TestAnalyzeTransaction_LowRisk(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	if redisClient == nil {
		return
	}
	defer cleanup()

	analyzer := NewRiskAnalyzer(redisClient)

	// Используем уникальный счет для каждого теста, чтобы избежать накопления данных
	tx := &models.Transaction{
		TransactionID:       "TXN-001",
		AccountNumber:       "ACC-LOW-RISK-001", // Уникальный счет
		Amount:              100000.0,            // Небольшая сумма
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyCountry: "RU", // Обычная страна
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC), // Дневное время
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	assert.LessOrEqual(t, analysis.RiskScore, 30)
	assert.Equal(t, "low", analysis.RiskLevel)
	assert.Equal(t, "auto_approve", analysis.Recommendation)
}

func TestAnalyzeTransaction_MediumRisk(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	if redisClient == nil {
		return
	}
	defer cleanup()

	analyzer := NewRiskAnalyzer(redisClient)

	// Используем уникальный счет для каждого теста
	tx := &models.Transaction{
		TransactionID:       "TXN-002",
		AccountNumber:       "ACC-MEDIUM-RISK-001", // Уникальный счет
		Amount:              600000.0,               // Средняя сумма
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyCountry: "KY", // Офшорная страна (уже в списке после InitializeBlacklists)
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	assert.Greater(t, analysis.RiskScore, 30)
	assert.LessOrEqual(t, analysis.RiskScore, 70)
	assert.Equal(t, "medium", analysis.RiskLevel)
	assert.Equal(t, "log_only", analysis.Recommendation)
	assert.Contains(t, analysis.Flags, "offshore_counterparty")
}

func TestAnalyzeTransaction_HighRisk_Blacklisted(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	if redisClient == nil {
		return
	}
	defer cleanup()

	// Добавляем счет в черный список
	err := redisClient.AddToBlacklist("ACC789012")
	require.NoError(t, err)

	analyzer := NewRiskAnalyzer(redisClient)

	tx := &models.Transaction{
		TransactionID:       "TXN-003",
		AccountNumber:       "ACC123456",
		Amount:              100000.0,
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyAccount: "ACC789012", // В черном списке
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	assert.GreaterOrEqual(t, analysis.RiskScore, 100)
	assert.Equal(t, "high", analysis.RiskLevel)
	assert.Equal(t, "require_verification", analysis.Recommendation)
	assert.Contains(t, analysis.Flags, "blacklisted_counterparty")
}

func TestAnalyzeTransaction_HighRisk_VeryLargeAmount(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	if redisClient == nil {
		return
	}
	defer cleanup()

	analyzer := NewRiskAnalyzer(redisClient)

	// Очень крупная сумма + офшорная страна = 50 + 40 = 90 баллов (high risk)
	tx := &models.Transaction{
		TransactionID:       "TXN-004",
		AccountNumber:       "ACC123456",
		Amount:              6000000.0, // Очень крупная сумма (50 баллов)
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyCountry: "KY", // Офшорная страна (40 баллов)
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	assert.Greater(t, analysis.RiskScore, 70)
	assert.Equal(t, "high", analysis.RiskLevel)
	assert.Contains(t, analysis.Flags, "very_large_amount")
	assert.Contains(t, analysis.Flags, "offshore_counterparty")
}

func TestAnalyzeTransaction_UnusualTime(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	if redisClient == nil {
		return
	}
	defer cleanup()

	analyzer := NewRiskAnalyzer(redisClient)

	tx := &models.Transaction{
		TransactionID:       "TXN-005",
		AccountNumber:       "ACC123456",
		Amount:              100000.0,
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyCountry: "RU",
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 3, 0, 0, 0, time.UTC), // Ночное время
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	assert.Contains(t, analysis.Flags, "unusual_time")
}

func TestAnalyzeTransaction_HighFrequency(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	if redisClient == nil {
		return
	}
	defer cleanup()

	// Устанавливаем высокую частоту транзакций (11 транзакций)
	// После AnalyzeTransaction счетчик увеличится до 12, что >= 10 (HighFrequencyThreshold)
	accountNumber := "ACC-HIGH-FREQ-001" // Уникальный счет для этого теста
	for i := 0; i < 11; i++ {
		err := redisClient.IncrementAccountDailyCount(accountNumber)
		require.NoError(t, err)
	}

	analyzer := NewRiskAnalyzer(redisClient)

	tx := &models.Transaction{
		TransactionID:       "TXN-006",
		AccountNumber:       accountNumber,
		Amount:              100000.0,
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyCountry: "RU",
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	assert.Contains(t, analysis.Flags, "high_frequency")
}

func TestAnalyzeTransaction_InternationalTransfer(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	if redisClient == nil {
		return
	}
	defer cleanup()

	analyzer := NewRiskAnalyzer(redisClient)

	tx := &models.Transaction{
		TransactionID:       "TXN-007",
		AccountNumber:       "ACC123456",
		Amount:              100000.0,
		Currency:            "RUB",
		TransactionType:     "international_transfer",
		CounterpartyCountry: "US",
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	assert.Contains(t, analysis.Flags, "international_transfer")
}

func TestAnalyzeTransaction_ATMTransaction(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	if redisClient == nil {
		return
	}
	defer cleanup()

	analyzer := NewRiskAnalyzer(redisClient)

	tx := &models.Transaction{
		TransactionID:       "TXN-008",
		AccountNumber:       "ACC123456",
		Amount:              600000.0, // Крупная сумма через ATM
		Currency:            "RUB",
		TransactionType:     "withdrawal",
		CounterpartyCountry: "RU",
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Channel:             "atm",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	assert.Contains(t, analysis.Flags, "large_atm_transaction")
}

func TestAnalyzeTransaction_RoundAmount(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	if redisClient == nil {
		return
	}
	defer cleanup()

	analyzer := NewRiskAnalyzer(redisClient)

	tx := &models.Transaction{
		TransactionID:       "TXN-009",
		AccountNumber:       "ACC123456",
		Amount:              1000000.0, // Круглая сумма
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyCountry: "RU",
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	assert.Contains(t, analysis.Flags, "round_amount")
}

func TestAnalyzeTransaction_HighRiskCurrency(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	if redisClient == nil {
		return
	}
	defer cleanup()

	analyzer := NewRiskAnalyzer(redisClient)

	tx := &models.Transaction{
		TransactionID:       "TXN-010",
		AccountNumber:       "ACC123456",
		Amount:              100000.0,
		Currency:            "CHF", // Высокорисковая валюта
		TransactionType:     "transfer",
		CounterpartyCountry: "RU",
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	assert.Contains(t, analysis.Flags, "high_risk_currency")
}

func TestCalculateRiskLevel(t *testing.T) {
	tests := []struct {
		name     string
		score    int
		expected string
	}{
		{"Low risk", 20, "low"},
		{"Low risk boundary", 30, "low"},
		{"Medium risk", 50, "medium"},
		{"Medium risk boundary", 70, "medium"},
		{"High risk", 80, "high"},
		{"High risk", 100, "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateRiskLevel(tt.score)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetActionRecommendation(t *testing.T) {
	tests := []struct {
		name     string
		score    int
		expected string
	}{
		{"Auto approve", 20, "auto_approve"},
		{"Auto approve boundary", 30, "auto_approve"},
		{"Log only", 50, "log_only"},
		{"Log only boundary", 70, "log_only"},
		{"Require verification", 80, "require_verification"},
		{"Require verification", 100, "require_verification"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getActionRecommendation(tt.score)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsOffshoreCountry(t *testing.T) {
	tests := []struct {
		name     string
		country  string
		expected bool
	}{
		{"Offshore - VG", "VG", true},
		{"Offshore - KY", "KY", true},
		{"Offshore - BS", "BS", true},
		{"Offshore - PA", "PA", true},
		{"Offshore - SC", "SC", true},
		{"Offshore - MU", "MU", true},
		{"Offshore - CH", "CH", true},
		{"Safe - RU", "RU", false},
		{"Safe - US", "US", false},
		{"Safe - GB", "GB", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOffshoreCountry(tt.country)
			assert.Equal(t, tt.expected, result)
		})
	}
}

