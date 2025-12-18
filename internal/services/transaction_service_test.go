package services

import (
	"errors"
	"testing"
	"time"

	kafkamocks "bank-aml-system/internal/kafka/mocks"
	"bank-aml-system/internal/models"
	storagemocks "bank-aml-system/internal/storage/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewTransactionService(t *testing.T) {
	mockRepo := new(storagemocks.MockTransactionRepository)
	mockProducer := new(kafkamocks.MockProducer)

	service := NewTransactionService(mockRepo, mockProducer)

	assert.NotNil(t, service)
	impl, ok := service.(*TransactionServiceImpl)
	require.True(t, ok)
	assert.Equal(t, mockRepo, impl.repo)
	assert.Equal(t, mockProducer, impl.producer)
}

func TestTransactionService_ProcessTransaction_Success(t *testing.T) {
	mockRepo := new(storagemocks.MockTransactionRepository)
	mockProducer := new(kafkamocks.MockProducer)
	service := NewTransactionService(mockRepo, mockProducer)

	req := &models.ProcessingRequest{
		Transaction: models.Transaction{
			TransactionID:       "TXN-001",
			AccountNumber:       "ACC123456",
			Amount:              100000.0,
			Currency:            "RUB",
			TransactionType:     "transfer",
			CounterpartyAccount: "ACC789012",
			CounterpartyCountry: "RU",
			Timestamp:           time.Now(),
			Channel:             "online",
		},
	}

	// Настраиваем моки
	mockRepo.On("SaveTransaction", mock.AnythingOfType("string"), &req.Transaction).Return(nil)
	mockProducer.On("SendTransactionEvent", mock.AnythingOfType("*models.KafkaTransactionEvent")).Return(nil)

	response, err := service.ProcessTransaction(req)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.NotEmpty(t, response.ProcessingID)
	assert.Contains(t, response.ProcessingID, "proc_")
	assert.Equal(t, "pending_review", response.Status)
	assert.Equal(t, "Transaction accepted for analysis", response.Message)

	mockRepo.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
}

func TestTransactionService_ProcessTransaction_RepositoryError(t *testing.T) {
	mockRepo := new(storagemocks.MockTransactionRepository)
	mockProducer := new(kafkamocks.MockProducer)
	service := NewTransactionService(mockRepo, mockProducer)

	req := &models.ProcessingRequest{
		Transaction: models.Transaction{
			TransactionID: "TXN-001",
			AccountNumber: "ACC123456",
			Amount:        100000.0,
			Currency:      "RUB",
		},
	}

	// Ошибка при сохранении в БД
	mockRepo.On("SaveTransaction", mock.AnythingOfType("string"), &req.Transaction).Return(errors.New("database error"))

	response, err := service.ProcessTransaction(req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "database error")

	mockRepo.AssertExpectations(t)
	mockProducer.AssertNotCalled(t, "SendTransactionEvent")
}

func TestTransactionService_ProcessTransaction_KafkaError(t *testing.T) {
	mockRepo := new(storagemocks.MockTransactionRepository)
	mockProducer := new(kafkamocks.MockProducer)
	service := NewTransactionService(mockRepo, mockProducer)

	req := &models.ProcessingRequest{
		Transaction: models.Transaction{
			TransactionID: "TXN-001",
			AccountNumber: "ACC123456",
			Amount:        100000.0,
			Currency:      "RUB",
		},
	}

	// Ошибка при отправке в Kafka
	mockRepo.On("SaveTransaction", mock.AnythingOfType("string"), &req.Transaction).Return(nil)
	mockProducer.On("SendTransactionEvent", mock.AnythingOfType("*models.KafkaTransactionEvent")).Return(errors.New("kafka error"))

	response, err := service.ProcessTransaction(req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "kafka error")

	mockRepo.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
}

func TestTransactionService_GetTransactionStatus_Success(t *testing.T) {
	mockRepo := new(storagemocks.MockTransactionRepository)
	mockProducer := new(kafkamocks.MockProducer)
	service := NewTransactionService(mockRepo, mockProducer)

	processingID := "proc_test_123"
	riskScore := 50
	riskLevel := "medium"
	analysisTime := time.Now()

	status := &models.TransactionStatus{
		ProcessingID:      processingID,
		TransactionID:     "TXN-001",
		Amount:            func() *float64 { v := 100000.0; return &v }(),
		Currency:          func() *string { v := "RUB"; return &v }(),
		Status:            "reviewed",
		RiskScore:         &riskScore,
		RiskLevel:         &riskLevel,
		AnalysisTimestamp: &analysisTime,
	}

	mockRepo.On("GetTransactionByProcessingID", processingID).Return(status, nil)

	response, err := service.GetTransactionStatus(processingID)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, processingID, response.ProcessingID)
	assert.Equal(t, "TXN-001", response.TransactionID)
	assert.Equal(t, "reviewed", response.Status)
	assert.Equal(t, &riskScore, response.RiskScore)
	assert.Equal(t, &riskLevel, response.RiskLevel)
	assert.Empty(t, response.Flags) // Флаги будут заполнены fraud сервисом при анализе

	mockRepo.AssertExpectations(t)
}

