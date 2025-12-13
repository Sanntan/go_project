package services

import (
	"bank-aml-system/internal/fraud"
	"bank-aml-system/internal/models"
	"bank-aml-system/internal/redis"
)

// RiskAnalyzerImpl реализует интерфейс RiskAnalyzer
type RiskAnalyzerImpl struct {
	analyzer *fraud.RiskAnalyzer
}

// NewRiskAnalyzer создает новый анализатор рисков
func NewRiskAnalyzer(redisClient redis.ClientInterface) RiskAnalyzer {
	analyzer := fraud.NewRiskAnalyzer(redisClient)
	return &RiskAnalyzerImpl{analyzer: analyzer}
}

// AnalyzeTransaction выполняет полный анализ транзакции на предмет рисков
func (r *RiskAnalyzerImpl) AnalyzeTransaction(tx *models.Transaction) (*models.RiskAnalysis, error) {
	return r.analyzer.AnalyzeTransaction(tx)
}

