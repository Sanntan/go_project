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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"bank-aml-system/internal/config"
	"bank-aml-system/internal/database"
	"bank-aml-system/internal/generator"
	"bank-aml-system/internal/kafka"
	"bank-aml-system/internal/logger"
	"bank-aml-system/internal/models"
)

type IngestionService struct {
	repo     *database.Repository
	producer *kafka.Producer
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

	// Инициализация Kafka Producer
	producer, err := kafka.NewProducer(cfg)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()

	service := &IngestionService{
		repo:     repo,
		producer: producer,
	}

	// Настройка Gin router
	router := gin.Default()
	
	// CORS middleware - применяем ко всем маршрутам
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

	// API endpoints
	api := router.Group("/api/v1")
	{
		api.POST("/transactions", service.handleTransaction)
		api.GET("/transactions", service.getAllTransactions)
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

	// Clear database endpoint
	api.DELETE("/transactions", func(c *gin.Context) {
		if err := repo.ClearAllTransactions(); err != nil {
			log.Printf("Error clearing transactions: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear transactions"})
			return
		}
		
		logger.LogEvent(logger.EventDBUpdated, "ingestion-service", "sqlite", map[string]interface{}{
			"action": "database_cleared",
		})
		
		// Очищаем localStorage на фронтенде через ответ
		c.JSON(http.StatusOK, gin.H{
			"message": "All transactions cleared successfully",
			"clear_storage": true,
		})
	})

	// Generate single random transaction for form (returns transaction data, doesn't save)
	api.GET("/transactions/generate", func(c *gin.Context) {
		gen := generator.NewTransactionGenerator()
		tx := gen.GenerateRandomTransaction()

		// Возвращаем данные для заполнения формы
		c.JSON(http.StatusOK, gin.H{
			"transaction_id":       tx.TransactionID,
			"account_number":       tx.AccountNumber,
			"amount":               tx.Amount,
			"currency":             tx.Currency,
			"transaction_type":    tx.TransactionType,
			"counterparty_account": tx.CounterpartyAccount,
			"counterparty_bank":    tx.CounterpartyBank,
			"counterparty_country": tx.CounterpartyCountry,
			"channel":              tx.Channel,
			"user_id":              tx.UserID,
			"branch_id":            tx.BranchID,
		})
	})

	// Запуск сервера
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

func (s *IngestionService) handleTransaction(c *gin.Context) {
	var req models.ProcessingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Генерируем processing_id
	processingID := "proc_" + uuid.New().String()

	// Логируем получение транзакции
	logger.LogEvent(logger.EventTransactionReceived, "ingestion-service", "api", map[string]interface{}{
		"processing_id":  processingID,
		"transaction_id": req.TransactionID,
		"amount":         req.Amount,
		"currency":       req.Currency,
	})

	// Сохраняем транзакцию в БД со статусом pending_review
	if err := s.repo.SaveTransaction(processingID, &req.Transaction); err != nil {
		log.Printf("Error saving transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transaction"})
		return
	}

	// Логируем сохранение в БД
	logger.LogEvent(logger.EventTransactionSaved, "ingestion-service", "sqlite", map[string]interface{}{
		"processing_id": processingID,
		"status":        "pending_review",
	})

	// Создаем событие для Kafka
	event := &models.KafkaTransactionEvent{
		EventID:   "evt_" + uuid.New().String(),
		EventType: "transaction_received",
		Timestamp: time.Now(),
		Data: models.KafkaTransactionData{
			ProcessingID:        processingID,
			TransactionID:       req.TransactionID,
			AccountNumber:       req.AccountNumber,
			Amount:              req.Amount,
			Currency:            req.Currency,
			TransactionType:     req.TransactionType,
			CounterpartyCountry: req.CounterpartyCountry,
			Channel:             req.Channel,
		},
	}

	// Отправляем событие в Kafka
	if err := s.producer.SendTransactionEvent(event); err != nil {
		log.Printf("Error sending event to Kafka: %v", err)
		// В продакшене можно добавить retry или сохранить в очередь
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send transaction for analysis"})
		return
	}

	// Логируем отправку в Kafka
	logger.LogEvent(logger.EventKafkaSent, "ingestion-service", "kafka", map[string]interface{}{
		"processing_id": processingID,
		"topic":         "bank.transactions.received",
		"event_id":      event.EventID,
	})

	response := models.ProcessingResponse{
		ProcessingID: processingID,
		Status:       "pending_review",
		Message:      "Transaction accepted for analysis",
	}

	c.JSON(http.StatusCreated, response)
}

func (s *IngestionService) getAllTransactions(c *gin.Context) {
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	transactions, err := s.repo.GetAllTransactions(limit)
	if err != nil {
		log.Printf("Error getting all transactions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transactions"})
		return
	}

	responses := make([]models.TransactionStatusResponse, 0, len(transactions))
	for _, tx := range transactions {
		responses = append(responses, models.TransactionStatusResponse{
			ProcessingID:      tx.ProcessingID,
			TransactionID:     tx.TransactionID,
			Amount:            tx.Amount,
			Currency:          tx.Currency,
			Status:            tx.Status,
			RiskScore:         tx.RiskScore,
			RiskLevel:         tx.RiskLevel,
			AnalysisTimestamp: tx.AnalysisTimestamp,
		})
	}

	c.JSON(http.StatusOK, gin.H{"transactions": responses})
}

func (s *IngestionService) getTransactionStatus(c *gin.Context) {
	processingID := c.Param("processing_id")

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

	c.JSON(http.StatusOK, response)
}

