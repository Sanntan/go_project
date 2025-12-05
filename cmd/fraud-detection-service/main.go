package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"bank-aml-system/internal/config"
	"bank-aml-system/internal/database"
	"bank-aml-system/internal/fraud"
	"bank-aml-system/internal/kafka"
	"bank-aml-system/internal/logger"
	"bank-aml-system/internal/models"
	"bank-aml-system/internal/redis"

	"github.com/gin-gonic/gin"
)

type FraudDetectionService struct {
	repo         *database.Repository
	redisClient  *redis.Client
	riskAnalyzer *fraud.RiskAnalyzer
}

func main() {
	cfg := config.Load()

	// Инициализация SQLite
	db, err := database.NewSQLiteDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)

	// Инициализация Redis
	redisClient, err := redis.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Инициализация черных списков в Redis
	if err := redisClient.InitializeBlacklists(); err != nil {
		log.Printf("Warning: Failed to initialize blacklists: %v", err)
	}

	// Инициализация анализатора рисков
	riskAnalyzer := fraud.NewRiskAnalyzer(redisClient)

	service := &FraudDetectionService{
		repo:         repo,
		redisClient:  redisClient,
		riskAnalyzer: riskAnalyzer,
	}

	// Настройка обработчика Kafka событий
	handler := func(event *models.KafkaTransactionEvent) error {
		// Логируем получение из Kafka
		logger.LogEvent(logger.EventKafkaReceived, "fraud-detection-service", "kafka", map[string]interface{}{
			"processing_id": event.Data.ProcessingID,
			"event_id":      event.EventID,
			"topic":         "bank.transactions.received",
		})
		return service.processTransaction(event)
	}

	// Инициализация Kafka Consumer
	consumer, err := kafka.NewConsumer(cfg, handler)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}
	defer consumer.Close()

	// Запуск Kafka consumer в отдельной горутине
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Println("Starting Kafka consumer...")
		if err := consumer.Start(ctx); err != nil {
			log.Fatalf("Kafka consumer error: %v", err)
		}
	}()

	// Настройка REST API для проверки статуса (опционально)
	router := gin.Default()
	
	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
	
	router.Use(gin.Logger(), gin.Recovery())

	api := router.Group("/api/v1")
	{
		api.GET("/transactions/:processing_id", service.getTransactionStatus)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Events endpoint
	router.GET("/api/v1/events", func(c *gin.Context) {
		limit := 100
		if limitStr := c.Query("limit"); limitStr != "" {
			if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 500 {
				limit = parsed
			}
		}
		events := logger.GetEvents(limit)
		c.JSON(http.StatusOK, gin.H{"events": events})
	})

	// Stats endpoint
	router.GET("/api/v1/stats", func(c *gin.Context) {
		stats := logger.GetStats()
		c.JSON(http.StatusOK, stats)
	})

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

func (s *FraudDetectionService) processTransaction(event *models.KafkaTransactionEvent) error {
	log.Printf("Processing transaction: %s", event.Data.ProcessingID)

	// Логируем начало анализа
	logger.LogEvent(logger.EventAnalysisStarted, "fraud-detection-service", "analyzer", map[string]interface{}{
		"processing_id": event.Data.ProcessingID,
	})

	// Получаем полную транзакцию из БД для анализа
	tx, err := s.repo.GetFullTransactionByProcessingID(event.Data.ProcessingID)
	if err != nil {
		return err
	}
	if tx == nil {
		log.Printf("Transaction not found: %s", event.Data.ProcessingID)
		return nil
	}

	// Выполняем анализ рисков
	analysis, err := s.riskAnalyzer.AnalyzeTransaction(tx)
	if err != nil {
		log.Printf("Error analyzing transaction: %v", err)
		return err
	}

	// Логируем завершение анализа
	logger.LogEvent(logger.EventAnalysisCompleted, "fraud-detection-service", "analyzer", map[string]interface{}{
		"processing_id": event.Data.ProcessingID,
		"risk_score":    analysis.RiskScore,
		"risk_level":    analysis.RiskLevel,
		"flags":         analysis.Flags,
	})

	// Сохраняем результаты анализа в Redis
	if err := s.redisClient.SaveAnalysis(event.Data.ProcessingID, analysis); err != nil {
		log.Printf("Error saving analysis to Redis: %v", err)
	} else {
		// Логируем сохранение в Redis
		logger.LogEvent(logger.EventRedisSaved, "fraud-detection-service", "redis", map[string]interface{}{
			"processing_id": event.Data.ProcessingID,
			"risk_score":    analysis.RiskScore,
			"risk_level":    analysis.RiskLevel,
		})
	}

	// Обновляем статус в SQLite
	if err := s.repo.UpdateTransactionAnalysis(
		event.Data.ProcessingID,
		analysis.RiskScore,
		analysis.RiskLevel,
		analysis.AnalyzedAt,
	); err != nil {
		log.Printf("Error updating transaction in DB: %v", err)
		return err
	}

	// Логируем обновление в БД
	logger.LogEvent(logger.EventDBUpdated, "fraud-detection-service", "sqlite", map[string]interface{}{
		"processing_id": event.Data.ProcessingID,
		"status":        "reviewed",
		"risk_score":    analysis.RiskScore,
		"risk_level":    analysis.RiskLevel,
	})

	// Обновляем статистику рисков
	if err := s.redisClient.IncrementRiskStats(analysis.RiskLevel); err != nil {
		log.Printf("Error updating risk stats: %v", err)
	}

	log.Printf("Transaction %s analyzed: risk_score=%d, risk_level=%s",
		event.Data.ProcessingID, analysis.RiskScore, analysis.RiskLevel)

	return nil
}

func (s *FraudDetectionService) getTransactionStatus(c *gin.Context) {
	processingID := c.Param("processing_id")

	// Сначала проверяем Redis (быстрый доступ)
	analysis, err := s.redisClient.GetAnalysis(processingID)
	if err != nil {
		log.Printf("Error getting analysis from Redis: %v", err)
	}

	// Получаем статус из БД
	status, err := s.repo.GetTransactionByProcessingID(processingID)
	if err != nil {
		log.Printf("Error getting transaction status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transaction status"})
		return
	}

	if status == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	response := models.TransactionStatusResponse{
		ProcessingID:      status.ProcessingID,
		TransactionID:     status.TransactionID,
		Amount:            status.Amount,
		Currency:          status.Currency,
		Status:            status.Status,
		RiskScore:         status.RiskScore,
		RiskLevel:         status.RiskLevel,
		AnalysisTimestamp: status.AnalysisTimestamp,
	}

	// Добавляем флаги из анализа, если есть
	if analysis != nil {
		response.Flags = analysis.Flags
	}

	c.JSON(http.StatusOK, response)
}
