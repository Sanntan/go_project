package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/IBM/sarama"
	"bank-aml-system/internal/config"
	"bank-aml-system/internal/models"
)

type ConsumerImpl struct {
	consumer sarama.ConsumerGroup
	topic    string
	handler  func(*models.KafkaTransactionEvent) error
}

func NewConsumer(cfg *config.Config, handler func(*models.KafkaTransactionEvent) error) (Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Version = sarama.V2_8_0_0

	consumer, err := sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.Kafka.ConsumerGroupID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	log.Println("Kafka consumer created successfully")
	return &ConsumerImpl{
		consumer: consumer,
		topic:    cfg.Kafka.TransactionTopic,
		handler:  handler,
	}, nil
}

func (c *ConsumerImpl) Start(ctx context.Context) error {
	topics := []string{c.topic}
	
	consumerHandler := &consumerGroupHandler{
		handler: c.handler,
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			if err := c.consumer.Consume(ctx, topics, consumerHandler); err != nil {
				log.Printf("Error from consumer: %v", err)
				return
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case err := <-c.consumer.Errors():
				if err != nil {
					log.Printf("Consumer error: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
	log.Println("Consumer context cancelled, shutting down...")
	wg.Wait()
	return c.consumer.Close()
}

func (c *ConsumerImpl) Close() error {
	return c.consumer.Close()
}

type consumerGroupHandler struct {
	handler func(*models.KafkaTransactionEvent) error
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			var event models.KafkaTransactionEvent
			if err := json.Unmarshal(message.Value, &event); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				session.MarkMessage(message, "")
				continue
			}

			if err := h.handler(&event); err != nil {
				log.Printf("Error handling message: %v", err)
				// В продакшене можно добавить retry логику или dead letter queue
			}

			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

