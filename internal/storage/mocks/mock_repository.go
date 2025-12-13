package mocks

import (
	"bank-aml-system/internal/models"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockTransactionRepository является моком для storage.TransactionRepository интерфейса
type MockTransactionRepository struct {
	mock.Mock
}

// SaveTransaction мок для SaveTransaction
func (m *MockTransactionRepository) SaveTransaction(processingID string, tx *models.Transaction) error {
	args := m.Called(processingID, tx)
	return args.Error(0)
}

// UpdateTransactionAnalysis мок для UpdateTransactionAnalysis
func (m *MockTransactionRepository) UpdateTransactionAnalysis(processingID string, riskScore int, riskLevel string, analysisTime time.Time) error {
	args := m.Called(processingID, riskScore, riskLevel, analysisTime)
	return args.Error(0)
}

// GetTransactionByProcessingID мок для GetTransactionByProcessingID
func (m *MockTransactionRepository) GetTransactionByProcessingID(processingID string) (*models.TransactionStatus, error) {
	args := m.Called(processingID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TransactionStatus), args.Error(1)
}

// GetFullTransactionByProcessingID мок для GetFullTransactionByProcessingID
func (m *MockTransactionRepository) GetFullTransactionByProcessingID(processingID string) (*models.Transaction, error) {
	args := m.Called(processingID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Transaction), args.Error(1)
}

// GetAllTransactions мок для GetAllTransactions
func (m *MockTransactionRepository) GetAllTransactions(limit int) ([]*models.TransactionStatus, error) {
	args := m.Called(limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.TransactionStatus), args.Error(1)
}

// ClearAllTransactions мок для ClearAllTransactions
func (m *MockTransactionRepository) ClearAllTransactions() error {
	args := m.Called()
	return args.Error(0)
}
