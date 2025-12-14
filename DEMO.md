
### 1. Инфраструктура


docker-compose up -d

### 2. Сервисы

**Ingestion Service:**

go run cmd/ingestion-service/main.go


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


**Статистика:**

Invoke-RestMethod -Uri "http://localhost:8080/api/v1/stats"


**События:**

Invoke-RestMethod -Uri "http://localhost:8080/api/v1/events?limit=10"

