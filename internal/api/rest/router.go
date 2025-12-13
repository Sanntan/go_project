package rest

import (
	"net/http"
	"strconv"

	"bank-aml-system/internal/logger"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// CORSMiddleware возвращает middleware для обработки CORS
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// SetupCommonEndpoints добавляет общие endpoints (health, events, stats) к роутеру
func SetupCommonEndpoints(router *gin.Engine) {
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
}

// SetupRouter настраивает маршруты REST API
func SetupRouter(handlers *Handlers) *gin.Engine {
	router := gin.Default()

	// CORS middleware
	router.Use(CORSMiddleware())

	router.Use(gin.Logger(), gin.Recovery())

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")))

	// API endpoints
	api := router.Group("/api/v1")
	{
		api.POST("/transactions", handlers.HandleTransaction)
		api.GET("/transactions", handlers.GetAllTransactions)
		api.GET("/transactions/:processing_id", handlers.GetTransactionStatus)
		api.DELETE("/transactions", handlers.ClearAllTransactions)
		api.GET("/transactions/generate", handlers.GenerateRandomTransaction)
	}

	// Общие endpoints (health, events, stats)
	SetupCommonEndpoints(router)

	return router
}
