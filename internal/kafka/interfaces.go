package kafka

import (
	"bank-aml-system/internal/models"
)

// Producer определяет интерфейс для отправки сообщений в Kafka
type Producer interface {
	SendTransactionEvent(event *models.KafkaTransactionEvent) error

	Close() error
}

