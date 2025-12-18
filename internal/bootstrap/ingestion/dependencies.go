package ingestion

import (
	"log"

	"bank-aml-system/config"
	"bank-aml-system/internal/kafka"
	"bank-aml-system/internal/services"
	"bank-aml-system/internal/storage"
	"bank-aml-system/internal/storage/sqlite"
)

// Dependencies содержит все зависимости для ingestion service
type Dependencies struct {
	StorageConn        *sqlite.SQLiteStorage
	StorageRepo        storage.TransactionRepository
	KafkaProducer      kafka.Producer
	TransactionService services.TransactionService
}

// InitializeDependencies инициализирует все зависимости для ingestion service
func InitializeDependencies(cfg *config.Config) (*Dependencies, error) {
	// Инициализация SQLite
	storage, err := sqlite.NewConnection(cfg)
	if err != nil {
		return nil, err
	}

	storageRepo := sqlite.NewRepository(storage)

	// Инициализация Kafka Producer
	log.Println("Connecting to Kafka...")
	producer, err := kafka.NewProducer(cfg)
	if err != nil {
		return nil, err
	}
	log.Println("Kafka producer connected successfully")

	// Создаем сервис транзакций
	transactionService := services.NewTransactionService(storageRepo, producer)

	return &Dependencies{
		StorageConn:        storage,
		StorageRepo:        storageRepo,
		KafkaProducer:      producer,
		TransactionService: transactionService,
	}, nil
}

// Close закрывает все соединения
func (d *Dependencies) Close() error {
	if d.KafkaProducer != nil {
		if err := d.KafkaProducer.Close(); err != nil {
			return err
		}
	}
	if d.StorageConn != nil {
		if err := d.StorageConn.Close(); err != nil {
			return err
		}
	}
	return nil
}
