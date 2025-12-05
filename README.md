# Bank AML System (Anti-Money Laundering)

Система мониторинга и анализа банковских транзакций на предмет подозрительной активности и возможного отмывания денег.

## Архитектура

Система состоит из двух основных сервисов:

1. **Transaction Ingestion Service** - REST API для приема транзакций от core banking системы
2. **Fraud Detection Service** - сервис анализа транзакций на предмет мошенничества

### Поток данных

```
Core Banking/Security Team 
  → Transaction Ingestion Service (REST API)
  → Kafka (асинхронная обработка)
  → Fraud Detection Service
  → PostgreSQL (хранение) + Redis (кэширование)
```

## Технологии

- **Go 1.21+**
- **SQLite** - хранение транзакций (в Docker контейнере)
- **Redis 7** - кэширование результатов анализа
- **Apache Kafka** - асинхронная обработка событий
- **Docker Compose** - оркестрация инфраструктуры

## Установка и запуск

### Предварительные требования

1. **Go 1.21 или выше**
   - Скачайте с [golang.org](https://golang.org/dl/)
   - Установите и добавьте в PATH

2. **Docker Desktop для Windows**
   - Скачайте с [docker.com](https://www.docker.com/products/docker-desktop/)
   - Установите и запустите Docker Desktop

### Шаг 1: Клонирование и настройка проекта

```bash
# Перейдите в директорию проекта
cd "C:\Users\santa\Desktop\go project"

# Создайте .env файл из примера (если его нет)
if (-not (Test-Path .env)) {
    Copy-Item env.example .env
}

# Или вручную скопируйте пример конфигурации
copy env.example .env
```

### Шаг 2: Запуск инфраструктуры (SQLite, Redis, Kafka)

```bash
# Запустите все сервисы через Docker Compose
docker-compose up -d

# Проверьте статус контейнеров
docker-compose ps
```

Ожидайте, пока все контейнеры перейдут в статус "healthy" (может занять 1-2 минуты).

### Шаг 3: Установка зависимостей Go

```bash
# Установите зависимости проекта
go mod download

# Или просто выполните (автоматически скачает зависимости)
go mod tidy
```

### Шаг 4: Запуск сервисов

Откройте **два отдельных терминала**:

#### Терминал 1: Transaction Ingestion Service

```bash
cd "C:\Users\santa\Desktop\go project"
go run cmd/ingestion-service/main.go
```

Сервис запустится на порту **8080**.

#### Терминал 2: Fraud Detection Service

```bash
cd "C:\Users\santa\Desktop\go project"
go run cmd/fraud-detection-service/main.go
```

Сервис запустится на порту **8081** и начнет потреблять события из Kafka.

## API Endpoints

### Transaction Ingestion Service (порт 8080)

#### POST /api/v1/transactions

Регистрация новой транзакции.

**Request:**
```json
{
  "transaction_id": "txn_987654321",
  "account_number": "40817810099910004321",
  "amount": 1250000.00,
  "currency": "RUB",
  "transaction_type": "international_transfer",
  "counterparty_account": "CH9300762011623852957",
  "counterparty_bank": "UBSWCHZH80A",
  "counterparty_country": "CH",
  "timestamp": "2024-01-15T14:30:00Z",
  "channel": "online_banking",
  "user_id": "user_12345",
  "branch_id": "branch_moscow_001"
}
```

**Response (201 Created):**
```json
{
  "processing_id": "proc_550e8400-e29b-41d4-a716-446655440000",
  "status": "pending_review",
  "message": "Transaction accepted for analysis"
}
```

#### GET /api/v1/transactions/{processing_id}

Проверка статуса обработки транзакции.

**Response:**
```json
{
  "processing_id": "proc_550e8400-e29b-41d4-a716-446655440000",
  "transaction_id": "txn_987654321",
  "status": "reviewed",
  "risk_score": 85,
  "risk_level": "high",
  "analysis_timestamp": "2024-01-15T14:30:05Z",
  "flags": ["large_amount", "offshore_counterparty", "unusual_behavior"]
}
```

### Fraud Detection Service (порт 8081)

#### GET /api/v1/transactions/{processing_id}

Проверка статуса транзакции (с кэшем из Redis).

## Бизнес-правила оценки рисков

Система оценивает транзакции по следующим критериям:

- **Крупные суммы** (> 1 млн руб): +30 баллов
- **Офшорные юрисдикции** (БВО, Кайманы и т.д.): +40 баллов
- **Необычное время** (операции ночью 00:00-06:00): +15 баллов
- **Высокая частота** (> 10 транзакций/день): +25 баллов
- **Черные списки**: +100 баллов (автоматически high risk)

### Уровни риска

- **0-30 баллов**: `low` - авто-одобрение
- **31-70 баллов**: `medium` - запись в лог
- **71-100+ баллов**: `high` - требование верификации

## Тестирование

### Пример запроса через curl

```bash
# Отправка транзакции
curl -X POST http://localhost:8080/api/v1/transactions ^
  -H "Content-Type: application/json" ^
  -d "{\"transaction_id\":\"txn_001\",\"account_number\":\"40817810099910004321\",\"amount\":1250000.00,\"currency\":\"RUB\",\"transaction_type\":\"international_transfer\",\"counterparty_account\":\"CH9300762011623852957\",\"counterparty_bank\":\"UBSWCHZH80A\",\"counterparty_country\":\"CH\",\"timestamp\":\"2024-01-15T14:30:00Z\",\"channel\":\"online_banking\",\"user_id\":\"user_12345\",\"branch_id\":\"branch_moscow_001\"}"

# Проверка статуса (замените processing_id на полученный)
curl http://localhost:8080/api/v1/transactions/{processing_id}
```

### Пример через PowerShell

```powershell
# Отправка транзакции
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

Invoke-RestMethod -Uri "http://localhost:8080/api/v1/transactions" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body
```

## Структура проекта

```
bank-aml-system/
├── cmd/
│   ├── ingestion-service/       # Transaction Ingestion Service
│   └── fraud-detection-service/ # Fraud Detection Service
├── internal/
│   ├── config/                   # Конфигурация
│   ├── database/                 # SQLite клиент и репозиторий
│   ├── redis/                    # Redis клиент
│   ├── kafka/                    # Kafka producer и consumer
│   ├── models/                   # Модели данных
│   └── fraud/                    # Бизнес-логика анализа рисков
├── docker-compose.yml            # Docker Compose конфигурация
├── go.mod                        # Go модуль
└── README.md                     # Документация
```

## Остановка сервисов

```bash
# Остановите Go сервисы (Ctrl+C в терминалах)

# Остановите Docker контейнеры
docker-compose down

# Остановите и удалите volumes (если нужно очистить данные)
docker-compose down -v
```

## Зависимости Go

Основные зависимости проекта:

- `github.com/gin-gonic/gin` - HTTP веб-фреймворк
- `github.com/IBM/sarama` - Kafka клиент
- `modernc.org/sqlite` - SQLite драйвер
- `github.com/redis/go-redis/v9` - Redis клиент
- `github.com/google/uuid` - Генерация UUID
- `github.com/joho/godotenv` - Загрузка .env файлов

Все зависимости автоматически устанавливаются при выполнении `go mod download` или `go mod tidy`.

## Troubleshooting

### Проблема: Kafka не запускается

Убедитесь, что порты 9092, 2181 свободны. Проверьте логи:
```bash
docker-compose logs kafka
```

### Проблема: SQLite connection error

Убедитесь, что контейнер SQLite запущен:
```bash
docker-compose ps
docker-compose logs sqlite-storage
```

### Проблема: Redis connection refused

Проверьте статус Redis контейнера:
```bash
docker-compose logs redis
```

## Лицензия

Этот проект создан в образовательных целях.

