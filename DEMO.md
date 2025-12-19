
### 1. Инфраструктура


docker-compose up -d

### 2. Сервисы

**Ingestion Service:**

go run cmd/ingestion-service/main.go


**Fraud Detection Service:**

go run cmd/fraud-detection-service/main.go


**Фронтенд**

cd frontend
npm run dev


### 3. Swagger UI - REST API документация


http://localhost:8080/swagger/index.html


**Команда для быстрого теста:**

# Случайная транзакции через API
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/transactions/generate"


### 4. gRPC - Высокопроизводительный API

**Список доступных сервисов:**

grpcurl -plaintext localhost:50051 list


**Генерация случайной транзакции:**

grpcurl -plaintext -d '{}' localhost:50051 transaction.TransactionService/GenerateRandomTransaction


**Отправка транзакции через gRPC (AnalyzeTransaction) - транзакция автоматически сохраняется, отправляется в Kafka, обрабатывается fraud-сервисом и отображается на фронте:**

grpcurl -plaintext -d '{"transaction_id":"TXN-GRPC-001","account_number":"ACC987654321","amount":2500000.0,"currency":"RUB","transaction_type":"international_transfer","counterparty_account":"ACC111222333","counterparty_bank":"Offshore Bank","counterparty_country":"KY","channel":"online","user_id":"user123","branch_id":"branch001","timestamp":"2024-01-15T14:30:00Z"}' localhost:50051 transaction.TransactionService/AnalyzeTransaction


**Быстрый тест с высоким риском (офшорная страна + крупная сумма):**

grpcurl -plaintext -d '{"transaction_id":"TXN-HIGH-RISK","account_number":"ACC123456","amount":5000000.0,"currency":"USD","transaction_type":"international_transfer","counterparty_country":"KY","channel":"online","timestamp":"2024-01-15T14:30:00Z"}' localhost:50051 transaction.TransactionService/AnalyzeTransaction


**Получение статуса транзакции:**

grpcurl -plaintext -d '{"processing_id": "proc_ваш-id"}' localhost:50051 transaction.TransactionService/GetTransactionStatus


### 5. Веб-интерфейс


http://localhost:3000


### 6. REST API - Примеры запросов

**Отправка транзакции:**
$body = @{
    transaction_id = "TXN-DEMO-001"
    account_number = "ACC123456789"
    amount = 1500000.0
    currency = "RUB"
    transaction_type = "international_transfer"
    counterparty_country = "KY"
    channel = "online"
    timestamp = "2024-01-15T14:30:00Z"
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/transactions" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body

Write-Host "Processing ID: $($response.processing_id)"


**Проверка статуса:**

Invoke-RestMethod -Uri "http://localhost:8080/api/v1/transactions/$($response.processing_id)"


**Получение всех транзакций:**

Invoke-RestMethod -Uri "http://localhost:8080/api/v1/transactions?limit=10"

## Проверка работы системы

**Health checks:**
# Ingestion Service
Invoke-RestMethod -Uri "http://localhost:8080/health"

# Fraud Detection Service
Invoke-RestMethod -Uri "http://localhost:8081/health"


**Статистика:**

Invoke-RestMethod -Uri "http://localhost:8080/api/v1/stats"


**События:**

Invoke-RestMethod -Uri "http://localhost:8080/api/v1/events?limit=10"

