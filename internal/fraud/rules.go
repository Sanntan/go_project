package fraud

import (
	"time"

	"bank-aml-system/internal/models"
	"bank-aml-system/internal/redis"
)

const (
	LargeAmountThreshold     = 1000000.0 // 1 млн руб
	VeryLargeAmountThreshold = 5000000.0 // 5 млн руб
	MediumAmountThreshold    = 500000.0  // 500 тыс руб
	HighFrequencyThreshold   = 10        // 10 транзакций в день
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

	// 1. Проверка суммы транзакции (несколько уровней)
	if tx.Amount >= VeryLargeAmountThreshold {
		// Очень крупная сумма (>= 5 млн)
		score += 50
		flags = append(flags, "very_large_amount")
	} else if tx.Amount >= LargeAmountThreshold {
		// Крупная сумма (>= 1 млн, < 5 млн)
		score += 30
		flags = append(flags, "large_amount")
	} else if tx.Amount >= MediumAmountThreshold {
		// Средняя сумма (>= 500 тыс, < 1 млн)
		score += 10
		flags = append(flags, "medium_amount")
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

	// 4. Проверка необычного времени
	hour := tx.Timestamp.Hour()
	if hour >= 0 && hour < 6 {
		// Ночное время (00:00 - 06:00)
		score += 15
		flags = append(flags, "unusual_time")
	} else if hour >= 22 || hour < 8 {
		// Поздний вечер/раннее утро (22:00 - 08:00)
		score += 8
		flags = append(flags, "late_hours")
	}

	// 5. Проверка частоты операций
	dailyCount, err := r.redisClient.GetAccountDailyCount(tx.AccountNumber)
	if err != nil {
		return nil, err
	}
	if dailyCount >= HighFrequencyThreshold {
		score += 25
		flags = append(flags, "high_frequency")
	} else if dailyCount >= 5 {
		// Средняя частота (5-9 транзакций в день)
		score += 10
		flags = append(flags, "medium_frequency")
	}

	// 6. Проверка типа транзакции
	if tx.TransactionType == "international_transfer" {
		score += 20
		flags = append(flags, "international_transfer")
	} else if tx.TransactionType == "withdrawal" {
		score += 5
		flags = append(flags, "withdrawal")
	}

	// 7. Проверка канала транзакции
	if tx.Channel == "atm" {
		// Банкоматы могут быть более рискованными для крупных сумм
		if tx.Amount >= MediumAmountThreshold {
			score += 12
			flags = append(flags, "large_atm_transaction")
		} else {
			score += 5
			flags = append(flags, "atm_transaction")
		}
	} else if tx.Channel == "mobile" && tx.Amount >= LargeAmountThreshold {
		// Крупные транзакции через мобильное приложение
		score += 8
		flags = append(flags, "large_mobile_transaction")
	}

	// 8. Проверка валюты (некоторые валюты могут быть более рискованными)
	highRiskCurrencies := map[string]int{
		"CHF": 8, // Швейцарский франк
		"JPY": 5, // Японская йена
	}
	if points, exists := highRiskCurrencies[tx.Currency]; exists {
		score += points
		flags = append(flags, "high_risk_currency")
	}

	// 9. Проверка на круглые суммы (может указывать на подозрительную активность)
	if tx.Amount > 0 {
		// Проверяем, является ли сумма "круглой" (кратной 10000, 100000, 1000000)
		amount := tx.Amount
		// Используем остаток от деления с учетом float64
		if (amount >= 10000 && amount < 100000 && int64(amount)%10000 == 0) ||
			(amount >= 100000 && amount < 1000000 && int64(amount)%100000 == 0) ||
			(amount >= 1000000 && int64(amount)%1000000 == 0) {
			score += 5
			flags = append(flags, "round_amount")
		}
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
		RiskScore:      score,
		RiskLevel:      riskLevel,
		Flags:          flags,
		Recommendation: recommendation,
		AnalyzedAt:     time.Now(),
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
