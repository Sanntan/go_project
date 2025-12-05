package generator

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"bank-aml-system/internal/models"
)

type TransactionGenerator struct {
	rand *rand.Rand
}

func NewTransactionGenerator() *TransactionGenerator {
	return &TransactionGenerator{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateTransaction генерирует транзакцию с заданным уровнем риска
func (g *TransactionGenerator) GenerateTransaction(riskLevel string) *models.Transaction {
	// Генерируем уникальный ID на основе времени и случайного числа
	baseID := time.Now().UnixNano() + g.rand.Int63n(1000000000)
	
	// Генерируем уникальный номер счета
	accountSuffix := 1000000000 + g.rand.Int63n(8999999999)
	
	// Добавляем небольшую случайную задержку к времени для уникальности
	timeOffset := time.Duration(g.rand.Intn(1000)) * time.Millisecond
	
	tx := &models.Transaction{
		TransactionID: fmt.Sprintf("TXN-AUTO-%d", baseID),
		AccountNumber:  fmt.Sprintf("ACC%d", accountSuffix),
		Currency:      "RUB",
		TransactionType: "transfer",
		Channel:       "online",
		UserID:        fmt.Sprintf("user%d", g.rand.Intn(100000)),
		BranchID:      fmt.Sprintf("branch%d", g.rand.Intn(1000)),
		Timestamp:     time.Now().Add(timeOffset),
	}

	switch riskLevel {
	case "low":
		g.generateLowRisk(tx)
	case "medium":
		g.generateMediumRisk(tx)
	case "high":
		g.generateHighRisk(tx)
	default:
		g.generateLowRisk(tx)
	}

	return tx
}

// generateLowRisk генерирует транзакцию с низким риском (0-30 баллов)
func (g *TransactionGenerator) generateLowRisk(tx *models.Transaction) {
	// Небольшая сумма (до 500k RUB) - 0 баллов
	tx.Amount = g.roundToTwoDecimals(10000.0 + g.rand.Float64()*490000.0)
	
	// Обычная страна (не офшор) - 0 баллов
	tx.CounterpartyCountry = g.getRandomSafeCountry()
	
	// Обычное время (8:00 - 22:00) - 0 баллов
	hour := 8 + g.rand.Intn(14)
	tx.Timestamp = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), hour, g.rand.Intn(60), 0, 0, time.Local)
	
	// Обычный счет
	tx.CounterpartyAccount = fmt.Sprintf("ACC%d", 2000000000+g.rand.Int63n(9999999999))
	tx.CounterpartyBank = g.getRandomBank()
}

// generateMediumRisk генерирует транзакцию со средним риском (31-70 баллов)
func (g *TransactionGenerator) generateMediumRisk(tx *models.Transaction) {
	// Вариант 1: Средняя сумма + офшор (0 + 40 = 40 баллов) - medium
	// Вариант 2: Крупная сумма + ночное время (30 + 15 = 45 баллов) - medium
	// Вариант 3: Средняя сумма + офшор + ночное время (0 + 40 + 15 = 55 баллов) - medium
	
	variant := g.rand.Intn(3)
	
	switch variant {
	case 0:
		// Средняя сумма (до 1M) + офшор = 40 баллов
		tx.Amount = g.roundToTwoDecimals(500000.0 + g.rand.Float64()*500000.0)
		tx.CounterpartyCountry = g.getRandomOffshoreCountry()
		tx.CounterpartyAccount = fmt.Sprintf("ACC%d", 3000000000+g.rand.Int63n(9999999999))
		tx.CounterpartyBank = g.getRandomOffshoreBank()
		// Обычное время
		hour := 8 + g.rand.Intn(14)
		tx.Timestamp = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), hour, g.rand.Intn(60), 0, 0, time.Local)
	case 1:
		// Крупная сумма + ночное время = 30 + 15 = 45 баллов
		tx.Amount = g.roundToTwoDecimals(1000000.0 + g.rand.Float64()*500000.0)
		tx.CounterpartyCountry = g.getRandomSafeCountry()
		tx.CounterpartyAccount = fmt.Sprintf("ACC%d", 4000000000+g.rand.Int63n(9999999999))
		tx.CounterpartyBank = g.getRandomBank()
		// Ночное время
		hour := g.rand.Intn(6)
		tx.Timestamp = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), hour, g.rand.Intn(60), 0, 0, time.Local)
	case 2:
		// Средняя сумма + офшор + ночное время = 0 + 40 + 15 = 55 баллов
		tx.Amount = g.roundToTwoDecimals(200000.0 + g.rand.Float64()*800000.0)
		tx.CounterpartyCountry = g.getRandomOffshoreCountry()
		tx.CounterpartyAccount = fmt.Sprintf("ACC%d", 5000000000+g.rand.Int63n(9999999999))
		tx.CounterpartyBank = g.getRandomOffshoreBank()
		// Ночное время (00:00 - 06:00)
		hour := g.rand.Intn(6)
		tx.Timestamp = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), hour, g.rand.Intn(60), 0, 0, time.Local)
	}
}

