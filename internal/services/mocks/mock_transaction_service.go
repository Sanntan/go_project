package mocks

import (
	"bank-aml-system/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockTransactionService является моком для services.TransactionService интерфейса
type MockTransactionService struct {
	mock.Mock
}

// ProcessTransaction мок для ProcessTransaction
func (m *MockTransactionService) ProcessTransaction(req *models.ProcessingRequest) (*models.ProcessingResponse, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessingResponse), args.Error(1)
}

// GetTransactionStatus мок для GetTransactionStatus
func (m *MockTransactionService) GetTransactionStatus(processingID string) (*models.TransactionStatusResponse, error) {
	args := m.Called(processingID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TransactionStatusResponse), args.Error(1)
}

// GetAllTransactions мок для GetAllTransactions
func (m *MockTransactionService) GetAllTransactions(limit int) ([]*models.TransactionStatusResponse, error) {
	args := m.Called(limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.TransactionStatusResponse), args.Error(1)
}

// ClearAllTransactions мок для ClearAllTransactions
func (m *MockTransactionService) ClearAllTransactions() error {
	args := m.Called()
	return args.Error(0)
}

