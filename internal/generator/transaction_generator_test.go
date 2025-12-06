package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransactionGenerator(t *testing.T) {
	gen := NewTransactionGenerator()
	require.NotNil(t, gen)
	assert.NotNil(t, gen.rand)
}

func TestTransactionGenerator_GenerateTransaction_LowRisk(t *testing.T) {
	gen := NewTransactionGenerator()

	tx := gen.GenerateTransaction("low")
	require.NotNil(t, tx)

	// Проверяем базовые поля
	assert.NotEmpty(t, tx.TransactionID)
	assert.NotEmpty(t, tx.AccountNumber)
	assert.NotEmpty(t, tx.Currency)
	assert.NotEmpty(t, tx.TransactionType)
	assert.NotEmpty(t, tx.Channel)
	assert.NotEmpty(t, tx.UserID)
	assert.NotEmpty(t, tx.BranchID)

	// Проверяем, что сумма небольшая (до 500k)
	assert.Less(t, tx.Amount, 500000.0)
	assert.Greater(t, tx.Amount, 0.0)

	// Проверяем, что страна не офшорная
	offshoreCountries := map[string]bool{
		"VG": true, "KY": true, "BS": true, "PA": true,
		"SC": true, "MU": true, "CH": true,
	}
	if tx.CounterpartyCountry != "" {
		assert.False(t, offshoreCountries[tx.CounterpartyCountry],
			"Low risk transaction should not have offshore country")
	}
}

func TestTransactionGenerator_GenerateTransaction_MediumRisk(t *testing.T) {
	gen := NewTransactionGenerator()

	tx := gen.GenerateTransaction("medium")
	require.NotNil(t, tx)

	// Проверяем базовые поля
	assert.NotEmpty(t, tx.TransactionID)
	assert.NotEmpty(t, tx.AccountNumber)

	// Для medium risk должна быть либо офшорная страна, либо крупная сумма, либо ночное время
	hasOffshore := false
	offshoreCountries := map[string]bool{
		"VG": true, "KY": true, "BS": true, "PA": true,
		"SC": true, "MU": true, "CH": true,
	}
	if tx.CounterpartyCountry != "" {
		hasOffshore = offshoreCountries[tx.CounterpartyCountry]
	}

	hasLargeAmount := tx.Amount >= 1000000.0
	hasNightTime := tx.Timestamp.Hour() >= 0 && tx.Timestamp.Hour() < 6

	// Хотя бы одно условие должно выполняться
	assert.True(t, hasOffshore || hasLargeAmount || hasNightTime,
		"Medium risk transaction should have at least one risk factor")
}

func TestTransactionGenerator_GenerateTransaction_HighRisk(t *testing.T) {
	gen := NewTransactionGenerator()

	tx := gen.GenerateTransaction("high")
	require.NotNil(t, tx)

	// Проверяем базовые поля
	assert.NotEmpty(t, tx.TransactionID)
	assert.NotEmpty(t, tx.AccountNumber)

	// Для high risk должна быть крупная сумма и офшорная страна или ночное время
	hasLargeAmount := tx.Amount >= 1000000.0
	assert.True(t, hasLargeAmount, "High risk transaction should have large amount")
}

func TestTransactionGenerator_GenerateTransaction_InvalidRisk(t *testing.T) {
	gen := NewTransactionGenerator()

	// Неизвестный уровень риска должен генерировать low risk транзакцию
	tx := gen.GenerateTransaction("invalid")
	require.NotNil(t, tx)

	assert.NotEmpty(t, tx.TransactionID)
	assert.Less(t, tx.Amount, 500000.0)
}

func TestTransactionGenerator_GenerateRandomTransaction(t *testing.T) {
	gen := NewTransactionGenerator()

	tx := gen.GenerateRandomTransaction()
	require.NotNil(t, tx)

	// Проверяем базовые поля
	assert.NotEmpty(t, tx.TransactionID)
	assert.NotEmpty(t, tx.AccountNumber)
	assert.NotEmpty(t, tx.Currency)
	assert.NotEmpty(t, tx.TransactionType)
	assert.NotEmpty(t, tx.Channel)

	// Проверяем диапазон суммы
	assert.GreaterOrEqual(t, tx.Amount, 1000.0)
	assert.LessOrEqual(t, tx.Amount, 10000000.0)
}

