package ingestion

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bank-aml-system/config"
	_ "bank-aml-system/docs" // Swagger docs
	"bank-aml-system/internal/api/rest"
	"bank-aml-system/internal/grpc"
)

// StartIngestionService запускает сервис приема транзакций
func StartIngestionService() {
	cfg := config.Load()

	// Инициализация зависимостей
	deps, err := InitializeDependencies(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize dependencies: %v", err)
	}
	defer deps.Close()

	// Настройка REST API
	handlers := rest.NewHandlers(deps.TransactionService)
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
	go func() {
		log.Printf("Starting gRPC server on port %d...", cfg.Server.GRPCPort)
		grpcServer := grpc.NewTransactionGRPCServer(deps.StorageRepo, deps.KafkaProducer)
		if err := grpc.StartGRPCServer(cfg, grpcServer); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

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
