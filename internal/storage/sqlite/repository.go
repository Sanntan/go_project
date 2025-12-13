package sqlite

import (
	"time"

	"bank-aml-system/internal/models"
	"bank-aml-system/internal/storage"
)

// Repository реализует интерфейс TransactionRepository для SQLite
type Repository struct {
	storage *SQLiteStorage
}

// NewRepository создает новый репозиторий SQLite
func NewRepository(storage *SQLiteStorage) storage.TransactionRepository {
	return &Repository{storage: storage}
}

// SaveTransaction сохраняет транзакцию в БД
func (r *Repository) SaveTransaction(processingID string, tx *models.Transaction) error {
	return r.storage.SaveTransaction(processingID, tx)
}

// UpdateTransactionAnalysis обновляет результаты анализа транзакции
func (r *Repository) UpdateTransactionAnalysis(processingID string, riskScore int, riskLevel string, analysisTime time.Time) error {
	return r.storage.UpdateTransactionAnalysis(processingID, riskScore, riskLevel, analysisTime)
}

// GetTransactionByProcessingID получает транзакцию по processing_id
func (r *Repository) GetTransactionByProcessingID(processingID string) (*models.TransactionStatus, error) {
	return r.storage.GetTransactionByProcessingID(processingID)
}

// GetFullTransactionByProcessingID получает полную транзакцию со всеми полями
func (r *Repository) GetFullTransactionByProcessingID(processingID string) (*models.Transaction, error) {
	return r.storage.GetFullTransactionByProcessingID(processingID)
}

// GetAllTransactions получает все транзакции из БД
func (r *Repository) GetAllTransactions(limit int) ([]*models.TransactionStatus, error) {
	return r.storage.GetAllTransactions(limit)
}

// ClearAllTransactions удаляет все транзакции из БД
func (r *Repository) ClearAllTransactions() error {
	return r.storage.ClearAllTransactions()
}

