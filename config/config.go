package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DB     DBConfig
	Kafka  KafkaConfig
	Server ServerConfig
}

type DBConfig struct {
	DBPath string // Путь к файлу SQLite
}

type KafkaConfig struct {
	Brokers          []string
	TransactionTopic string
}

type ServerConfig struct {
	IngestionPort int
	GRPCPort      int
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
		Kafka: KafkaConfig{
			Brokers:          []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			TransactionTopic: getEnv("KAFKA_TRANSACTION_TOPIC", "bank.transactions.received"),
		},
		Server: ServerConfig{
			IngestionPort: getEnvAsInt("INGESTION_SERVICE_PORT", 8080),
			GRPCPort:      getEnvAsInt("GRPC_PORT", 50051),
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

