package kafka

import (
	"context"

	"bank-aml-system/internal/models"
)

// Producer определяет интерфейс для отправки сообщений в Kafka
type Producer interface {
	SendTransactionEvent(event *models.KafkaTransactionEvent) error

	Close() error
}

// Consumer определяет интерфейс для получения сообщений из Kafka
type Consumer interface {
	Start(ctx context.Context) error

	Close() error
}
