package api

import (
	"context"

	transaction "bank-aml-system/api/proto"
)

// RESTHandler определяет интерфейс для REST API обработчиков
type RESTHandler interface {
	// HandleTransaction обрабатывает POST запрос на создание транзакции
	HandleTransaction(ctx context.Context, req interface{}) (interface{}, error)
	
	// GetTransactionStatus обрабатывает GET запрос на получение статуса транзакции
	GetTransactionStatus(ctx context.Context, processingID string) (interface{}, error)
	
	// GetAllTransactions обрабатывает GET запрос на получение всех транзакций
	GetAllTransactions(ctx context.Context, limit int) (interface{}, error)
	
	// ClearAllTransactions обрабатывает DELETE запрос на очистку всех транзакций
	ClearAllTransactions(ctx context.Context) error
}

// GRPCHandler определяет интерфейс для gRPC обработчиков
type GRPCHandler interface {
	// AnalyzeTransaction анализирует транзакцию на предмет рисков через gRPC
	AnalyzeTransaction(ctx context.Context, req *transaction.AnalyzeTransactionRequest) (*transaction.AnalyzeTransactionResponse, error)
	
	// GetTransactionStatus возвращает статус транзакции
	GetTransactionStatus(ctx context.Context, req *transaction.GetTransactionStatusRequest) (*transaction.GetTransactionStatusResponse, error)
	
	// GenerateRandomTransaction генерирует случайную транзакцию
	GenerateRandomTransaction(ctx context.Context, req *transaction.GenerateRandomTransactionRequest) (*transaction.GenerateRandomTransactionResponse, error)
}

