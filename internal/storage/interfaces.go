package storage

import (
	"time"

	"bank-aml-system/internal/models"
)

// TransactionRepository определяет интерфейс для работы с транзакциями в хранилище
type TransactionRepository interface {
	// SaveTransaction сохраняет транзакцию в БД со статусом pending_review
	SaveTransaction(processingID string, tx *models.Transaction) error
	
	// UpdateTransactionAnalysis обновляет результаты анализа транзакции
	UpdateTransactionAnalysis(processingID string, riskScore int, riskLevel string, analysisTime time.Time) error
	
	// GetTransactionByProcessingID получает транзакцию по processing_id
	GetTransactionByProcessingID(processingID string) (*models.TransactionStatus, error)
	
	// GetFullTransactionByProcessingID получает полную транзакцию со всеми полями
	GetFullTransactionByProcessingID(processingID string) (*models.Transaction, error)
	
	// GetAllTransactions получает все транзакции из БД
	GetAllTransactions(limit int) ([]*models.TransactionStatus, error)
	
	// ClearAllTransactions удаляет все транзакции из БД
	ClearAllTransactions() error
}

