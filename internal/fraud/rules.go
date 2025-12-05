package fraud

import (
	"time"

	"bank-aml-system/internal/models"
	"bank-aml-system/internal/redis"
)

const (
	LargeAmountThreshold = 1000000.0 // 1 млн руб
	HighFrequencyThreshold = 10      // 10 транзакций в день
)

type RiskAnalyzer struct {
	redisClient *redis.Client
}

func NewRiskAnalyzer(redisClient *redis.Client) *RiskAnalyzer {
	return &RiskAnalyzer{
		redisClient: redisClient,
	}
}

// AnalyzeTransaction выполняет полный анализ транзакции на предмет рисков
func (r *RiskAnalyzer) AnalyzeTransaction(tx *models.Transaction) (*models.RiskAnalysis, error) {
	score := 0
	var flags []string

	// 1. Проверка крупной суммы
	if tx.Amount > LargeAmountThreshold {
		score += 30
		flags = append(flags, "large_amount")
	}

	// 2. Проверка офшорной юрисдикции
	if tx.CounterpartyCountry != "" {
		isHighRisk, err := r.redisClient.IsHighRiskCountry(tx.CounterpartyCountry)
		if err != nil {
			return nil, err
		}
		if isHighRisk {
			score += 40
			flags = append(flags, "offshore_counterparty")
		}
	}

	// 3. Проверка черного списка
	if tx.CounterpartyAccount != "" {
		isBlacklisted, err := r.redisClient.IsAccountBlacklisted(tx.CounterpartyAccount)
		if err != nil {
			return nil, err
		}
		if isBlacklisted {
			score += 100 // Автоматически high risk
			flags = append(flags, "blacklisted_counterparty")
		}
	}

	// 4. Проверка необычного времени (ночные операции: 00:00 - 06:00)
	hour := tx.Timestamp.Hour()
	if hour >= 0 && hour < 6 {
		score += 15
		flags = append(flags, "unusual_time")
	}

	// 5. Проверка частоты операций
	dailyCount, err := r.redisClient.GetAccountDailyCount(tx.AccountNumber)
	if err != nil {
		return nil, err
	}
	if dailyCount >= HighFrequencyThreshold {
		score += 25
		flags = append(flags, "high_frequency")
	}

	// Увеличиваем счетчик транзакций по счету
	if err := r.redisClient.IncrementAccountDailyCount(tx.AccountNumber); err != nil {
		return nil, err
	}

	// Определяем уровень риска
	riskLevel := calculateRiskLevel(score)
	
	// Определяем рекомендацию
	recommendation := getActionRecommendation(score)

	return &models.RiskAnalysis{
		RiskScore:     score,
		RiskLevel:     riskLevel,
		Flags:         flags,
		Recommendation: recommendation,
		AnalyzedAt:    time.Now(),
	}, nil
}

// calculateRiskLevel определяет уровень риска на основе баллов
func calculateRiskLevel(score int) string {
	if score <= 30 {
		return "low"
	} else if score <= 70 {
		return "medium"
	}
	return "high"
}

// getActionRecommendation возвращает рекомендацию по действию
func getActionRecommendation(score int) string {
	if score <= 30 {
		return "auto_approve"
	} else if score <= 70 {
		return "log_only"
	}
	return "require_verification"
}

// isOffshoreCountry проверяет, является ли страна офшорной (можно расширить)
func isOffshoreCountry(countryCode string) bool {
	offshoreCountries := map[string]bool{
		"VG": true, // Британские Виргинские острова
		"KY": true, // Каймановы острова
		"BS": true, // Багамы
		"PA": true, // Панама
		"SC": true, // Сейшелы
		"MU": true, // Маврикий
		"CH": true, // Швейцария (для примера)
	}
	return offshoreCountries[countryCode]
}

