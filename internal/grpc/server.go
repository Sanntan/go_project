package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"bank-aml-system/config"
	"bank-aml-system/internal/generator"
	"bank-aml-system/internal/kafka"
	"bank-aml-system/internal/models"
	"bank-aml-system/internal/storage"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	transaction "bank-aml-system/api/proto"
)

type TransactionGRPCServer struct {
	transaction.UnimplementedTransactionServiceServer
	repo      storage.TransactionRepository
	producer  kafka.Producer
	generator *generator.TransactionGenerator
}

func NewTransactionGRPCServer(
	repo storage.TransactionRepository,
	producer kafka.Producer,
) *TransactionGRPCServer {
	return &TransactionGRPCServer{
		repo:      repo,
		producer:  producer,
		generator: generator.NewTransactionGenerator(),
	}
}

// AnalyzeTransaction принимает транзакцию через gRPC, сохраняет её и отправляет в Kafka
// Анализ транзакции выполняется fraud сервисом асинхронно через Kafka
func (s *TransactionGRPCServer) AnalyzeTransaction(ctx context.Context, req *transaction.AnalyzeTransactionRequest) (*transaction.AnalyzeTransactionResponse, error) {
	// Парсим timestamp
	timestamp, err := time.Parse(time.RFC3339, req.Timestamp)
	if err != nil {
		timestamp = time.Now()
	}

	// Создаем транзакцию из запроса
	tx := &models.Transaction{
		TransactionID:       req.TransactionId,
		AccountNumber:       req.AccountNumber,
		Amount:              req.Amount,
		Currency:            req.Currency,
		TransactionType:     req.TransactionType,
		CounterpartyAccount: req.CounterpartyAccount,
		CounterpartyBank:    req.CounterpartyBank,
		CounterpartyCountry: req.CounterpartyCountry,
		Channel:             req.Channel,
		UserID:              req.UserId,
		BranchID:            req.BranchId,
		Timestamp:           timestamp,
	}

	// Генерируем processing_id
	processingID := "proc_" + uuid.New().String()

	// Сохраняем транзакцию в БД
	if err := s.repo.SaveTransaction(processingID, tx); err != nil {
		log.Printf("Error saving transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to save transaction: %v", err)
	}

	// Отправляем в Kafka для асинхронной обработки fraud сервисом
	event := &models.KafkaTransactionEvent{
		EventID:   "evt_" + uuid.New().String(),
		EventType: "transaction_received",
		Timestamp: time.Now(),
		Data: models.KafkaTransactionData{
			ProcessingID:        processingID,
			TransactionID:       tx.TransactionID,
			AccountNumber:       tx.AccountNumber,
			Amount:              tx.Amount,
			Currency:            tx.Currency,
			TransactionType:     tx.TransactionType,
			CounterpartyCountry: tx.CounterpartyCountry,
			Channel:             tx.Channel,
		},
	}

	if err := s.producer.SendTransactionEvent(event); err != nil {
		log.Printf("Error sending event to Kafka: %v", err)
		// Продолжаем выполнение, даже если Kafka недоступен
	}

	// Возвращаем ответ без анализа (анализ будет выполнен fraud сервисом асинхронно)
	return &transaction.AnalyzeTransactionResponse{
		ProcessingId:   processingID,
		RiskScore:      0,
		RiskLevel:      "pending",
		Flags:          []string{},
		Recommendation: "pending_analysis",
		AnalyzedAt:     time.Now().Format(time.RFC3339),
		Status:         "pending_review",
	}, nil
}

// GetTransactionStatus возвращает статус транзакции из БД
func (s *TransactionGRPCServer) GetTransactionStatus(ctx context.Context, req *transaction.GetTransactionStatusRequest) (*transaction.GetTransactionStatusResponse, error) {
	// Получаем транзакцию из БД
	tx, err := s.repo.GetTransactionByProcessingID(req.ProcessingId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Transaction not found")
	}

	if tx == nil {
		return nil, status.Errorf(codes.NotFound, "Transaction not found")
	}

	var riskScore int32
	var riskLevel string
	var flags []string
	var analysisTimestamp string

	if tx.RiskScore != nil {
		riskScore = int32(*tx.RiskScore)
	}
	if tx.RiskLevel != nil {
		riskLevel = *tx.RiskLevel
	}
	if tx.AnalysisTimestamp != nil {
		analysisTimestamp = tx.AnalysisTimestamp.Format(time.RFC3339)
	}

	return &transaction.GetTransactionStatusResponse{
		ProcessingId:      tx.ProcessingID,
		TransactionId:     tx.TransactionID,
		Status:            tx.Status,
		RiskScore:         riskScore,
		RiskLevel:         riskLevel,
		Flags:             flags,
		AnalysisTimestamp: analysisTimestamp,
	}, nil
}

// GenerateRandomTransaction генерирует случайную транзакцию
func (s *TransactionGRPCServer) GenerateRandomTransaction(ctx context.Context, req *transaction.GenerateRandomTransactionRequest) (*transaction.GenerateRandomTransactionResponse, error) {
	tx := s.generator.GenerateRandomTransaction()

	return &transaction.GenerateRandomTransactionResponse{
		TransactionId:       tx.TransactionID,
		AccountNumber:       tx.AccountNumber,
		Amount:              tx.Amount,
		Currency:            tx.Currency,
		TransactionType:     tx.TransactionType,
		CounterpartyAccount: tx.CounterpartyAccount,
		CounterpartyBank:    tx.CounterpartyBank,
		CounterpartyCountry: tx.CounterpartyCountry,
		Channel:             tx.Channel,
		UserId:              tx.UserID,
		BranchId:            tx.BranchID,
	}, nil
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// StartGRPCServer запускает gRPC сервер
func StartGRPCServer(cfg *config.Config, server *TransactionGRPCServer) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	transaction.RegisterTransactionServiceServer(s, server)

	// Включаем reflection API для grpcurl и других инструментов
	reflection.Register(s)

	log.Printf("gRPC server listening on port %d", cfg.Server.GRPCPort)
	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return nil
}
