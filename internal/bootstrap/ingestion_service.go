package bootstrap

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "bank-aml-system/docs" // Swagger docs
	"bank-aml-system/internal/api/rest"
	"bank-aml-system/internal/config"
	"bank-aml-system/internal/kafka"
	"bank-aml-system/internal/services"
	"bank-aml-system/internal/storage/sqlite"
	"bank-aml-system/internal/grpc"
	"bank-aml-system/internal/redis"
	"bank-aml-system/internal/fraud"
)

// StartIngestionService запускает сервис приема транзакций
func StartIngestionService() {
	cfg := config.Load()

	// Инициализация SQLite
	storage, err := sqlite.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite: %v", err)
	}
	defer storage.Close()

	storageRepo := sqlite.NewRepository(storage)

	// Инициализация Kafka Producer
	log.Println("Connecting to Kafka...")
	producer, err := kafka.NewProducer(cfg)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()
	log.Println("Kafka producer connected successfully")

	// Инициализация Redis для gRPC сервера
	log.Println("Connecting to Redis...")
	redisClient, err := redis.NewClient(cfg)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis (gRPC will have limited functionality): %v", err)
	} else {
		log.Println("Redis connection established")
		defer redisClient.Close()
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

	// Настройка REST API
	handlers := rest.NewHandlers(transactionService)
	router := rest.SetupRouter(handlers)

	// Запуск HTTP сервера
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.IngestionPort),
		Handler: router,
	}

	go func() {
		log.Printf("Transaction Ingestion Service starting on port %d", cfg.Server.IngestionPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Запуск gRPC сервера в отдельной горутине
	if redisClient != nil && riskAnalyzer != nil {
		go func() {
			log.Printf("Starting gRPC server on port %d...", cfg.Server.GRPCPort)
			grpcServer := grpc.NewTransactionGRPCServer(storageRepo, producer, redisClient, riskAnalyzer)
			if err := grpc.StartGRPCServer(cfg, grpcServer); err != nil {
				log.Fatalf("Failed to start gRPC server: %v", err)
			}
		}()
	} else {
		log.Println("Warning: gRPC server not started (Redis not available)")
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

