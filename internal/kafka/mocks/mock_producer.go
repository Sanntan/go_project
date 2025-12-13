package mocks

import (
	"bank-aml-system/internal/models"

	"github.com/stretchr/testify/mock"
)

// MockProducer является моком для kafka.Producer интерфейса
type MockProducer struct {
	mock.Mock
}

// SendTransactionEvent мок для SendTransactionEvent
func (m *MockProducer) SendTransactionEvent(event *models.KafkaTransactionEvent) error {
	args := m.Called(event)
	return args.Error(0)
}

// Close мок для Close
func (m *MockProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}
