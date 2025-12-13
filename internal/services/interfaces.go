package services

import (
	"bank-aml-system/internal/models"
)

// TransactionService определяет интерфейс для работы с транзакциями
type TransactionService interface {
	// ProcessTransaction обрабатывает транзакцию
	ProcessTransaction(req *models.ProcessingRequest) (*models.ProcessingResponse, error)
	
	// GetTransactionStatus возвращает статус транзакции
	GetTransactionStatus(processingID string) (*models.TransactionStatusResponse, error)
	
	// GetAllTransactions возвращает все транзакции
	GetAllTransactions(limit int) ([]*models.TransactionStatusResponse, error)
	
	// ClearAllTransactions очищает все транзакции
	ClearAllTransactions() error
}

// RiskAnalyzer определяет интерфейс для анализа рисков
type RiskAnalyzer interface {
	// AnalyzeTransaction выполняет полный анализ транзакции на предмет рисков
	AnalyzeTransaction(tx *models.Transaction) (*models.RiskAnalysis, error)
}

