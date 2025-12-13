package kafka

import (
	"context"

	"bank-aml-system/internal/models"
)

// Producer определяет интерфейс для отправки сообщений в Kafka
type Producer interface {
	// SendTransactionEvent отправляет событие транзакции в Kafka
	SendTransactionEvent(event *models.KafkaTransactionEvent) error
	
	// Close закрывает соединение с Kafka
	Close() error
}

// Consumer определяет интерфейс для получения сообщений из Kafka
type Consumer interface {
	// Start запускает consumer для обработки сообщений
	Start(ctx context.Context) error
	
	// Close закрывает соединение с Kafka
	Close() error
}

