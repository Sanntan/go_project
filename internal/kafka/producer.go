package kafka

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"bank-aml-system/config"
	"bank-aml-system/internal/models"
)

type ProducerImpl struct {
	producer sarama.SyncProducer
	topic    string
}

func NewProducer(cfg *config.Config) (Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5

	producer, err := sarama.NewSyncProducer(cfg.Kafka.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	log.Println("Kafka producer created successfully")
	return &ProducerImpl{
		producer: producer,
		topic:    cfg.Kafka.TransactionTopic,
	}, nil
}

func (p *ProducerImpl) SendTransactionEvent(event *models.KafkaTransactionEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic:     p.topic,
		Value:     sarama.StringEncoder(data),
		Timestamp: time.Now(),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	log.Printf("Message sent to topic %s, partition %d, offset %d", p.topic, partition, offset)
	return nil
}

func (p *ProducerImpl) Close() error {
	return p.producer.Close()
}