func TestTransactionService_GetTransactionStatus_NotFound(t *testing.T) {
	mockRepo := new(storagemocks.MockTransactionRepository)
	mockProducer := new(kafkamocks.MockProducer)
	service := NewTransactionService(mockRepo, mockProducer)

	processingID := "proc_not_found"

	mockRepo.On("GetTransactionByProcessingID", processingID).Return(nil, nil)

	response, err := service.GetTransactionStatus(processingID)

	require.NoError(t, err)
	assert.Nil(t, response)

	mockRepo.AssertExpectations(t)
}

func TestTransactionService_GetTransactionStatus_RepositoryError(t *testing.T) {
	mockRepo := new(storagemocks.MockTransactionRepository)
	mockProducer := new(kafkamocks.MockProducer)
	service := NewTransactionService(mockRepo, mockProducer)

	processingID := "proc_error"

	mockRepo.On("GetTransactionByProcessingID", processingID).Return(nil, errors.New("database error"))

	response, err := service.GetTransactionStatus(processingID)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "database error")

	mockRepo.AssertExpectations(t)
}

func TestTransactionService_GetAllTransactions_Success(t *testing.T) {
	mockRepo := new(storagemocks.MockTransactionRepository)
	mockProducer := new(kafkamocks.MockProducer)
	service := NewTransactionService(mockRepo, mockProducer)

	riskScore1 := 30
	riskLevel1 := "low"
	riskScore2 := 70
	riskLevel2 := "high"

	transactions := []*models.TransactionStatus{
		{
			ProcessingID:  "proc_1",
			TransactionID: "TXN-001",
			Status:        "reviewed",
			RiskScore:     &riskScore1,
			RiskLevel:     &riskLevel1,
		},
		{
			ProcessingID:  "proc_2",
			TransactionID: "TXN-002",
			Status:        "reviewed",
			RiskScore:     &riskScore2,
			RiskLevel:     &riskLevel2,
		},
	}

	mockRepo.On("GetAllTransactions", 100).Return(transactions, nil)

	responses, err := service.GetAllTransactions(100)

	require.NoError(t, err)
	require.Len(t, responses, 2)
	assert.Equal(t, "proc_1", responses[0].ProcessingID)
	assert.Equal(t, "proc_2", responses[1].ProcessingID)
	assert.Empty(t, responses[0].Flags) // Флаги будут заполнены fraud сервисом при анализе

	mockRepo.AssertExpectations(t)
}

func TestTransactionService_GetAllTransactions_RepositoryError(t *testing.T) {
	mockRepo := new(storagemocks.MockTransactionRepository)
	mockProducer := new(kafkamocks.MockProducer)
	service := NewTransactionService(mockRepo, mockProducer)

	mockRepo.On("GetAllTransactions", 100).Return(nil, errors.New("database error"))

	responses, err := service.GetAllTransactions(100)

	assert.Error(t, err)
	assert.Nil(t, responses)
	assert.Contains(t, err.Error(), "database error")

	mockRepo.AssertExpectations(t)
}

func TestTransactionService_ClearAllTransactions_Success(t *testing.T) {
	mockRepo := new(storagemocks.MockTransactionRepository)
	mockProducer := new(kafkamocks.MockProducer)
	service := NewTransactionService(mockRepo, mockProducer)

	mockRepo.On("ClearAllTransactions").Return(nil)

	err := service.ClearAllTransactions()

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestTransactionService_ClearAllTransactions_Error(t *testing.T) {
	mockRepo := new(storagemocks.MockTransactionRepository)
	mockProducer := new(kafkamocks.MockProducer)
	service := NewTransactionService(mockRepo, mockProducer)

	mockRepo.On("ClearAllTransactions").Return(errors.New("database error"))

	err := service.ClearAllTransactions()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")

	mockRepo.AssertExpectations(t)
}
