package ingestion

import (
	"log"

	"bank-aml-system/config"
	"bank-aml-system/internal/fraud"
	"bank-aml-system/internal/kafka"
	"bank-aml-system/internal/redis"
	"bank-aml-system/internal/services"
	"bank-aml-system/internal/storage"
	"bank-aml-system/internal/storage/sqlite"
)

// Dependencies содержит все зависимости для ingestion service
type Dependencies struct {
	StorageConn        *sqlite.SQLiteStorage
	StorageRepo        storage.TransactionRepository
	KafkaProducer      kafka.Producer
	RedisClient        *redis.Client
	RiskAnalyzer       *fraud.RiskAnalyzer
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

	// Инициализация Redis для gRPC сервера
	log.Println("Connecting to Redis...")
	redisClient, err := redis.NewClient(cfg)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis (gRPC will have limited functionality): %v", err)
	} else {
		log.Println("Redis connection established")
		if err := redisClient.InitializeBlacklists(); err != nil {
			log.Printf("Warning: Failed to initialize blacklists: %v", err)
		} else {
			log.Println("Redis blacklists initialized")
		}
	}

	// Инициализация анализатора рисков для gRPC
	var riskAnalyzer *fraud.RiskAnalyzer
	if redisClient != nil {
		riskAnalyzer = fraud.NewRiskAnalyzer(redisClient)
	}

	// Создаем сервис транзакций
	transactionService := services.NewTransactionService(storageRepo, producer)

	return &Dependencies{
		StorageConn:        storage,
		StorageRepo:        storageRepo,
		KafkaProducer:      producer,
		RedisClient:        redisClient,
		RiskAnalyzer:       riskAnalyzer,
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
	if d.RedisClient != nil {
		if err := d.RedisClient.Close(); err != nil {
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
