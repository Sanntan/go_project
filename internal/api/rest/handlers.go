package rest

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"bank-aml-system/internal/generator"
	"bank-aml-system/internal/logger"
	"bank-aml-system/internal/models"
	"bank-aml-system/internal/services"
)

// Handlers содержит обработчики REST API
type Handlers struct {
	transactionService services.TransactionService
	generator          *generator.TransactionGenerator
}

// NewHandlers создает новые обработчики REST API
func NewHandlers(transactionService services.TransactionService) *Handlers {
	return &Handlers{
		transactionService: transactionService,
		generator:          generator.NewTransactionGenerator(),
	}
}

// HandleTransaction обрабатывает POST запрос на создание транзакции
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

// GetAllTransactions возвращает список всех транзакций
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

