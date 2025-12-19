package fraud_detection

import (
	"log"

	"bank-aml-system/internal/logger"
	"bank-aml-system/internal/models"
	"bank-aml-system/internal/redis"
	"bank-aml-system/internal/services"
	"bank-aml-system/internal/storage"
)

// processTransaction обрабатывает транзакцию из Kafka события
func processTransaction(
	event *models.KafkaTransactionEvent,
	repo storage.TransactionRepository,
	redisClient *redis.Client,
	riskAnalyzer services.RiskAnalyzer,
) error {
	log.Printf("Processing transaction: %s", event.Data.ProcessingID)

	logger.LogEvent(logger.EventKafkaReceived, "fraud-detection-service", "kafka", map[string]interface{}{
		"processing_id": event.Data.ProcessingID,
		"event_id":      event.EventID,
		"topic":         "bank.transactions.received",
	})

	logger.LogEvent(logger.EventAnalysisStarted, "fraud-detection-service", "analyzer", map[string]interface{}{
		"processing_id": event.Data.ProcessingID,
	})

	// Получаем транзакцию с retry логикой (встроенной в GetFullTransactionByProcessingID)
	// Это обрабатывает race condition, когда Kafka событие приходит раньше, чем транзакция сохраняется
	tx, err := repo.GetFullTransactionByProcessingID(event.Data.ProcessingID)
	if err != nil {
		log.Printf("Error getting transaction %s: %v", event.Data.ProcessingID, err)
		return err
	}
	if tx == nil {
		log.Printf("Transaction not found after retries: %s (may have been processed already or not saved yet)", event.Data.ProcessingID)
		// Не возвращаем ошибку, чтобы не блокировать обработку других транзакций
		// Транзакция может быть уже обработана или еще не успела сохраниться
		return nil
	}

	analysis, err := riskAnalyzer.AnalyzeTransaction(tx)
	if err != nil {
		log.Printf("Error analyzing transaction: %v", err)
		return err
	}

	logger.LogEvent(logger.EventAnalysisCompleted, "fraud-detection-service", "analyzer", map[string]interface{}{
		"processing_id": event.Data.ProcessingID,
		"risk_score":    analysis.RiskScore,
		"risk_level":    analysis.RiskLevel,
		"flags":         analysis.Flags,
	})

	if err := redisClient.SaveAnalysis(event.Data.ProcessingID, analysis); err != nil {
		log.Printf("Error saving analysis to Redis: %v", err)
	} else {
		logger.LogEvent(logger.EventRedisSaved, "fraud-detection-service", "redis", map[string]interface{}{
			"processing_id": event.Data.ProcessingID,
			"risk_score":    analysis.RiskScore,
			"risk_level":    analysis.RiskLevel,
		})
	}

	if err := repo.UpdateTransactionAnalysis(
		event.Data.ProcessingID,
		analysis.RiskScore,
		analysis.RiskLevel,
		analysis.AnalyzedAt,
	); err != nil {
		log.Printf("Error updating transaction in DB: %v", err)
		return err
	}

	logger.LogEvent(logger.EventDBUpdated, "fraud-detection-service", "sqlite", map[string]interface{}{
		"processing_id": event.Data.ProcessingID,
		"status":        "reviewed",
		"risk_score":    analysis.RiskScore,
		"risk_level":    analysis.RiskLevel,
	})

	if err := redisClient.IncrementRiskStats(analysis.RiskLevel); err != nil {
		log.Printf("Error updating risk stats: %v", err)
	}

	log.Printf("Transaction %s analyzed: risk_score=%d, risk_level=%s",
		event.Data.ProcessingID, analysis.RiskScore, analysis.RiskLevel)

	return nil
}
