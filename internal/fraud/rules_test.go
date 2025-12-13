package fraud

import (
	"errors"
	"testing"
	"time"

	"bank-aml-system/internal/models"
	"bank-aml-system/internal/redis/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRiskAnalyzer(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	assert.NotNil(t, analyzer)
	assert.Equal(t, mockRedis, analyzer.redisClient)
}

func TestAnalyzeTransaction_LowRisk(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	// Настраиваем моки для безопасной транзакции
	mockRedis.On("IsHighRiskCountry", "RU").Return(false, nil)
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(false, nil)
	mockRedis.On("GetAccountDailyCount", "ACC123456").Return(int64(2), nil)
	mockRedis.On("IncrementAccountDailyCount", "ACC123456").Return(nil)

	tx := &models.Transaction{
		TransactionID:       "TXN-001",
		AccountNumber:       "ACC123456",
		Amount:              100000.0, // Небольшая сумма
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyCountry: "RU", // Безопасная страна
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC), // Обычное время
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	assert.LessOrEqual(t, analysis.RiskScore, 30)
	assert.Equal(t, "low", analysis.RiskLevel)
	assert.Equal(t, "auto_approve", analysis.Recommendation)
	assert.NotContains(t, analysis.Flags, "large_amount")
	assert.NotContains(t, analysis.Flags, "offshore_counterparty")

	mockRedis.AssertExpectations(t)
}

