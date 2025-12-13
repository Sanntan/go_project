package bootstrap

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

	"github.com/gin-gonic/gin"
	"bank-aml-system/internal/config"
	"bank-aml-system/internal/kafka"
	"bank-aml-system/internal/logger"
	"bank-aml-system/internal/models"
	"bank-aml-system/internal/redis"
	"bank-aml-system/internal/services"
	"bank-aml-system/internal/storage"
	"bank-aml-system/internal/storage/sqlite"
)

// StartFraudDetectionService запускает сервис обнаружения мошенничества
func StartFraudDetectionService() {
	cfg := config.Load()

	// Инициализация SQLite
	storageConn, err := sqlite.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite: %v", err)
	}
	defer storageConn.Close()

	storageRepo := sqlite.NewRepository(storageConn)

	// Инициализация Redis
	log.Println("Connecting to Redis...")
	redisClient, err := redis.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Redis connection established")

	if err := redisClient.InitializeBlacklists(); err != nil {
		log.Printf("Warning: Failed to initialize blacklists: %v", err)
	} else {
		log.Println("Redis blacklists initialized")
	}

	// Инициализация анализатора рисков
	riskAnalyzerService := services.NewRiskAnalyzer(redisClient)

	// Создаем сервис транзакций для получения статусов
	transactionService := services.NewTransactionService(storageRepo, nil)

	// Настройка обработчика Kafka событий
	handler := func(event *models.KafkaTransactionEvent) error {
		logger.LogEvent(logger.EventKafkaReceived, "fraud-detection-service", "kafka", map[string]interface{}{
			"processing_id": event.Data.ProcessingID,
			"event_id":      event.EventID,
			"topic":         "bank.transactions.received",
		})
		return processTransaction(event, storageRepo, redisClient, riskAnalyzerService)
	}

	// Инициализация Kafka Consumer
	log.Println("Connecting to Kafka...")
	consumer, err := kafka.NewConsumer(cfg, handler)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}
	defer consumer.Close()
	log.Println("Kafka consumer connected successfully")

	// Запуск Kafka consumer в отдельной горутине
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Println("Starting Kafka consumer...")
		if err := consumer.Start(ctx); err != nil {
			log.Fatalf("Kafka consumer error: %v", err)
		}
	}()

	// Настройка REST API
	router := gin.Default()
	
	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
	
	router.Use(gin.Logger(), gin.Recovery())

	api := router.Group("/api/v1")
	{
		api.GET("/transactions/:processing_id", func(c *gin.Context) {
			processingID := c.Param("processing_id")
			status, err := transactionService.GetTransactionStatus(processingID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transaction status"})
				return
			}
			if status == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
				return
			}
			c.JSON(http.StatusOK, status)
		})
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

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

	router.GET("/api/v1/stats", func(c *gin.Context) {
		stats := logger.GetStats()
		c.JSON(http.StatusOK, stats)
	})

	api.DELETE("/transactions", func(c *gin.Context) {
		if err := storageRepo.ClearAllTransactions(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear transactions"})
			return
		}
		
		if err := redisClient.ClearTransactionData(); err != nil {
			log.Printf("Warning: Failed to clear Redis data: %v", err)
		}
		
		logger.LogEvent(logger.EventDBUpdated, "fraud-detection-service", "sqlite", map[string]interface{}{
			"action": "database_cleared",
		})
		
		c.JSON(http.StatusOK, gin.H{
			"message":       "All transactions and cache cleared successfully",
			"clear_storage": true,
		})
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

func processTransaction(
	event *models.KafkaTransactionEvent,
	repo storage.TransactionRepository,
	redisClient *redis.Client,
	riskAnalyzer services.RiskAnalyzer,
) error {
	log.Printf("Processing transaction: %s", event.Data.ProcessingID)

	logger.LogEvent(logger.EventAnalysisStarted, "fraud-detection-service", "analyzer", map[string]interface{}{
		"processing_id": event.Data.ProcessingID,
	})

	tx, err := repo.GetFullTransactionByProcessingID(event.Data.ProcessingID)
	if err != nil {
		return err
	}
	if tx == nil {
		log.Printf("Transaction not found: %s", event.Data.ProcessingID)
		return nil
	}

	analysis, err := riskAnalyzer.AnalyzeTransaction(tx)
	if err != nil {
		log.Printf("Error analyzing transaction: %v", err)
		return err
	}

	logger.LogEvent(logger.EventAnalysisCompleted, "fraud-detection-service", "analyzer", map[string]interface{}{
		"processing_id": event.Data.ProcessingID,
		"risk_score":    analysis.RiskScore,
		"risk_level":     analysis.RiskLevel,
		"flags":          analysis.Flags,
	})

	if err := redisClient.SaveAnalysis(event.Data.ProcessingID, analysis); err != nil {
		log.Printf("Error saving analysis to Redis: %v", err)
	} else {
		logger.LogEvent(logger.EventRedisSaved, "fraud-detection-service", "redis", map[string]interface{}{
			"processing_id": event.Data.ProcessingID,
			"risk_score":    analysis.RiskScore,
			"risk_level":    analysis.RiskLevel,
		})
	}

	if err := repo.UpdateTransactionAnalysis(
		event.Data.ProcessingID,
		analysis.RiskScore,
		analysis.RiskLevel,
		analysis.AnalyzedAt,
	); err != nil {
		log.Printf("Error updating transaction in DB: %v", err)
		return err
	}

	logger.LogEvent(logger.EventDBUpdated, "fraud-detection-service", "sqlite", map[string]interface{}{
		"processing_id": event.Data.ProcessingID,
		"status":        "reviewed",
		"risk_score":    analysis.RiskScore,
		"risk_level":    analysis.RiskLevel,
	})

	if err := redisClient.IncrementRiskStats(analysis.RiskLevel); err != nil {
		log.Printf("Error updating risk stats: %v", err)
	}

	log.Printf("Transaction %s analyzed: risk_score=%d, risk_level=%s",
		event.Data.ProcessingID, analysis.RiskScore, analysis.RiskLevel)

	return nil
}

