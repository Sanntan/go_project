## Обзор cmd

В каталоге `cmd` находятся два исполняемых сервиса:
- `fraud-detection-service` — слушает Kafka события о транзакциях, проводит анализ рисков, кеширует результаты в Redis и отдаёт статус по HTTP.
- `ingestion-service` — принимает транзакции по HTTP/Swagger, сохраняет в SQLite, публикует событие в Kafka для анализа и (если доступен Redis) поднимает gRPC сервер.

### cmd/fraud-detection-service/main.go
- `type FraudDetectionService struct { repo; redisClient; riskAnalyzer }` — контейнер зависимостей.
- `main()`:
  - грузит конфиг (`config.Load`), подключает SQLite (`database.NewSQLiteDB`) и Redis (`redis.NewClient`, `InitializeBlacklists`).
  - создаёт `RiskAnalyzer` и Kafka consumer (`kafka.NewConsumer`) с обработчиком `processTransaction`.
  - поднимает Gin HTTP API: `GET /api/v1/transactions/:processing_id` (статус), `GET /health`, `GET /api/v1/events`, `GET /api/v1/stats`, `DELETE /api/v1/transactions` (очистка БД + Redis).
  - запускает HTTP сервер и consumer, организует graceful shutdown.
- `processTransaction(event *models.KafkaTransactionEvent) error`:
  - логирует старт анализа, достаёт полную транзакцию из БД (`GetFullTransactionByProcessingID`).
  - выполняет риск-анализ (`riskAnalyzer.AnalyzeTransaction`), сохраняет результат в Redis (`SaveAnalysis`) и обновляет SQLite (`UpdateTransactionAnalysis`).
  - логирует результат и обновляет статистику рисков в Redis (`IncrementRiskStats`).
- `getTransactionStatus(c *gin.Context)`:
  - читает анализ из Redis (если есть) и статус из БД (`GetTransactionByProcessingID`).
  - формирует `models.TransactionStatusResponse`, добавляя флаги анализа, и возвращает 200/404/500.

### cmd/ingestion-service/main.go
- `type IngestionService struct { repo; producer }` — хранит репозиторий и Kafka producer.
- `main()`:
  - грузит конфиг, подключает SQLite (`NewSQLiteDB`), создаёт репозиторий и Kafka producer.
  - опционально подключает Redis и подготавливает `RiskAnalyzer` для gRPC; при наличии Redis запускает gRPC сервер (`grpc.NewTransactionGRPCServer`, `grpc.StartGRPCServer`).
  - поднимает Gin HTTP API с CORS и Swagger UI (`/swagger/*any`).
  - эндпоинты:  
    - `POST /api/v1/transactions` → `handleTransaction`  
    - `GET /api/v1/transactions` → `getAllTransactions`  
    - `GET /api/v1/transactions/:processing_id` → `getTransactionStatus`  
    - `GET /api/v1/transactions/generate` — разовая генерация случайной транзакции для формы (без сохранения)  
    - `GET /api/v1/events`, `GET /api/v1/stats`, `DELETE /api/v1/transactions`, `GET /health`
  - graceful shutdown HTTP сервера.
- `handleTransaction(c *gin.Context)`:
  - принимает `models.ProcessingRequest`, генерирует `processing_id`, логирует.
  - сохраняет в БД со статусом `pending_review` (`SaveTransaction`).
  - формирует `models.KafkaTransactionEvent` и отправляет в Kafka (`producer.SendTransactionEvent`), логирует.
  - возвращает `models.ProcessingResponse` со статусом 201.
- `getAllTransactions(c *gin.Context)`:
  - читает `limit` (<=500), получает список из БД (`GetAllTransactions`), мапит в `TransactionStatusResponse` и возвращает JSON.
- `getTransactionStatus(c *gin.Context)`:
  - достаёт транзакцию по `processing_id` (`GetTransactionByProcessingID`), возвращает детали или 404/500.

