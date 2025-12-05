# Быстрый старт

## Шаг 1: Настройка окружения

```powershell
# Создайте .env файл из примера (если его нет)
if (-not (Test-Path .env)) {
    Copy-Item env.example .env
}
```

## Шаг 2: Запуск инфраструктуры

```powershell
# Запустите SQLite, Redis и Kafka
docker-compose up -d

# Дождитесь, пока все контейнеры станут healthy (1-2 минуты)
docker-compose ps
```

## Шаг 3: Запуск сервисов

### Терминал 1: Transaction Ingestion Service
```powershell
go run cmd/ingestion-service/main.go
```

### Терминал 2: Fraud Detection Service
```powershell
go run cmd/fraud-detection-service/main.go
```

## Шаг 4: Запуск веб-интерфейса

```powershell
cd frontend
npm install
npm run dev
```

Веб-интерфейс откроется автоматически на `http://localhost:3000`

## Шаг 5: Тестирование

### Через веб-интерфейс

1. Заполните форму отправки транзакции
2. Нажмите "Отправить транзакцию"
3. Транзакция появится в списке ниже
4. Через несколько секунд статус изменится на "reviewed" с оценкой риска
5. Кликните на транзакцию для просмотра деталей

### Через скрипт

```powershell
.\TEST.ps1
```

### Вручную через PowerShell

```powershell
$body = @{
    transaction_id = "txn_001"
    account_number = "40817810099910004321"
    amount = 1250000.00
    currency = "RUB"
    transaction_type = "international_transfer"
    counterparty_account = "CH9300762011623852957"
    counterparty_bank = "UBSWCHZH80A"
    counterparty_country = "CH"
    timestamp = "2024-01-15T14:30:00Z"
    channel = "online_banking"
    user_id = "user_12345"
    branch_id = "branch_moscow_001"
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/transactions" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body

Write-Host "Processing ID: $($response.processing_id)"
```

### Проверка статуса

```powershell
# Замените {processing_id} на полученный ID
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/transactions/{processing_id}"
```

## Ожидаемый результат

Транзакция с суммой 1,250,000 RUB и офшорным контрагентом (Швейцария) должна получить:
- **Risk Score**: 70+ (large_amount + offshore_counterparty)
- **Risk Level**: `high`
- **Flags**: `["large_amount", "offshore_counterparty"]`

## Возможности веб-интерфейса

- ✅ Визуальный статус сервисов (Online/Offline)
- ✅ Форма для отправки транзакций
- ✅ Список всех транзакций с цветовой индикацией риска
- ✅ Детальная информация о каждой транзакции
- ✅ Автоматическое обновление данных
- ✅ Красивый и понятный интерфейс для демонстрации
