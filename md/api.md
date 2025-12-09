## Обзор api/proto

В каталоге `api/proto` хранится описание gRPC контракта и сгенерированные Go-артефакты для сервиса транзакций.

### transaction.proto
- `package transaction`, `go_package = "bank-aml-system/api/proto;transaction"`.
- `service TransactionService`:
  - `AnalyzeTransaction(AnalyzeTransactionRequest) → AnalyzeTransactionResponse` — запускает анализ рисков.
  - `GetTransactionStatus(GetTransactionStatusRequest) → GetTransactionStatusResponse` — возвращает статус/результаты анализа по `processing_id`.
  - `GenerateRandomTransaction(GenerateRandomTransactionRequest) → GenerateRandomTransactionResponse` — выдаёт случайную транзакцию (для демо/форм).
- Сообщения:
  - `AnalyzeTransactionRequest`: id транзакции, счёт, сумма, валюта, тип, реквизиты контрагента, канал, user_id, branch_id, timestamp.
  - `AnalyzeTransactionResponse`: processing_id, риск-скор, уровень риска, флаги, рекомендация, время анализа, статус.
  - `GetTransactionStatusRequest`: processing_id.
  - `GetTransactionStatusResponse`: processing_id, transaction_id, статус, риск-скор/уровень, флаги, время анализа.
  - `GenerateRandomTransactionRequest`: пустой запрос.
  - `GenerateRandomTransactionResponse`: сгенерированные поля транзакции (id, счёт, сумма, валюта, тип, контрагент, канал, user/branch).

### Сгенерированные файлы
- `transaction.pb.go` — модели сообщений и вспомогательные методы protobuf.
- `transaction_grpc.pb.go` — клиент/сервер stubs gRPC.
- Эти файлы подключаются в gRPC-сервере (`internal/grpc`) и вызываются из ingestion-service.

### Генерация
- Для Linux/macOS: `./scripts/generate-proto.sh`
- Для Windows: `pwsh ./scripts/generate-proto.ps1`
- Оба скрипта используют `paths=source_relative`, поэтому корректные файлы лежат прямо в `api/proto`.

### Как исправить дубликат `api/proto/api/proto`
- Источник: генерация запускалась без `paths=source_relative` или из другого CWD, поэтому пути вложились в `api/proto/api/proto`.
- Правильно: удалить всю вложенную папку `api/proto/api/proto` и заново сгенерировать через `./scripts/generate-proto.sh` (Linux/macOS) или `pwsh ./scripts/generate-proto.ps1` (Windows). После этого файлы должны находиться только в `api/proto`.

