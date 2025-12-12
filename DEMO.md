
### 1. Инфраструктура

```powershell
docker-compose up -d
```
### 2. Сервисы

**Ingestion Service:**
```powershell
go run cmd/ingestion-service/main.go
```

**Fraud Detection Service:**
```powershell
go run cmd/fraud-detection-service/main.go
```

**Фронтенд**
```powershell
cd frontend
npm run dev
```

### 3. Swagger UI - REST API документация

```
http://localhost:8080/swagger/index.html
```

**Команда для быстрого теста:**
```powershell
# Случайная транзакции через API
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/transactions/generate"
```

### 4. gRPC - Высокопроизводительный API

**Список доступных сервисов:**
```powershell
grpcurl -plaintext localhost:50051 list
```

**Генерация случайной транзакции:**
```powershell
grpcurl -plaintext -d '{}' localhost:50051 transaction.TransactionService/GenerateRandomTransaction
```

**Получение статуса транзакции:**
```powershell
grpcurl -plaintext -d '{"processing_id": "proc_ваш-id"}' localhost:50051 transaction.TransactionService/GetTransactionStatus
```

### 5. Веб-интерфейс

```
http://localhost:3000
```

### 6. REST API - Примеры запросов

**Отправка транзакции:**
```powershell
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
```

**Проверка статуса:**
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/transactions/$($response.processing_id)"
```

**Получение всех транзакций:**
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/transactions?limit=10"
```
## Проверка работы системы

**Health checks:**
```powershell
# Ingestion Service
Invoke-RestMethod -Uri "http://localhost:8080/health"

# Fraud Detection Service
Invoke-RestMethod -Uri "http://localhost:8081/health"
```

**Статистика:**
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/stats"
```

**События:**
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/events?limit=10"
```
