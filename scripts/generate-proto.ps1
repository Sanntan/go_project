# PowerShell скрипт для генерации Go кода из proto файлов

Write-Host "=== Генерация gRPC кода из proto файлов ===" -ForegroundColor Green

# Проверяем наличие protoc
if (-not (Get-Command protoc -ErrorAction SilentlyContinue)) {
    Write-Host "ОШИБКА: protoc не найден в PATH!" -ForegroundColor Red
    Write-Host "`nУстановка protoc:" -ForegroundColor Yellow
    Write-Host "1. : .\scripts\install-protoc-windows.ps1" -ForegroundColor Cyan
    exit 1
}

# Проверяем наличие Go плагинов
$protocGenGo = Get-Command protoc-gen-go -ErrorAction SilentlyContinue
$protocGenGoGrpc = Get-Command protoc-gen-go-grpc -ErrorAction SilentlyContinue

if (-not $protocGenGo) {
    Write-Host "Установка protoc-gen-go..." -ForegroundColor Yellow
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ОШИБКА: Не удалось установить protoc-gen-go" -ForegroundColor Red
        exit 1
    }
}

if (-not $protocGenGoGrpc) {
    Write-Host "Установка protoc-gen-go-grpc..." -ForegroundColor Yellow
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ОШИБКА: Не удалось установить protoc-gen-go-grpc" -ForegroundColor Red
        exit 1
    }
}

$PROTO_DIR = "./api/proto"
$OUTPUT_DIR = "./api/proto"

# Проверяем наличие proto файлов
$protoFiles = Get-ChildItem -Path $PROTO_DIR -Filter "*.proto" -ErrorAction SilentlyContinue
if (-not $protoFiles) {
    Write-Host "ОШИБКА: Proto файлы не найдены в $PROTO_DIR" -ForegroundColor Red
    exit 1
}

Write-Host "Генерация кода из proto файлов..." -ForegroundColor Yellow

# Генерируем Go код из proto файлов
foreach ($file in $protoFiles) {
    Write-Host "Обработка: $($file.Name)" -ForegroundColor Gray
    & protoc --go_out=$OUTPUT_DIR `
             --go_opt=paths=source_relative `
             --go-grpc_out=$OUTPUT_DIR `
             --go-grpc_opt=paths=source_relative `
             $file.FullName
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ОШИБКА: Не удалось сгенерировать код из $($file.Name)" -ForegroundColor Red
        exit 1
    }
}

Write-Host "`nProto files generated successfully!" -ForegroundColor Green
Write-Host "Сгенерированные файлы:" -ForegroundColor Yellow
Get-ChildItem -Path $OUTPUT_DIR -Filter "*.pb.go" | ForEach-Object {
    Write-Host "  - $($_.Name)" -ForegroundColor Cyan
}

