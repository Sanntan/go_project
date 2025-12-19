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

	transaction "bank-aml-system/api/proto"
	"bank-aml-system/config"
	_ "bank-aml-system/docs" // Swagger docs
	"bank-aml-system/internal/api/rest"
	"bank-aml-system/internal/grpc"

	grpcLib "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	// Настраиваем gRPC-клиент, чтобы из REST можно было вызывать gRPC-сервис
	var grpcConn *grpcLib.ClientConn
	var grpcClient transaction.TransactionServiceClient

	grpcAddress := fmt.Sprintf("localhost:%d", cfg.Server.GRPCPort)
	grpcConn, err = grpcLib.Dial(grpcAddress, grpcLib.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Warning: failed to connect to gRPC server at %s: %v", grpcAddress, err)
	} else {
		log.Printf("gRPC client connected to %s", grpcAddress)
		grpcClient = transaction.NewTransactionServiceClient(grpcConn)
		defer grpcConn.Close()
	}

	// Настройка REST API
	handlers := rest.NewHandlers(deps.TransactionService, grpcClient)
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
	if deps.RedisClient != nil && deps.RiskAnalyzer != nil {
		go func() {
			log.Printf("Starting gRPC server on port %d...", cfg.Server.GRPCPort)
			grpcServer := grpc.NewTransactionGRPCServer(deps.StorageRepo, deps.KafkaProducer, deps.RedisClient, deps.RiskAnalyzer)
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
