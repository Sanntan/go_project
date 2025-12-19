package rest

import (
	"context"
	"net/http"
	"strconv"
	"time"

	transaction "bank-aml-system/api/proto"
	"bank-aml-system/internal/generator"
	"bank-aml-system/internal/logger"
	"bank-aml-system/internal/models"
	"bank-aml-system/internal/services"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	transactionService services.TransactionService
	generator          *generator.TransactionGenerator
	grpcClient         transaction.TransactionServiceClient
}

// Создает новые обработчики REST API
func NewHandlers(transactionService services.TransactionService, grpcClient transaction.TransactionServiceClient) *Handlers {
	return &Handlers{
		transactionService: transactionService,
		generator:          generator.NewTransactionGenerator(),
		grpcClient:         grpcClient,
	}
}

// HandleTransaction обрабатывает POST запрос на создание транзакции (через REST)
// @Summary Отправить транзакцию на анализ (REST)
// @Description Принимает транзакцию и отправляет её на анализ рисков через REST API. Транзакция сохраняется в БД и отправляется в Kafka для асинхронной обработки fraud-сервисом.
// @Tags transactions
// @Accept json
// @Produce json
// @Param transaction body models.ProcessingRequest true "Данные транзакции"
// @Success 201 {object} models.ProcessingResponse "Транзакция принята на обработку"
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /transactions [post]
func (h *Handlers) HandleTransaction(c *gin.Context) {
	var req models.ProcessingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Логируем получение транзакции
	logger.LogEvent(logger.EventTransactionReceived, "ingestion-service", "api", map[string]interface{}{
		"transaction_id": req.TransactionID,
		"amount":         req.Amount,
		"currency":       req.Currency,
	})

	response, err := h.transactionService.ProcessTransaction(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process transaction"})
		return
	}

	// Логируем сохранение в БД
	logger.LogEvent(logger.EventTransactionSaved, "ingestion-service", "sqlite", map[string]interface{}{
		"processing_id": response.ProcessingID,
		"status":        response.Status,
	})

	c.JSON(http.StatusCreated, response)
}

// HandleTransactionGRPC обрабатывает POST запрос и проксирует его в gRPC-сервис
// Это позволяет с фронтенда "отправить через gRPC", оставаясь при этом в HTTP.
// @Summary Отправить транзакцию через gRPC
// @Description Принимает транзакцию и отправляет её на анализ через gRPC-сервис. Транзакция автоматически сохраняется в БД, отправляется в Kafka, обрабатывается fraud-сервисом и возвращается с результатами анализа рисков (risk_score, risk_level, flags).
// @Tags transactions
// @Accept json
// @Produce json
// @Param transaction body models.ProcessingRequest true "Данные транзакции"
// @Success 201 {object} map[string]interface{} "Транзакция успешно обработана через gRPC"
// @Failure 400 {object} map[string]string "Bad Request - неверный формат данных"
// @Failure 500 {object} map[string]string "Internal Server Error - ошибка обработки"
// @Failure 503 {object} map[string]string "Service Unavailable - gRPC клиент недоступен"
// @Router /transactions/grpc [post]
func (h *Handlers) HandleTransactionGRPC(c *gin.Context) {
	if h.grpcClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gRPC client is not available"})
		return
	}

	var req models.ProcessingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Гарантируем, что у нас есть timestamp
	timestamp := req.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	// Логируем получение транзакции
	logger.LogEvent(logger.EventTransactionReceived, "ingestion-service", "api", map[string]interface{}{
		"transaction_id": req.TransactionID,
		"amount":         req.Amount,
		"currency":       req.Currency,
		"via":            "grpc",
	})

	grpcReq := &transaction.AnalyzeTransactionRequest{
		TransactionId:       req.TransactionID,
		AccountNumber:       req.AccountNumber,
		Amount:              req.Amount,
		Currency:            req.Currency,
		TransactionType:     req.TransactionType,
		CounterpartyAccount: req.CounterpartyAccount,
		CounterpartyBank:    req.CounterpartyBank,
		CounterpartyCountry: req.CounterpartyCountry,
		Channel:             req.Channel,
		UserId:              req.UserID,
		BranchId:            req.BranchID,
		Timestamp:           timestamp.Format(time.RFC3339),
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := h.grpcClient.AnalyzeTransaction(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process transaction via gRPC"})
		return
	}

	// Логируем сохранение/обработку
	logger.LogEvent(logger.EventTransactionSaved, "ingestion-service", "sqlite", map[string]interface{}{
		"processing_id": resp.ProcessingId,
		"status":        resp.Status,
		"via":           "grpc",
	})

	c.JSON(http.StatusCreated, gin.H{
		"processing_id": resp.ProcessingId,
		"status":        resp.Status,
		"risk_score":    resp.RiskScore,
		"risk_level":    resp.RiskLevel,
		"flags":         resp.Flags,
		"analyzed_at":   resp.AnalyzedAt,
		"message":       "Transaction accepted and analyzed via gRPC",
	})
}

