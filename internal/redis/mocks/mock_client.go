package mocks

import (
	"bank-aml-system/internal/models"

	"github.com/stretchr/testify/mock"
)

// MockClientInterface является моком для redis.ClientInterface интерфейса
type MockClientInterface struct {
	mock.Mock
}

// SaveAnalysis мок для SaveAnalysis
func (m *MockClientInterface) SaveAnalysis(processingID string, analysis *models.RiskAnalysis) error {
	args := m.Called(processingID, analysis)
	return args.Error(0)
}

// GetAnalysis мок для GetAnalysis
func (m *MockClientInterface) GetAnalysis(processingID string) (*models.RiskAnalysis, error) {
	args := m.Called(processingID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RiskAnalysis), args.Error(1)
}

// IncrementAccountDailyCount мок для IncrementAccountDailyCount
func (m *MockClientInterface) IncrementAccountDailyCount(accountNumber string) error {
	args := m.Called(accountNumber)
	return args.Error(0)
}

// GetAccountDailyCount мок для GetAccountDailyCount
func (m *MockClientInterface) GetAccountDailyCount(accountNumber string) (int64, error) {
	args := m.Called(accountNumber)
	return args.Get(0).(int64), args.Error(1)
}

// IsAccountBlacklisted мок для IsAccountBlacklisted
func (m *MockClientInterface) IsAccountBlacklisted(accountNumber string) (bool, error) {
	args := m.Called(accountNumber)
	return args.Bool(0), args.Error(1)
}

// IsHighRiskCountry мок для IsHighRiskCountry
func (m *MockClientInterface) IsHighRiskCountry(countryCode string) (bool, error) {
	args := m.Called(countryCode)
	return args.Bool(0), args.Error(1)
}

// InitializeBlacklists мок для InitializeBlacklists
func (m *MockClientInterface) InitializeBlacklists() error {
	args := m.Called()
	return args.Error(0)
}

// AddToBlacklist мок для AddToBlacklist
func (m *MockClientInterface) AddToBlacklist(accountNumber string) error {
	args := m.Called(accountNumber)
	return args.Error(0)
}

// Close мок для Close
func (m *MockClientInterface) Close() error {
	args := m.Called()
	return args.Error(0)
}