func TestAnalyzeTransaction_HighRisk_Blacklisted(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	// Настраиваем моки - счет в черном списке
	mockRedis.On("IsHighRiskCountry", "RU").Return(false, nil)
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(true, nil) // В черном списке!
	mockRedis.On("GetAccountDailyCount", "ACC123456").Return(int64(2), nil)
	mockRedis.On("IncrementAccountDailyCount", "ACC123456").Return(nil)

	tx := &models.Transaction{
		TransactionID:       "TXN-002",
		AccountNumber:       "ACC123456",
		Amount:              100000.0,
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyAccount: "ACC789012", // В черном списке
		CounterpartyCountry: "RU",
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

	mockRedis.AssertExpectations(t)
}

func TestAnalyzeTransaction_HighRisk_VeryLargeAmount_Offshore(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	// Очень крупная сумма + офшорная страна = 50 + 40 = 90 баллов (high risk)
	mockRedis.On("IsHighRiskCountry", "KY").Return(true, nil) // Офшорная страна
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(false, nil)
	mockRedis.On("GetAccountDailyCount", "ACC123456").Return(int64(2), nil)
	mockRedis.On("IncrementAccountDailyCount", "ACC123456").Return(nil)

	tx := &models.Transaction{
		TransactionID:       "TXN-003",
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

	mockRedis.AssertExpectations(t)
}

func TestAnalyzeTransaction_MediumRisk_LargeAmount(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	mockRedis.On("IsHighRiskCountry", "RU").Return(false, nil)
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(false, nil)
	mockRedis.On("GetAccountDailyCount", "ACC123456").Return(int64(2), nil)
	mockRedis.On("IncrementAccountDailyCount", "ACC123456").Return(nil)

	tx := &models.Transaction{
		TransactionID:       "TXN-004",
		AccountNumber:       "ACC123456",
		Amount:              1500000.0, // Крупная сумма (30 баллов) + ночное время (15) = 45 баллов (medium)
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyCountry: "RU",
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 3, 0, 0, 0, time.UTC), // Ночное время для medium risk
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	assert.Greater(t, analysis.RiskScore, 30)
	assert.LessOrEqual(t, analysis.RiskScore, 70)
	assert.Equal(t, "medium", analysis.RiskLevel)
	assert.Contains(t, analysis.Flags, "large_amount")
	assert.Contains(t, analysis.Flags, "unusual_time")

	mockRedis.AssertExpectations(t)
}

func TestAnalyzeTransaction_UnusualTime(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	mockRedis.On("IsHighRiskCountry", "RU").Return(false, nil)
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(false, nil)
	mockRedis.On("GetAccountDailyCount", "ACC123456").Return(int64(2), nil)
	mockRedis.On("IncrementAccountDailyCount", "ACC123456").Return(nil)

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
	assert.Greater(t, analysis.RiskScore, 0)

	mockRedis.AssertExpectations(t)
}

func TestAnalyzeTransaction_HighFrequency(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	// Высокая частота транзакций (12 транзакций >= 10)
	mockRedis.On("IsHighRiskCountry", "RU").Return(false, nil)
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(false, nil)
	mockRedis.On("GetAccountDailyCount", "ACC-HIGH-FREQ-001").Return(int64(12), nil)
	mockRedis.On("IncrementAccountDailyCount", "ACC-HIGH-FREQ-001").Return(nil)

	tx := &models.Transaction{
		TransactionID:       "TXN-006",
		AccountNumber:       "ACC-HIGH-FREQ-001",
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
	assert.GreaterOrEqual(t, analysis.RiskScore, 25)

	mockRedis.AssertExpectations(t)
}

func TestAnalyzeTransaction_MediumFrequency(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	// Средняя частота (7 транзакций >= 5, < 10)
	mockRedis.On("IsHighRiskCountry", "RU").Return(false, nil)
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(false, nil)
	mockRedis.On("GetAccountDailyCount", "ACC123456").Return(int64(7), nil)
	mockRedis.On("IncrementAccountDailyCount", "ACC123456").Return(nil)

	tx := &models.Transaction{
		TransactionID:       "TXN-007",
		AccountNumber:       "ACC123456",
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

	assert.Contains(t, analysis.Flags, "medium_frequency")
	assert.GreaterOrEqual(t, analysis.RiskScore, 10)

	mockRedis.AssertExpectations(t)
}

func TestAnalyzeTransaction_InternationalTransfer(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	mockRedis.On("IsHighRiskCountry", "US").Return(false, nil)
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(false, nil)
	mockRedis.On("GetAccountDailyCount", "ACC123456").Return(int64(2), nil)
	mockRedis.On("IncrementAccountDailyCount", "ACC123456").Return(nil)

	tx := &models.Transaction{
		TransactionID:       "TXN-008",
		AccountNumber:       "ACC123456",
		Amount:              100000.0,
		Currency:            "RUB",
		TransactionType:     "international_transfer", // Международный перевод
		CounterpartyCountry: "US",
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	assert.Contains(t, analysis.Flags, "international_transfer")
	assert.GreaterOrEqual(t, analysis.RiskScore, 20)

	mockRedis.AssertExpectations(t)
}

func TestAnalyzeTransaction_ATMTransaction(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	mockRedis.On("IsHighRiskCountry", "RU").Return(false, nil)
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(false, nil)
	mockRedis.On("GetAccountDailyCount", "ACC123456").Return(int64(2), nil)
	mockRedis.On("IncrementAccountDailyCount", "ACC123456").Return(nil)

	tx := &models.Transaction{
		TransactionID:       "TXN-009",
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
	assert.GreaterOrEqual(t, analysis.RiskScore, 12)

	mockRedis.AssertExpectations(t)
}

func TestAnalyzeTransaction_RoundAmount(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	mockRedis.On("IsHighRiskCountry", "RU").Return(false, nil)
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(false, nil)
	mockRedis.On("GetAccountDailyCount", "ACC123456").Return(int64(2), nil)
	mockRedis.On("IncrementAccountDailyCount", "ACC123456").Return(nil)

	tx := &models.Transaction{
		TransactionID:       "TXN-010",
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
	assert.Contains(t, analysis.Flags, "large_amount") // Также должна быть крупная сумма

	mockRedis.AssertExpectations(t)
}

func TestAnalyzeTransaction_HighRiskCurrency(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	mockRedis.On("IsHighRiskCountry", "RU").Return(false, nil)
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(false, nil)
	mockRedis.On("GetAccountDailyCount", "ACC123456").Return(int64(2), nil)
	mockRedis.On("IncrementAccountDailyCount", "ACC123456").Return(nil)

	tx := &models.Transaction{
		TransactionID:       "TXN-011",
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
	assert.GreaterOrEqual(t, analysis.RiskScore, 8)

	mockRedis.AssertExpectations(t)
}

// Тесты на ошибочные сценарии

func TestAnalyzeTransaction_RedisError_IsHighRiskCountry(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	// Ошибка при проверке страны
	mockRedis.On("IsHighRiskCountry", "KY").Return(false, errors.New("redis connection error"))

	tx := &models.Transaction{
		TransactionID:       "TXN-012",
		AccountNumber:       "ACC123456",
		Amount:              100000.0,
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyCountry: "KY",
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	assert.Error(t, err)
	assert.Nil(t, analysis)
	assert.Contains(t, err.Error(), "redis connection error")

	mockRedis.AssertExpectations(t)
}

func TestAnalyzeTransaction_RedisError_IsAccountBlacklisted(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	// Ошибка при проверке черного списка
	mockRedis.On("IsHighRiskCountry", "RU").Return(false, nil)
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(false, errors.New("redis connection error"))

	tx := &models.Transaction{
		TransactionID:       "TXN-013",
		AccountNumber:       "ACC123456",
		Amount:              100000.0,
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyCountry: "RU",
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	assert.Error(t, err)
	assert.Nil(t, analysis)
	assert.Contains(t, err.Error(), "redis connection error")

	mockRedis.AssertExpectations(t)
}

func TestAnalyzeTransaction_RedisError_GetAccountDailyCount(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	// Ошибка при получении счетчика транзакций
	mockRedis.On("IsHighRiskCountry", "RU").Return(false, nil)
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(false, nil)
	mockRedis.On("GetAccountDailyCount", "ACC123456").Return(int64(0), errors.New("redis connection error"))

	tx := &models.Transaction{
		TransactionID:       "TXN-014",
		AccountNumber:       "ACC123456",
		Amount:              100000.0,
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyCountry: "RU",
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	assert.Error(t, err)
	assert.Nil(t, analysis)
	assert.Contains(t, err.Error(), "redis connection error")

	mockRedis.AssertExpectations(t)
}

func TestAnalyzeTransaction_RedisError_IncrementAccountDailyCount(t *testing.T) {
	mockRedis := new(mocks.MockClientInterface)
	analyzer := NewRiskAnalyzer(mockRedis)

	// Ошибка при увеличении счетчика
	mockRedis.On("IsHighRiskCountry", "RU").Return(false, nil)
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(false, nil)
	mockRedis.On("GetAccountDailyCount", "ACC123456").Return(int64(2), nil)
	mockRedis.On("IncrementAccountDailyCount", "ACC123456").Return(errors.New("redis connection error"))

	tx := &models.Transaction{
		TransactionID:       "TXN-015",
		AccountNumber:       "ACC123456",
		Amount:              100000.0,
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyCountry: "RU",
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Channel:             "online",
	}

	analysis, err := analyzer.AnalyzeTransaction(tx)
	assert.Error(t, err)
	assert.Nil(t, analysis)
	assert.Contains(t, err.Error(), "redis connection error")

	mockRedis.AssertExpectations(t)
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
		{"Zero score", 0, "low"},
		{"Negative score", -10, "low"},
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
		{"Zero score", 0, "auto_approve"},
		{"Negative score", -10, "auto_approve"},
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
		{"Empty string", "", false},
		{"Unknown country", "XX", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOffshoreCountry(tt.country)
			assert.Equal(t, tt.expected, result)
		})
	}
}
