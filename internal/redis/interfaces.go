package redis

import (
	"bank-aml-system/internal/models"
)

// ClientInterface определяет интерфейс для работы с Redis
// Это позволяет легко создавать моки для тестирования
// Реализуется типом Client
type ClientInterface interface {
	// SaveAnalysis сохраняет результаты анализа в Redis
	SaveAnalysis(processingID string, analysis *models.RiskAnalysis) error
	
	// GetAnalysis получает результаты анализа из Redis
	GetAnalysis(processingID string) (*models.RiskAnalysis, error)
	
	// IncrementRiskStats увеличивает счетчик статистики рисков
	IncrementRiskStats(riskLevel string) error
	
	// IncrementAccountDailyCount увеличивает счетчик транзакций по счету за день
	IncrementAccountDailyCount(accountNumber string) error
	
	// GetAccountDailyCount получает количество транзакций по счету за день
	GetAccountDailyCount(accountNumber string) (int64, error)
	
	// IsAccountBlacklisted проверяет, находится ли счет в черном списке
	IsAccountBlacklisted(accountNumber string) (bool, error)
	
	// IsHighRiskCountry проверяет, является ли страна высокорисковой
	IsHighRiskCountry(countryCode string) (bool, error)
	
	// InitializeBlacklists инициализирует черные списки
	InitializeBlacklists() error
	
	// AddToBlacklist добавляет счет в черный список
	AddToBlacklist(accountNumber string) error
	
	// ClearTransactionData очищает все данные транзакций из Redis
	ClearTransactionData() error
	
	// Close закрывает соединение с Redis
	Close() error
}

// Убеждаемся, что Client реализует ClientInterface
var _ ClientInterface = (*Client)(nil)