// GetAllTransactions возвращает список всех транзакций
// @Summary Получить список транзакций
// @Description Возвращает список всех транзакций с пагинацией
// @Tags transactions
// @Accept json
// @Produce json
// @Param limit query int false "Лимит результатов (максимум 500)" default(100)
// @Success 200 {object} map[string]interface{} "Список транзакций"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /transactions [get]
func (h *Handlers) GetAllTransactions(c *gin.Context) {
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	transactions, err := h.transactionService.GetAllTransactions(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": transactions})
}

// GetTransactionStatus возвращает статус транзакции по processing_id
// @Summary Получить статус транзакции
// @Description Возвращает детальную информацию о транзакции и её анализе рисков
// @Tags transactions
// @Accept json
// @Produce json
// @Param processing_id path string true "ID обработки транзакции"
// @Success 200 {object} models.TransactionStatusResponse "Статус транзакции"
// @Failure 404 {object} map[string]string "Not Found"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /transactions/{processing_id} [get]
func (h *Handlers) GetTransactionStatus(c *gin.Context) {
	processingID := c.Param("processing_id")

	status, err := h.transactionService.GetTransactionStatus(processingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transaction status"})
		return
	}

	if status == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ClearAllTransactions очищает все транзакции
// @Summary Очистить все транзакции
// @Description Удаляет все транзакции из базы данных
// @Tags transactions
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Транзакции очищены"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /transactions [delete]
func (h *Handlers) ClearAllTransactions(c *gin.Context) {
	if err := h.transactionService.ClearAllTransactions(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear transactions"})
		return
	}

	logger.LogEvent(logger.EventDBUpdated, "ingestion-service", "sqlite", map[string]interface{}{
		"action": "database_cleared",
	})

	c.JSON(http.StatusOK, gin.H{
		"message":       "All transactions cleared successfully",
		"clear_storage": true,
	})
}

// GenerateRandomTransaction генерирует случайную транзакцию
// @Summary Сгенерировать случайную транзакцию
// @Description Генерирует случайную транзакцию для тестирования
// @Tags transactions
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Сгенерированная транзакция"
// @Router /transactions/generate [get]
func (h *Handlers) GenerateRandomTransaction(c *gin.Context) {
	tx := h.generator.GenerateRandomTransaction()

	c.JSON(http.StatusOK, gin.H{
		"transaction_id":       tx.TransactionID,
		"account_number":       tx.AccountNumber,
		"amount":               tx.Amount,
		"currency":             tx.Currency,
		"transaction_type":     tx.TransactionType,
		"counterparty_account": tx.CounterpartyAccount,
		"counterparty_bank":    tx.CounterpartyBank,
		"counterparty_country": tx.CounterpartyCountry,
		"channel":              tx.Channel,
		"user_id":              tx.UserID,
		"branch_id":            tx.BranchID,
	})
}
