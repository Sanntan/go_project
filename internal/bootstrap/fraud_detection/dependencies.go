package fraud_detection

import (
	"log"

	"bank-aml-system/config"
	"bank-aml-system/internal/kafka"
	"bank-aml-system/internal/models"
	"bank-aml-system/internal/redis"
	"bank-aml-system/internal/services"
	"bank-aml-system/internal/storage"
	"bank-aml-system/internal/storage/sqlite"
)

// Dependencies содержит все зависимости для fraud detection service
type Dependencies struct {
	StorageConn        *sqlite.SQLiteStorage
	StorageRepo        storage.TransactionRepository
	RedisClient        *redis.Client
	RiskAnalyzer       services.RiskAnalyzer
	TransactionService services.TransactionService
	KafkaConsumer      kafka.Consumer
}

// InitializeDependencies инициализирует все зависимости для fraud detection service
func InitializeDependencies(cfg *config.Config) (*Dependencies, error) {
	// Инициализация SQLite
	storageConn, err := sqlite.NewConnection(cfg)
	if err != nil {
		return nil, err
	}

	storageRepo := sqlite.NewRepository(storageConn)

	// Инициализация Redis
	log.Println("Connecting to Redis...")
	redisClient, err := redis.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	log.Println("Redis connection established")

	if err := redisClient.InitializeBlacklists(); err != nil {
		log.Printf("Warning: Failed to initialize blacklists: %v", err)
	} else {
		log.Println("Redis blacklists initialized")
	}

	// Инициализация анализатора рисков
	riskAnalyzerService := services.NewRiskAnalyzer(redisClient)

	// Создаем сервис транзакций для получения статусов с поддержкой Redis (для флагов)
	transactionService := services.NewTransactionServiceWithRedis(storageRepo, nil, redisClient)

	// Настройка обработчика Kafka событий
	handler := func(event *models.KafkaTransactionEvent) error {
		return processTransaction(event, storageRepo, redisClient, riskAnalyzerService)
	}

	// Инициализация Kafka Consumer
	log.Println("Connecting to Kafka...")
	consumer, err := kafka.NewConsumer(cfg, handler)
	if err != nil {
		return nil, err
	}
	log.Println("Kafka consumer connected successfully")

	return &Dependencies{
		StorageConn:        storageConn,
		StorageRepo:        storageRepo,
		RedisClient:        redisClient,
		RiskAnalyzer:       riskAnalyzerService,
		TransactionService: transactionService,
		KafkaConsumer:      consumer,
	}, nil
}

// Close закрывает все соединения
func (d *Dependencies) Close() error {
	if d.KafkaConsumer != nil {
		if err := d.KafkaConsumer.Close(); err != nil {
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
