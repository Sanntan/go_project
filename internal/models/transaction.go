package models

import (
	"time"
)

// Transaction представляет банковскую транзакцию
type Transaction struct {
	TransactionID       string    `json:"transaction_id" binding:"required"`
	AccountNumber       string    `json:"account_number" binding:"required"`
	Amount              float64   `json:"amount" binding:"required,gt=0"`
	Currency            string    `json:"currency" binding:"required"`
	TransactionType     string    `json:"transaction_type" binding:"required"`
	CounterpartyAccount string    `json:"counterparty_account"`
	CounterpartyBank    string    `json:"counterparty_bank"`
	CounterpartyCountry string    `json:"counterparty_country"`
	Timestamp           time.Time `json:"timestamp"`
	Channel             string    `json:"channel"`
	UserID              string    `json:"user_id"`
	BranchID            string    `json:"branch_id"`
}

// ProcessingRequest представляет запрос на обработку транзакции
type ProcessingRequest struct {
	Transaction
}

// ProcessingResponse представляет ответ на запрос обработки
type ProcessingResponse struct {
	ProcessingID string `json:"processing_id"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

// TransactionStatus представляет статус транзакции в БД
type TransactionStatus struct {
	ID                int64     `db:"id"`
	ProcessingID      string    `db:"processing_id"`
	TransactionID     string    `db:"transaction_id"`
	Amount            *float64  `db:"amount"`
	Currency          *string   `db:"currency"`
	Status            string    `db:"status"`
	RiskScore         *int      `db:"risk_score"`
	RiskLevel         *string   `db:"risk_level"`
	AnalysisTimestamp *time.Time `db:"analysis_timestamp"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

// TransactionStatusResponse представляет ответ на запрос статуса
type TransactionStatusResponse struct {
	ProcessingID      string    `json:"processing_id"`
	TransactionID     string    `json:"transaction_id"`
	Amount            *float64  `json:"amount,omitempty"`
	Currency          *string   `json:"currency,omitempty"`
	Status            string    `json:"status"`
	RiskScore         *int      `json:"risk_score,omitempty"`
	RiskLevel         *string   `json:"risk_level,omitempty"`
	AnalysisTimestamp *time.Time `json:"analysis_timestamp,omitempty"`
	Flags             []string  `json:"flags,omitempty"`
}

// RiskAnalysis представляет результат анализа рисков
type RiskAnalysis struct {
	RiskScore     int      `json:"risk_score"`
	RiskLevel     string   `json:"risk_level"`
	Flags         []string `json:"flags"`
	Recommendation string  `json:"recommendation"`
	AnalyzedAt    time.Time `json:"analyzed_at"`
}

// KafkaTransactionEvent представляет событие транзакции в Kafka
type KafkaTransactionEvent struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      KafkaTransactionData   `json:"data"`
}

// KafkaTransactionData представляет данные транзакции в Kafka
type KafkaTransactionData struct {
	ProcessingID      string  `json:"processing_id"`
	TransactionID     string  `json:"transaction_id"`
	AccountNumber     string  `json:"account_number"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
	TransactionType   string  `json:"transaction_type"`
	CounterpartyCountry string `json:"counterparty_country"`
	Channel           string  `json:"channel"`
}

