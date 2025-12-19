package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bank-aml-system/internal/models"
	servicemocks "bank-aml-system/internal/services/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupTestRouter(handlers *Handlers) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	api := router.Group("/api/v1")
	{
		api.POST("/transactions", handlers.HandleTransaction)
		api.GET("/transactions", handlers.GetAllTransactions)
		api.GET("/transactions/:processing_id", handlers.GetTransactionStatus)
		api.DELETE("/transactions", handlers.ClearAllTransactions)
		api.GET("/transactions/generate", handlers.GenerateRandomTransaction)
	}

	return router
}

func TestHandlers_HandleTransaction_Success(t *testing.T) {
	mockService := new(servicemocks.MockTransactionService)
	handlers := NewHandlers(mockService, nil) // nil для grpcClient в тестах
	router := setupTestRouter(handlers)

	reqBody := models.ProcessingRequest{
		Transaction: models.Transaction{
			TransactionID:       "TXN-001",
			AccountNumber:       "ACC123456",
			Amount:              100000.0,
			Currency:            "RUB",
			TransactionType:     "transfer",
			CounterpartyAccount: "ACC789012",
			CounterpartyCountry: "RU",
			Timestamp:           time.Now(),
			Channel:             "online",
		},
	}

	response := &models.ProcessingResponse{
		ProcessingID: "proc_test_123",
		Status:       "pending_review",
		Message:      "Transaction accepted for analysis",
	}

	mockService.On("ProcessTransaction", mock.AnythingOfType("*models.ProcessingRequest")).Return(response, nil)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result models.ProcessingResponse
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "proc_test_123", result.ProcessingID)
	assert.Equal(t, "pending_review", result.Status)

	mockService.AssertExpectations(t)
}

func TestHandlers_HandleTransaction_InvalidJSON(t *testing.T) {
	mockService := new(servicemocks.MockTransactionService)
	handlers := NewHandlers(mockService, nil) // nil для grpcClient в тестах
	router := setupTestRouter(handlers)

	req := httptest.NewRequest("POST", "/api/v1/transactions", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Contains(t, result, "error")

	mockService.AssertNotCalled(t, "ProcessTransaction")
}

func TestHandlers_HandleTransaction_ServiceError(t *testing.T) {
	mockService := new(servicemocks.MockTransactionService)
	handlers := NewHandlers(mockService, nil) // nil для grpcClient в тестах
	router := setupTestRouter(handlers)

	reqBody := models.ProcessingRequest{
		Transaction: models.Transaction{
			TransactionID:       "TXN-001",
			AccountNumber:       "ACC123456",
			Amount:              100000.0,
			Currency:            "RUB",
			TransactionType:     "transfer",
			CounterpartyAccount: "ACC789012",
			CounterpartyCountry: "RU",
			Timestamp:           time.Now(),
			Channel:             "online",
		},
	}

	mockService.On("ProcessTransaction", mock.AnythingOfType("*models.ProcessingRequest")).Return(nil, errors.New("service error"))

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "Failed to process transaction")

	mockService.AssertExpectations(t)
}

func TestHandlers_GetTransactionStatus_Success(t *testing.T) {
	mockService := new(servicemocks.MockTransactionService)
	handlers := NewHandlers(mockService, nil) // nil для grpcClient в тестах
	router := setupTestRouter(handlers)

	processingID := "proc_test_123"
	riskScore := 50
	riskLevel := "medium"
	flags := []string{"large_amount"}

	status := &models.TransactionStatusResponse{
		ProcessingID:  processingID,
		TransactionID: "TXN-001",
		Status:        "reviewed",
		RiskScore:     &riskScore,
		RiskLevel:     &riskLevel,
		Flags:         flags,
	}

	mockService.On("GetTransactionStatus", processingID).Return(status, nil)

	req := httptest.NewRequest("GET", "/api/v1/transactions/"+processingID, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.TransactionStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, processingID, result.ProcessingID)
	assert.Equal(t, flags, result.Flags)

	mockService.AssertExpectations(t)
}

