package services

import (
	"time"

	"github.com/google/uuid"

	"bank-aml-system/internal/kafka"
	"bank-aml-system/internal/logger"
	"bank-aml-system/internal/models"
	"bank-aml-system/internal/redis"
	"bank-aml-system/internal/storage"
)

// TransactionServiceImpl реализует интерфейс TransactionService
type TransactionServiceImpl struct {
	repo        storage.TransactionRepository
	producer    kafka.Producer
	redisClient *redis.Client // Опциональный Redis клиент для получения флагов
}

// NewTransactionService создает новый сервис транзакций
func NewTransactionService(repo storage.TransactionRepository, producer kafka.Producer) TransactionService {
	return &TransactionServiceImpl{
		repo:     repo,
		producer: producer,
	}
}

// NewTransactionServiceWithRedis создает новый сервис транзакций с поддержкой Redis
func NewTransactionServiceWithRedis(repo storage.TransactionRepository, producer kafka.Producer, redisClient *redis.Client) TransactionService {
	return &TransactionServiceImpl{
		repo:        repo,
		producer:    producer,
		redisClient: redisClient,
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

	// Логируем отправку в Kafka
	logger.LogEvent(logger.EventKafkaSent, "ingestion-service", "kafka", map[string]interface{}{
		"processing_id":  processingID,
		"event_id":       event.EventID,
		"transaction_id": req.TransactionID,
	})

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

	response := &models.TransactionStatusResponse{
		ProcessingID:      status.ProcessingID,
		TransactionID:     status.TransactionID,
		Amount:            status.Amount,
		Currency:          status.Currency,
		Status:            status.Status,
		RiskScore:         status.RiskScore,
		RiskLevel:         status.RiskLevel,
		AnalysisTimestamp: status.AnalysisTimestamp,
		Flags:             []string{}, // По умолчанию пустой массив
	}

	// Если есть Redis клиент, пытаемся получить флаги из кэша
	if s.redisClient != nil {
		analysis, err := s.redisClient.GetAnalysis(processingID)
		if err == nil && analysis != nil && analysis.Flags != nil {
			response.Flags = analysis.Flags
		}
	}

	return response, nil
}

// GetAllTransactions возвращает все транзакции
func (s *TransactionServiceImpl) GetAllTransactions(limit int) ([]*models.TransactionStatusResponse, error) {
	transactions, err := s.repo.GetAllTransactions(limit)
	if err != nil {
		return nil, err
	}

	responses := make([]*models.TransactionStatusResponse, 0, len(transactions))
	for _, tx := range transactions {
		response := &models.TransactionStatusResponse{
			ProcessingID:      tx.ProcessingID,
			TransactionID:     tx.TransactionID,
			Amount:            tx.Amount,
			Currency:          tx.Currency,
			Status:            tx.Status,
			RiskScore:         tx.RiskScore,
			RiskLevel:         tx.RiskLevel,
			AnalysisTimestamp: tx.AnalysisTimestamp,
			Flags:             []string{}, // По умолчанию пустой массив
		}

		// Если есть Redis клиент, пытаемся получить флаги из кэша
		if s.redisClient != nil {
			analysis, err := s.redisClient.GetAnalysis(tx.ProcessingID)
			if err == nil && analysis != nil && analysis.Flags != nil {
				response.Flags = analysis.Flags
			}
		}

		responses = append(responses, response)
	}

	return responses, nil
}

// ClearAllTransactions очищает все транзакции
func (s *TransactionServiceImpl) ClearAllTransactions() error {
	return s.repo.ClearAllTransactions()
}
