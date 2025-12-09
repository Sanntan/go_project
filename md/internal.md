## Обзор internal

Пакет `internal` содержит все доменные службы проекта: конфиг, хранение, анализ, очереди, кэш, логи, генераторы и gRPC.

### config
- `config.go` — загрузка `.env`/переменных среды через `godotenv`. Структуры `Config` с секциями DB/Redis/Kafka/Server. Функции: `Load()` собирает конфиг; `getEnv`, `getEnvAsInt` — чтение с дефолтами.

### database
- `sqlite.go` — обёртка `SQLiteDB`:
  - `NewSQLiteDB(cfg)` — создаёт каталог, открывает SQLite (DSN с WAL и foreign_keys), настраивает пул, вызывает `initSchema`, пинг.
  - `initSchema()` — создаёт таблицу `transactions` и индексы.
  - `Close()` — закрывает соединение.
- `repository.go` — CRUD/чтение:
  - `NewRepository(db)` — конструктор.
  - `SaveTransaction(processingID, *Transaction)` — insert со статусом `pending_review`.
  - `UpdateTransactionAnalysis(processingID, riskScore, riskLevel, analysisTime)` — выставляет статус `reviewed`, риск и время анализа.
  - `GetTransactionByProcessingID(processingID)` — читает статус транзакции (может вернуть nil при отсутствии).
  - `GetFullTransactionByProcessingID(processingID)` — возвращает полные данные для анализа (может nil).
  - `GetAllTransactions(limit)` — список транзакций по дате убыв.
  - `ClearAllTransactions()` — truncate таблицы.

### fraud
- `rules.go` — анализатор рисков:
  - `RiskAnalyzer` с зависимостью `redis.Client`.
  - `AnalyzeTransaction(tx)` — считает risk score и флаги на основе суммы, офшоров (Redis set), blacklist счёта (Redis), времени, частоты операций (Redis counters), типа операции, канала, валюты, «круглых» сумм; инкрементирует счётчики; возвращает `RiskAnalysis` с рекомендацией.
  - Вспомогательные `calculateRiskLevel`, `getActionRecommendation`.

### generator
- `transaction_generator.go` — генератор тестовых транзакций:
  - `TransactionGenerator` с локальным `rand`.
  - `GenerateTransaction(riskLevel)` — создаёт транзакцию с заданным уровнем риска, вызывая профили low/medium/high.
  - `GenerateRandomTransaction()` — полностью случайная транзакция (сумма, валюта, тип, канал, контрагент, время).
  - Приватные `generateLowRisk/MediumRisk/HighRisk`, выбор офшорных/безопасных стран и банков, округление суммы до 2 знаков.

### grpc
- `server.go` — gRPC сервер для `TransactionService` (из `api/proto`):
  - `TransactionGRPCServer` хранит repo, producer, redis, riskAnalyzer, generator.
  - `NewTransactionGRPCServer(...)` — конструктор.
  - `AnalyzeTransaction` — парсит запрос, генерирует `processing_id`, сохраняет в БД, отправляет событие в Kafka, делает синхронный риск-анализ, сохраняет в Redis, обновляет БД, возвращает ответ с риск-полями.
  - `GetTransactionStatus` — читает анализ из Redis, иначе из БД, формирует ответ.
  - `GenerateRandomTransaction` — отдаёт случайную транзакцию из генератора.
  - `StartGRPCServer(cfg, srv)` — поднимает gRPC на `cfg.Server.GRPCPort`, включает reflection, graceful shutdown на контекст.

### kafka
- `producer.go` — синхронный producer (Sarama):
  - `NewProducer(cfg)` — конфиг acks/all, retries, создаёт producer, берёт топик из `cfg.Kafka.TransactionTopic`.
  - `SendTransactionEvent(event)` — маршалит в JSON, шлёт в Kafka, логирует partition/offset.
  - `Close()` — закрывает producer.
- `consumer.go` — consumer group:
  - `NewConsumer(cfg, handler)` — создаёт consumer group с round-robin и offset oldest; топик `cfg.Kafka.TransactionTopic`.
  - `Start(ctx)` — запускает consume loop, обрабатывает ошибки, закрывает по ctx.
  - Внутренний `consumerGroupHandler` вызывает переданный `handler(*KafkaTransactionEvent)` для каждого сообщения.

### logger
- `event_logger.go` — in-memory лог последних событий:
  - Глобальный `EventLogger` (до 1000 событий).
  - `LogEvent`, `GetEvents(limit)`, `GetStats()` — запись/чтение/агрегации по компоненту/сервису/типу.
  - Сериализация в JSON с RFC3339 timestamp.

### models
- `transaction.go` — DTO/модели:
  - `Transaction`, `ProcessingRequest`, `ProcessingResponse`.
  - `TransactionStatus` (хранение в БД), `TransactionStatusResponse` (API ответ).
  - `RiskAnalysis` (результат анализа).
  - `KafkaTransactionEvent`, `KafkaTransactionData` (события для Kafka).

### redis
- `client.go` — работа с Redis (go-redis v9):
  - `NewClient(cfg)` — подключение, ping.
  - `SaveAnalysis/GetAnalysis` — кэш результатов анализа с TTL 1h.
  - `IncrementRiskStats` — счётчики по уровням риска.
  - `IncrementAccountDailyCount/GetAccountDailyCount` — лимиты частоты по счёту (24h TTL).
  - `IsAccountBlacklisted`, `IsHighRiskCountry` — membership в сетах.
  - `InitializeBlacklists` — наполняет high_risk_countries.
  - `AddToBlacklist` — добавить счёт в blacklist.
  - `ClearTransactionData` — удалить ключи транзакций/статистики/лимитов, оставляя списки.

