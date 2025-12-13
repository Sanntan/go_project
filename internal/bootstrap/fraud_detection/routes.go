package fraud_detection

import (
	"log"
	"net/http"

	"bank-aml-system/internal/api/rest"
	"bank-aml-system/internal/logger"
	"bank-aml-system/internal/services"
	"bank-aml-system/internal/storage"

	"github.com/gin-gonic/gin"
)

// SetupRoutes настраивает маршруты для fraud detection service
func SetupRoutes(router *gin.Engine, transactionService services.TransactionService, storageRepo storage.TransactionRepository, redisClient interface{ ClearTransactionData() error }) {
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
	}

	// Используем общие endpoints (health, events, stats)
	rest.SetupCommonEndpoints(router)
}