// generateHighRisk генерирует транзакцию с высоким риском (71+ баллов)
func (g *TransactionGenerator) generateHighRisk(tx *models.Transaction) {
	// Вариант 1: Крупная сумма + офшор + ночное время (30 + 40 + 15 = 85 баллов) - high
	// Вариант 2: Очень крупная сумма + офшор (30 + 40 = 70, но добавим ночное время = 85) - high
	// Вариант 3: Крупная сумма + офшор + ночное время (30 + 40 + 15 = 85) - high
	
	variant := g.rand.Intn(3)
	
	switch variant {
	case 0:
		// Крупная сумма + офшор + ночное время = 30 + 40 + 15 = 85 баллов
		tx.Amount = g.roundToTwoDecimals(1000000.0 + g.rand.Float64()*2000000.0)
		tx.CounterpartyCountry = g.getRandomOffshoreCountry()
		tx.CounterpartyAccount = fmt.Sprintf("ACC%d", 6000000000+g.rand.Int63n(9999999999))
		tx.CounterpartyBank = g.getRandomOffshoreBank()
		// Ночное время
		hour := g.rand.Intn(6)
		tx.Timestamp = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), hour, g.rand.Intn(60), 0, 0, time.Local)
	case 1:
		// Очень крупная сумма + офшор + ночное время = 30 + 40 + 15 = 85 баллов
		tx.Amount = g.roundToTwoDecimals(3000000.0 + g.rand.Float64()*5000000.0)
		tx.CounterpartyCountry = g.getRandomOffshoreCountry()
		tx.CounterpartyAccount = fmt.Sprintf("ACC%d", 7000000000+g.rand.Int63n(9999999999))
		tx.CounterpartyBank = g.getRandomOffshoreBank()
		// Ночное время
		hour := g.rand.Intn(6)
		tx.Timestamp = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), hour, g.rand.Intn(60), 0, 0, time.Local)
	case 2:
		// Крупная сумма + офшор + ночное время + международный перевод = 30 + 40 + 15 = 85 баллов
		tx.Amount = g.roundToTwoDecimals(2000000.0 + g.rand.Float64()*3000000.0)
		tx.CounterpartyCountry = g.getRandomOffshoreCountry()
		tx.TransactionType = "international_transfer"
		tx.CounterpartyAccount = fmt.Sprintf("ACC%d", 8000000000+g.rand.Int63n(9999999999))
		tx.CounterpartyBank = g.getRandomOffshoreBank()
		// Ночное время
		hour := g.rand.Intn(6)
		tx.Timestamp = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), hour, g.rand.Intn(60), 0, 0, time.Local)
	}
}

func (g *TransactionGenerator) getRandomSafeCountry() string {
	safeCountries := []string{"US", "GB", "RU", "DE", "FR", "IT", "ES", "NL", "BE", "PL"}
	return safeCountries[g.rand.Intn(len(safeCountries))]
}

func (g *TransactionGenerator) getRandomOffshoreCountry() string {
	offshoreCountries := []string{"VG", "KY", "CH", "BS", "PA", "SC", "MU"}
	return offshoreCountries[g.rand.Intn(len(offshoreCountries))]
}

func (g *TransactionGenerator) getRandomBank() string {
	banks := []string{"Sberbank", "VTB", "Alfa Bank", "Gazprombank", "Raiffeisen", "Tinkoff"}
	return banks[g.rand.Intn(len(banks))]
}

func (g *TransactionGenerator) getRandomOffshoreBank() string {
	banks := []string{"UBS", "Credit Suisse", "HSBC Offshore", "Cayman National Bank", "BVI Bank", "Swiss Private Bank"}
	return banks[g.rand.Intn(len(banks))]
}

// roundToTwoDecimals округляет число до 2 знаков после запятой
func (g *TransactionGenerator) roundToTwoDecimals(value float64) float64 {
	return math.Round(value*100) / 100
}

