# Bank AML System

Система мониторинга и анализа банковских транзакций на предмет подозрительной активности и возможного отмывания денег.

## 📋 Что нужно установить

### 1. Go (версия 1.21 или выше)

**Windows:**
1. Скачайте установщик с [golang.org/dl](https://golang.org/dl/)
2. Запустите установщик и следуйте инструкциям
3. Проверьте установку:
   ```powershell
   go version
   ```

**Linux/Mac:**
```bash
# Linux
sudo apt-get update
sudo apt-get install golang-go

# Mac (через Homebrew)
brew install go
```

### 2. Node.js (для фронтенда)

**Windows:**
1. Скачайте установщик с [nodejs.org](https://nodejs.org/)
2. Рекомендуется LTS версия
3. Проверьте установку:
   ```powershell
   node --version
   npm --version
   ```

**Linux/Mac:**
```bash
# Linux
sudo apt-get install nodejs npm

# Mac
brew install node
```

### 3. Docker Desktop

**Windows:**
1. Скачайте с [docker.com/products/docker-desktop](https://www.docker.com/products/docker-desktop/)
2. Установите и запустите Docker Desktop
3. Проверьте установку:
   ```powershell
   docker --version
   docker-compose --version
   ```

**Linux:**
```bash
sudo apt-get update
sudo apt-get install docker.io docker-compose
```

**Mac:**
```bash
brew install --cask docker
```

### 4. Protocol Buffers Compiler (protoc) - для gRPC

**Windows:**
```powershell
# Автоматическая установка (рекомендуется)
.\scripts\install-protoc-windows.ps1

# Или через Chocolatey (требуются права администратора)
choco install protoc -y
```

**Linux:**
```bash
sudo apt-get install protobuf-compiler
```

**Mac:**
```bash
brew install protobuf
```

Проверьте установку:
```powershell
protoc --version
```

## 🚀 Установка и запуск проекта

### Шаг 1: Клонирование проекта

```powershell
# Перейдите в директорию проекта
cd "C:\Users\santa\Desktop\go project"
```

### Шаг 2: Настройка окружения

```powershell
# Создайте .env файл из примера
if (-not (Test-Path .env)) {
    Copy-Item env.example .env
}
```

### Шаг 3: Установка зависимостей Go

```powershell
# Установите все зависимости проекта
go mod download

# Или автоматически обновите зависимости
go mod tidy
```

### Шаг 4: Установка зависимостей для Swagger и gRPC

```powershell
# Swagger зависимости
go install github.com/swaggo/swag/cmd/swag@latest
go get github.com/swaggo/files github.com/swaggo/gin-swagger

# gRPC зависимости
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go get google.golang.org/grpc

# grpcurl для тестирования gRPC (опционально)
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

### Шаг 5: Генерация кода

```powershell
# Генерация Swagger документации
swag init -g cmd/ingestion-service/main.go -o ./docs

# Генерация gRPC кода (Windows)
.\scripts\generate-proto.ps1

# Или вручную:
protoc --go_out=./api/proto --go_opt=paths=source_relative --go-grpc_out=./api/proto --go-grpc_opt=paths=source_relative ./api/proto/transaction.proto
```

### Шаг 6: Запуск инфраструктуры (Docker)

```powershell
# Запустите все контейнеры (SQLite, Redis, Kafka, Zookeeper)
docker-compose up -d

# Проверьте статус контейнеров
docker-compose ps

# Дождитесь, пока все контейнеры станут "healthy" (1-2 минуты)
```

### Шаг 7: Запуск сервисов

Откройте **два отдельных терминала**:

#### Терминал 1: Transaction Ingestion Service
```powershell
cd "C:\Users\santa\Desktop\go project"
go run cmd/ingestion-service/main.go
```

Сервис запустится на порту **8080** (REST API) и **50051** (gRPC).

#### Терминал 2: Fraud Detection Service
```powershell
cd "C:\Users\santa\Desktop\go project"
go run cmd/fraud-detection-service/main.go
```

Сервис запустится на порту **8081**.

### Шаг 8: Запуск фронтенда (опционально)

```powershell
cd frontend
npm install
npm run dev
```

Фронтенд откроется на `http://localhost:3000` (или другом порту, который укажет Vite).

## 📖 Использование

### Swagger UI

Откройте в браузере: **http://localhost:8080/swagger/index.html**

Здесь вы можете:
- Просмотреть все API endpoints
- Протестировать запросы прямо из браузера
- Увидеть схемы запросов и ответов

### gRPC

```powershell
# Список сервисов
grpcurl -plaintext localhost:50051 list

# Генерация случайной транзакции
grpcurl -plaintext -d '{}' localhost:50051 transaction.TransactionService/GenerateRandomTransaction
```

### Веб-интерфейс

Откройте фронтенд в браузере и используйте графический интерфейс для работы с транзакциями.

## 🏗️ Архитектура

```
Core Banking System
  ↓
Transaction Ingestion Service (REST API + gRPC)
  ↓
Kafka (асинхронная обработка)
  ↓
Fraud Detection Service
  ↓
SQLite (хранение) + Redis (кэширование)
```

## 📁 Структура проекта

```
bank-aml-system/
├── cmd/
│   ├── ingestion-service/       # REST API и gRPC сервер
│   └── fraud-detection-service/ # Сервис анализа рисков
├── internal/
│   ├── config/                  # Конфигурация
│   ├── database/                # SQLite репозиторий
│   ├── redis/                   # Redis клиент
│   ├── kafka/                   # Kafka producer/consumer
│   ├── fraud/                   # Бизнес-логика анализа рисков
│   ├── grpc/                    # gRPC сервер
│   └── models/                  # Модели данных
├── api/proto/                   # gRPC proto файлы
├── docs/                        # Swagger документация
├── frontend/                    # Vue.js фронтенд
├── scripts/                     # Вспомогательные скрипты
├── docker-compose.yml           # Docker конфигурация
└── go.mod                       # Go зависимости
```

## 🔧 Остановка сервисов

```powershell
# Остановите Go сервисы (Ctrl+C в терминалах)

# Остановите Docker контейнеры
docker-compose down

# Остановите и удалите volumes (очистка данных)
docker-compose down -v
```

## ❓ Решение проблем

### Проблема: "go: command not found"
- Убедитесь, что Go установлен и добавлен в PATH
- Перезапустите терминал после установки

### Проблема: Docker не запускается
- Убедитесь, что Docker Desktop запущен
- Проверьте, что порты 9092, 6379, 2181 свободны

### Проблема: "protoc not found"
- Установите protoc (см. раздел "Что нужно установить")
- Добавьте в PATH: `$env:Path += ";C:\Users\santa\protoc\bin"`

### Проблема: Swagger не открывается
- Убедитесь, что выполнили `swag init`
- Проверьте, что сервис запущен на порту 8080

### Проблема: gRPC не работает
- Убедитесь, что Redis запущен (gRPC требует Redis)
- Проверьте, что выполнили генерацию proto файлов

## 📚 Дополнительная информация

Для демонстрации работы системы см. файл **DEMO.md**

## 📝 Лицензия

Этот проект создан в образовательных целях.
