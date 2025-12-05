package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DB     DBConfig
	Redis  RedisConfig
	Kafka  KafkaConfig
	Server ServerConfig
}

type DBConfig struct {
	DBPath string // Путь к файлу SQLite
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

type KafkaConfig struct {
	Brokers            []string
	TransactionTopic   string
	AnalyzedTopic      string
	ConsumerGroupID    string
}

type ServerConfig struct {
	IngestionPort      int
	FraudDetectionPort int
}

func Load() *Config {
	// Загружаем .env файл, если он существует
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		DB: DBConfig{
			DBPath: getEnv("DB_PATH", "./data/bank_aml.db"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
		},
		Kafka: KafkaConfig{
			Brokers:            []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			TransactionTopic:   getEnv("KAFKA_TRANSACTION_TOPIC", "bank.transactions.received"),
			AnalyzedTopic:      getEnv("KAFKA_ANALYZED_TOPIC", "bank.transactions.analyzed"),
			ConsumerGroupID:    getEnv("KAFKA_CONSUMER_GROUP", "fraud-detection-group"),
		},
		Server: ServerConfig{
			IngestionPort:      getEnvAsInt("INGESTION_SERVICE_PORT", 8080),
			FraudDetectionPort: getEnvAsInt("FRAUD_DETECTION_SERVICE_PORT", 8081),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

