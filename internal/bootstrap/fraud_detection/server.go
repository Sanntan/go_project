package fraud_detection

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
	"bank-aml-system/internal/api/rest"

	"github.com/gin-gonic/gin"
)

// StartFraudDetectionService запускает сервис обнаружения мошенничества
func StartFraudDetectionService() {
	cfg := config.Load()

	// Инициализация зависимостей
	deps, err := InitializeDependencies(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize dependencies: %v", err)
	}
	defer deps.Close()

	// Запуск Kafka consumer в отдельной горутине
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Println("Starting Kafka consumer...")
		if err := deps.KafkaConsumer.Start(ctx); err != nil {
			log.Fatalf("Kafka consumer error: %v", err)
		}
	}()

	// Настройка REST API
	router := gin.Default()

	// Используем общий CORS middleware
	router.Use(rest.CORSMiddleware())
	router.Use(gin.Logger(), gin.Recovery())

	// Настройка маршрутов
	SetupRoutes(router, deps.TransactionService, deps.StorageRepo, deps.RedisClient)

	// Запуск сервера
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.FraudDetectionPort),
		Handler: router,
	}

	go func() {
		log.Printf("Fraud Detection Service starting on port %d", cfg.Server.FraudDetectionPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down services...")
	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Services exited")
}