func TestTransactionGenerator_GenerateTransaction_UniqueIDs(t *testing.T) {
	gen := NewTransactionGenerator()

	// Генерируем несколько транзакций
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		tx := gen.GenerateTransaction("low")
		require.NotNil(t, tx)
		
		// Проверяем уникальность ID
		assert.False(t, ids[tx.TransactionID], "Transaction ID should be unique")
		ids[tx.TransactionID] = true
	}
}

func TestTransactionGenerator_RoundToTwoDecimals(t *testing.T) {
	gen := NewTransactionGenerator()

	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"Simple", 123.456, 123.46},
		{"Simple", 123.454, 123.45},
		{"Large", 1000000.123, 1000000.12},
		{"Small", 0.001, 0.00},
		{"Integer", 100.0, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.roundToTwoDecimals(tt.input)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestTransactionGenerator_GetRandomSafeCountry(t *testing.T) {
	gen := NewTransactionGenerator()

	safeCountries := map[string]bool{
		"US": true, "GB": true, "RU": true, "DE": true,
		"FR": true, "IT": true, "ES": true, "NL": true,
		"BE": true, "PL": true,
	}

	// Генерируем несколько раз и проверяем, что все страны из списка
	for i := 0; i < 20; i++ {
		country := gen.getRandomSafeCountry()
		assert.True(t, safeCountries[country], "Country %s should be in safe countries list", country)
	}
}

func TestTransactionGenerator_GetRandomOffshoreCountry(t *testing.T) {
	gen := NewTransactionGenerator()

	offshoreCountries := map[string]bool{
		"VG": true, "KY": true, "CH": true, "BS": true,
		"PA": true, "SC": true, "MU": true,
	}

	// Генерируем несколько раз и проверяем, что все страны из списка
	for i := 0; i < 20; i++ {
		country := gen.getRandomOffshoreCountry()
		assert.True(t, offshoreCountries[country], "Country %s should be in offshore countries list", country)
	}
}

func TestTransactionGenerator_GetRandomBank(t *testing.T) {
	gen := NewTransactionGenerator()

	banks := map[string]bool{
		"Sberbank": true, "VTB": true, "Alfa Bank": true,
		"Gazprombank": true, "Raiffeisen": true, "Tinkoff": true,
	}

	// Генерируем несколько раз и проверяем, что все банки из списка
	for i := 0; i < 20; i++ {
		bank := gen.getRandomBank()
		assert.True(t, banks[bank], "Bank %s should be in banks list", bank)
	}
}

func TestTransactionGenerator_GetRandomOffshoreBank(t *testing.T) {
	gen := NewTransactionGenerator()

	offshoreBanks := map[string]bool{
		"UBS": true, "Credit Suisse": true, "HSBC Offshore": true,
		"Cayman National Bank": true, "BVI Bank": true, "Swiss Private Bank": true,
	}

	// Генерируем несколько раз и проверяем, что все банки из списка
	for i := 0; i < 20; i++ {
		bank := gen.getRandomOffshoreBank()
		assert.True(t, offshoreBanks[bank], "Bank %s should be in offshore banks list", bank)
	}
}

func TestTransactionGenerator_TransactionFields(t *testing.T) {
	gen := NewTransactionGenerator()

	tx := gen.GenerateTransaction("low")
	require.NotNil(t, tx)

	// Проверяем формат TransactionID
	assert.Contains(t, tx.TransactionID, "TXN-AUTO-")

	// Проверяем формат AccountNumber
	assert.Contains(t, tx.AccountNumber, "ACC")

	// Проверяем формат UserID
	assert.Contains(t, tx.UserID, "user")

	// Проверяем формат BranchID
	assert.Contains(t, tx.BranchID, "branch")
}

func TestTransactionGenerator_CurrencyAndType(t *testing.T) {
	gen := NewTransactionGenerator()

	// Генерируем случайные транзакции
	currencies := make(map[string]bool)
	types := make(map[string]bool)

	for i := 0; i < 50; i++ {
		tx := gen.GenerateRandomTransaction()
		if tx.Currency != "" {
			currencies[tx.Currency] = true
		}
		if tx.TransactionType != "" {
			types[tx.TransactionType] = true
		}
	}

	// Проверяем, что генерируются разные валюты и типы
	assert.Greater(t, len(currencies), 1, "Should generate different currencies")
	assert.Greater(t, len(types), 1, "Should generate different transaction types")
}

