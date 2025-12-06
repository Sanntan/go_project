package database

import (
	"os"
	"testing"
	"time"

	"bank-aml-system/internal/config"
	"bank-aml-system/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*Repository, func()) {
	// Создаем временный файл БД для тестов
	tmpFile := "test_" + time.Now().Format("20060102150405") + ".db"
	
	cfg := &config.Config{
		DB: config.DBConfig{
			DBPath: tmpFile,
		},
	}

	db, err := NewSQLiteDB(cfg)
	require.NoError(t, err)

	repo := NewRepository(db)

	// Функция очистки
	cleanup := func() {
		db.Close()
		os.Remove(tmpFile)
	}

	return repo, cleanup
}

func TestNewRepository(t *testing.T) {
	cfg := &config.Config{
		DB: config.DBConfig{
			DBPath: ":memory:",
		},
	}

	db, err := NewSQLiteDB(cfg)
	require.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestRepository_SaveTransaction(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	processingID := "PROC-001"
	tx := &models.Transaction{
		TransactionID:       "TXN-001",
		AccountNumber:       "ACC123456",
		Amount:              100000.0,
		Currency:            "RUB",
		TransactionType:     "transfer",
		CounterpartyAccount: "ACC789012",
		CounterpartyBank:    "Bank Test",
		CounterpartyCountry: "RU",
		Timestamp:           time.Now(),
		Channel:             "online",
		UserID:              "user123",
		BranchID:            "branch001",
	}

	err := repo.SaveTransaction(processingID, tx)
	require.NoError(t, err)

	// Проверяем, что транзакция сохранена
	saved, err := repo.GetTransactionByProcessingID(processingID)
	require.NoError(t, err)
	require.NotNil(t, saved)

	assert.Equal(t, processingID, saved.ProcessingID)
	assert.Equal(t, tx.TransactionID, saved.TransactionID)
	assert.Equal(t, "pending_review", saved.Status)
}

func TestRepository_UpdateTransactionAnalysis(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	processingID := "PROC-002"
	tx := &models.Transaction{
		TransactionID:   "TXN-002",
		AccountNumber:    "ACC123456",
		Amount:           100000.0,
		Currency:         "RUB",
		TransactionType: "transfer",
		Timestamp:        time.Now(),
	}

	// Сохраняем транзакцию
	err := repo.SaveTransaction(processingID, tx)
	require.NoError(t, err)

	// Обновляем анализ
	riskScore := 50
	riskLevel := "medium"
	analysisTime := time.Now()

	err = repo.UpdateTransactionAnalysis(processingID, riskScore, riskLevel, analysisTime)
	require.NoError(t, err)

	// Проверяем обновление
	updated, err := repo.GetTransactionByProcessingID(processingID)
	require.NoError(t, err)
	require.NotNil(t, updated)

	assert.Equal(t, "reviewed", updated.Status)
	assert.NotNil(t, updated.RiskScore)
	assert.Equal(t, riskScore, *updated.RiskScore)
	assert.NotNil(t, updated.RiskLevel)
	assert.Equal(t, riskLevel, *updated.RiskLevel)
	assert.NotNil(t, updated.AnalysisTimestamp)
}

func TestRepository_GetTransactionByProcessingID(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	processingID := "PROC-003"
	tx := &models.Transaction{
		TransactionID:   "TXN-003",
		AccountNumber:    "ACC123456",
		Amount:           200000.0,
		Currency:         "USD",
		TransactionType: "transfer",
		Timestamp:        time.Now(),
	}

	err := repo.SaveTransaction(processingID, tx)
	require.NoError(t, err)

	// Получаем транзакцию
	saved, err := repo.GetTransactionByProcessingID(processingID)
	require.NoError(t, err)
	require.NotNil(t, saved)

	assert.Equal(t, processingID, saved.ProcessingID)
	assert.Equal(t, tx.TransactionID, saved.TransactionID)
	assert.NotNil(t, saved.Amount)
	assert.Equal(t, tx.Amount, *saved.Amount)
	assert.NotNil(t, saved.Currency)
	assert.Equal(t, tx.Currency, *saved.Currency)

	// Проверяем несуществующую транзакцию
	notFound, err := repo.GetTransactionByProcessingID("NONEXISTENT")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestRepository_GetFullTransactionByProcessingID(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	processingID := "PROC-004"
	tx := &models.Transaction{
		TransactionID:       "TXN-004",
		AccountNumber:       "ACC123456",
		Amount:              300000.0,
		Currency:            "EUR",
		TransactionType:     "international_transfer",
		CounterpartyAccount: "ACC789012",
		CounterpartyBank:    "Bank Europe",
		CounterpartyCountry: "DE",
		Timestamp:           time.Now(),
		Channel:             "mobile",
		UserID:              "user456",
		BranchID:            "branch002",
	}

	err := repo.SaveTransaction(processingID, tx)
	require.NoError(t, err)

	// Получаем полную транзакцию
	fullTx, err := repo.GetFullTransactionByProcessingID(processingID)
	require.NoError(t, err)
	require.NotNil(t, fullTx)

	assert.Equal(t, tx.TransactionID, fullTx.TransactionID)
	assert.Equal(t, tx.AccountNumber, fullTx.AccountNumber)
	assert.Equal(t, tx.Amount, fullTx.Amount)
	assert.Equal(t, tx.Currency, fullTx.Currency)
	assert.Equal(t, tx.TransactionType, fullTx.TransactionType)
	assert.Equal(t, tx.CounterpartyAccount, fullTx.CounterpartyAccount)
	assert.Equal(t, tx.CounterpartyBank, fullTx.CounterpartyBank)
	assert.Equal(t, tx.CounterpartyCountry, fullTx.CounterpartyCountry)
	assert.Equal(t, tx.Channel, fullTx.Channel)
	assert.Equal(t, tx.UserID, fullTx.UserID)
	assert.Equal(t, tx.BranchID, fullTx.BranchID)

	// Проверяем несуществующую транзакцию
	notFound, err := repo.GetFullTransactionByProcessingID("NONEXISTENT")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestRepository_GetAllTransactions(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	// Создаем несколько транзакций
	for i := 0; i < 5; i++ {
		processingID := "PROC-" + string(rune('0'+i))
		tx := &models.Transaction{
			TransactionID:   "TXN-00" + string(rune('0'+i)),
			AccountNumber:    "ACC123456",
			Amount:           float64(100000 + i*10000),
			Currency:         "RUB",
			TransactionType: "transfer",
			Timestamp:        time.Now().Add(time.Duration(i) * time.Second),
		}

		err := repo.SaveTransaction(processingID, tx)
		require.NoError(t, err)
	}

	// Получаем все транзакции
	transactions, err := repo.GetAllTransactions(10)
	require.NoError(t, err)
	assert.Len(t, transactions, 5)

	// Проверяем порядок (должны быть отсортированы по created_at DESC)
	assert.GreaterOrEqual(t, transactions[0].CreatedAt.Unix(), transactions[1].CreatedAt.Unix())

	// Проверяем лимит
	limited, err := repo.GetAllTransactions(3)
	require.NoError(t, err)
	assert.Len(t, limited, 3)
}

func TestRepository_ClearAllTransactions(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	// Создаем несколько транзакций
	for i := 0; i < 3; i++ {
		processingID := "PROC-CLEAR-" + string(rune('0'+i))
		tx := &models.Transaction{
			TransactionID:   "TXN-CLEAR-" + string(rune('0'+i)),
			AccountNumber:    "ACC123456",
			Amount:           100000.0,
			Currency:         "RUB",
			TransactionType: "transfer",
			Timestamp:        time.Now(),
		}

		err := repo.SaveTransaction(processingID, tx)
		require.NoError(t, err)
	}

	// Проверяем, что транзакции есть
	transactions, err := repo.GetAllTransactions(10)
	require.NoError(t, err)
	assert.Len(t, transactions, 3)

	// Очищаем все транзакции
	err = repo.ClearAllTransactions()
	require.NoError(t, err)

	// Проверяем, что транзакций нет
	transactions, err = repo.GetAllTransactions(10)
	require.NoError(t, err)
	assert.Len(t, transactions, 0)
}

func TestRepository_TransactionStatusFlow(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	processingID := "PROC-FLOW-001"
	tx := &models.Transaction{
		TransactionID:   "TXN-FLOW-001",
		AccountNumber:    "ACC123456",
		Amount:           500000.0,
		Currency:         "RUB",
		TransactionType: "transfer",
		Timestamp:        time.Now(),
	}

	// 1. Сохраняем транзакцию
	err := repo.SaveTransaction(processingID, tx)
	require.NoError(t, err)

	// 2. Проверяем статус pending_review
	status1, err := repo.GetTransactionByProcessingID(processingID)
	require.NoError(t, err)
	require.NotNil(t, status1)
	assert.Equal(t, "pending_review", status1.Status)

	// 3. Обновляем анализ
	err = repo.UpdateTransactionAnalysis(processingID, 75, "high", time.Now())
	require.NoError(t, err)

	// 4. Проверяем статус reviewed
	status2, err := repo.GetTransactionByProcessingID(processingID)
	require.NoError(t, err)
	require.NotNil(t, status2)
	assert.Equal(t, "reviewed", status2.Status)
	assert.NotNil(t, status2.RiskScore)
	assert.Equal(t, 75, *status2.RiskScore)
	assert.NotNil(t, status2.RiskLevel)
	assert.Equal(t, "high", *status2.RiskLevel)
}

