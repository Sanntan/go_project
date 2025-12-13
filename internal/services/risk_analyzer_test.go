package services

import (
	"errors"
	"testing"
	"time"

	"bank-aml-system/internal/models"
	redismocks "bank-aml-system/internal/redis/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRiskAnalyzer(t *testing.T) {
	mockRedis := new(redismocks.MockClientInterface)
	service := NewRiskAnalyzer(mockRedis)

	assert.NotNil(t, service)
	impl, ok := service.(*RiskAnalyzerImpl)
	require.True(t, ok)
	assert.NotNil(t, impl.analyzer)
}

func TestRiskAnalyzer_AnalyzeTransaction_Success(t *testing.T) {
	mockRedis := new(redismocks.MockClientInterface)
	service := NewRiskAnalyzer(mockRedis)

	// Настраиваем моки
	mockRedis.On("IsHighRiskCountry", "RU").Return(false, nil)
	mockRedis.On("IsAccountBlacklisted", "ACC789012").Return(false, nil)
	mockRedis.On("GetAccountDailyCount", "ACC123456").Return(int64(2), nil)
	mockRedis.On("IncrementAccountDailyCount", "ACC123456").Return(nil)

	tx := &models.Transaction{
		TransactionID:       "TXN-001",
		AccountNumber:       "ACC123456",
		Amount:              100000.0,
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyCountry: "RU",
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		Channel:             "online",
	}

	analysis, err := service.AnalyzeTransaction(tx)

	require.NoError(t, err)
	require.NotNil(t, analysis)
	assert.LessOrEqual(t, analysis.RiskScore, 30)
	assert.Equal(t, "low", analysis.RiskLevel)

	mockRedis.AssertExpectations(t)
}

func TestRiskAnalyzer_AnalyzeTransaction_Error(t *testing.T) {
	mockRedis := new(redismocks.MockClientInterface)
	service := NewRiskAnalyzer(mockRedis)

	// Ошибка при проверке страны
	mockRedis.On("IsHighRiskCountry", "KY").Return(false, errors.New("redis error"))

	tx := &models.Transaction{
		TransactionID:       "TXN-001",
		AccountNumber:       "ACC123456",
		Amount:              100000.0,
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyCountry: "KY",
		CounterpartyAccount: "ACC789012",
		Timestamp:           time.Now(),
		Channel:             "online",
	}

	analysis, err := service.AnalyzeTransaction(tx)

	assert.Error(t, err)
	assert.Nil(t, analysis)
	assert.Contains(t, err.Error(), "redis error")

	mockRedis.AssertExpectations(t)
}