func TestHandlers_GetTransactionStatus_NotFound(t *testing.T) {
	mockService := new(servicemocks.MockTransactionService)
	handlers := NewHandlers(mockService, nil) // nil для grpcClient в тестах
	router := setupTestRouter(handlers)

	processingID := "proc_not_found"

	mockService.On("GetTransactionStatus", processingID).Return(nil, nil)

	req := httptest.NewRequest("GET", "/api/v1/transactions/"+processingID, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "Transaction not found")

	mockService.AssertExpectations(t)
}

func TestHandlers_GetTransactionStatus_ServiceError(t *testing.T) {
	mockService := new(servicemocks.MockTransactionService)
	handlers := NewHandlers(mockService, nil) // nil для grpcClient в тестах
	router := setupTestRouter(handlers)

	processingID := "proc_error"

	mockService.On("GetTransactionStatus", processingID).Return(nil, errors.New("database error"))

	req := httptest.NewRequest("GET", "/api/v1/transactions/"+processingID, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "Failed to get transaction status")

	mockService.AssertExpectations(t)
}

func TestHandlers_GetAllTransactions_Success(t *testing.T) {
	mockService := new(servicemocks.MockTransactionService)
	handlers := NewHandlers(mockService, nil) // nil для grpcClient в тестах
	router := setupTestRouter(handlers)

	transactions := []*models.TransactionStatusResponse{
		{
			ProcessingID:  "proc_1",
			TransactionID: "TXN-001",
			Status:        "reviewed",
		},
		{
			ProcessingID:  "proc_2",
			TransactionID: "TXN-002",
			Status:        "pending_review",
		},
	}

	mockService.On("GetAllTransactions", 100).Return(transactions, nil)

	req := httptest.NewRequest("GET", "/api/v1/transactions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Contains(t, result, "transactions")

	mockService.AssertExpectations(t)
}

func TestHandlers_GetAllTransactions_WithLimit(t *testing.T) {
	mockService := new(servicemocks.MockTransactionService)
	handlers := NewHandlers(mockService, nil) // nil для grpcClient в тестах
	router := setupTestRouter(handlers)

	transactions := []*models.TransactionStatusResponse{}

	mockService.On("GetAllTransactions", 50).Return(transactions, nil)

	req := httptest.NewRequest("GET", "/api/v1/transactions?limit=50", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandlers_GetAllTransactions_ServiceError(t *testing.T) {
	mockService := new(servicemocks.MockTransactionService)
	handlers := NewHandlers(mockService, nil) // nil для grpcClient в тестах
	router := setupTestRouter(handlers)

	mockService.On("GetAllTransactions", 100).Return(nil, errors.New("database error"))

	req := httptest.NewRequest("GET", "/api/v1/transactions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "Failed to get transactions")

	mockService.AssertExpectations(t)
}

func TestHandlers_ClearAllTransactions_Success(t *testing.T) {
	mockService := new(servicemocks.MockTransactionService)
	handlers := NewHandlers(mockService, nil) // nil для grpcClient в тестах
	router := setupTestRouter(handlers)

	mockService.On("ClearAllTransactions").Return(nil)

	req := httptest.NewRequest("DELETE", "/api/v1/transactions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Contains(t, result["message"], "All transactions cleared successfully")
	assert.True(t, result["clear_storage"].(bool))

	mockService.AssertExpectations(t)
}

func TestHandlers_ClearAllTransactions_ServiceError(t *testing.T) {
	mockService := new(servicemocks.MockTransactionService)
	handlers := NewHandlers(mockService, nil) // nil для grpcClient в тестах
	router := setupTestRouter(handlers)

	mockService.On("ClearAllTransactions").Return(errors.New("database error"))

	req := httptest.NewRequest("DELETE", "/api/v1/transactions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Contains(t, result["error"], "Failed to clear transactions")

	mockService.AssertExpectations(t)
}

func TestHandlers_GenerateRandomTransaction(t *testing.T) {
	mockService := new(servicemocks.MockTransactionService)
	handlers := NewHandlers(mockService, nil) // nil для grpcClient в тестах
	router := setupTestRouter(handlers)

	req := httptest.NewRequest("GET", "/api/v1/transactions/generate", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Contains(t, result, "transaction_id")
	assert.Contains(t, result, "account_number")
	assert.Contains(t, result, "amount")
	assert.Contains(t, result, "currency")
}
