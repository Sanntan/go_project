package services

import (
	"github.com/google/uuid"
	"time"

	"bank-aml-system/internal/models"
	"bank-aml-system/internal/storage"
	"bank-aml-system/internal/kafka"
)

// TransactionServiceImpl реализует интерфейс TransactionService
type TransactionServiceImpl struct {
	repo     storage.TransactionRepository
	producer kafka.Producer
}

// NewTransactionService создает новый сервис транзакций
func NewTransactionService(repo storage.TransactionRepository, producer kafka.Producer) TransactionService {
	return &TransactionServiceImpl{
		repo:     repo,
		producer: producer,
	}
}

// ProcessTransaction обрабатывает транзакцию
func (s *TransactionServiceImpl) ProcessTransaction(req *models.ProcessingRequest) (*models.ProcessingResponse, error) {
	processingID := "proc_" + uuid.New().String()

	// Сохраняем транзакцию в БД
	if err := s.repo.SaveTransaction(processingID, &req.Transaction); err != nil {
		return nil, err
	}

	// Создаем событие для Kafka
	event := &models.KafkaTransactionEvent{
		EventID:   "evt_" + uuid.New().String(),
		EventType: "transaction_received",
		Timestamp: time.Now(),
		Data: models.KafkaTransactionData{
			ProcessingID:        processingID,
			TransactionID:       req.TransactionID,
			AccountNumber:       req.AccountNumber,
			Amount:              req.Amount,
			Currency:            req.Currency,
			TransactionType:     req.TransactionType,
			CounterpartyCountry: req.CounterpartyCountry,
			Channel:             req.Channel,
		},
	}

	// Отправляем событие в Kafka
	if err := s.producer.SendTransactionEvent(event); err != nil {
		return nil, err
	}

	return &models.ProcessingResponse{
		ProcessingID: processingID,
		Status:       "pending_review",
		Message:      "Transaction accepted for analysis",
	}, nil
}

// GetTransactionStatus возвращает статус транзакции
func (s *TransactionServiceImpl) GetTransactionStatus(processingID string) (*models.TransactionStatusResponse, error) {
	status, err := s.repo.GetTransactionByProcessingID(processingID)
	if err != nil {
		return nil, err
	}

	if status == nil {
		return nil, nil
	}

	return &models.TransactionStatusResponse{
		ProcessingID:      status.ProcessingID,
		TransactionID:     status.TransactionID,
		Amount:            status.Amount,
		Currency:          status.Currency,
		Status:            status.Status,
		RiskScore:         status.RiskScore,
		RiskLevel:         status.RiskLevel,
		AnalysisTimestamp: status.AnalysisTimestamp,
	}, nil
}

// GetAllTransactions возвращает все транзакции
func (s *TransactionServiceImpl) GetAllTransactions(limit int) ([]*models.TransactionStatusResponse, error) {
	transactions, err := s.repo.GetAllTransactions(limit)
	if err != nil {
		return nil, err
	}

	responses := make([]*models.TransactionStatusResponse, 0, len(transactions))
	for _, tx := range transactions {
		responses = append(responses, &models.TransactionStatusResponse{
			ProcessingID:      tx.ProcessingID,
			TransactionID:     tx.TransactionID,
			Amount:            tx.Amount,
			Currency:          tx.Currency,
			Status:            tx.Status,
			RiskScore:         tx.RiskScore,
			RiskLevel:         tx.RiskLevel,
			AnalysisTimestamp: tx.AnalysisTimestamp,
		})
	}

	return responses, nil
}

// ClearAllTransactions очищает все транзакции
func (s *TransactionServiceImpl) ClearAllTransactions() error {
	return s.repo.ClearAllTransactions()
}

