# Скрипт для установки protoc на Windows без прав администратора

Write-Host "=== Установка protoc для Windows ===" -ForegroundColor Green

# Проверяем, установлен ли уже protoc
if (Get-Command protoc -ErrorAction SilentlyContinue) {
    Write-Host "protoc уже установлен!" -ForegroundColor Green
    protoc --version
    exit 0
}

Write-Host "`nВыберите способ установки:" -ForegroundColor Yellow
Write-Host "1. Автоматическая загрузка и установка (рекомендуется)"
Write-Host "2. Показать инструкцию для ручной установки"
$choice = Read-Host "Введите номер (1 или 2)"

if ($choice -eq "1") {
    Write-Host "`n=== Автоматическая установка ===" -ForegroundColor Green
    
    # Определяем архитектуру
    $arch = if ([Environment]::Is64BitOperatingSystem) { "win64" } else { "win32" }
    
    # Последняя версия (можно обновить при необходимости)
    $version = "25.1"
    $protocUrl = "https://github.com/protocolbuffers/protobuf/releases/download/v$version/protoc-$version-$arch.zip"
    
    # Путь для установки (в домашней директории пользователя)
    $installDir = "$env:USERPROFILE\protoc"
    $binDir = "$installDir\bin"
    
    Write-Host "Скачивание protoc..." -ForegroundColor Yellow
    Write-Host "URL: $protocUrl" -ForegroundColor Gray
    
    # Создаем временную директорию
    $tempDir = "$env:TEMP\protoc-install"
    if (Test-Path $tempDir) {
        Remove-Item $tempDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    
    $zipPath = "$tempDir\protoc.zip"
    
    try {
        # Скачиваем protoc
        Invoke-WebRequest -Uri $protocUrl -OutFile $zipPath -UseBasicParsing
        Write-Host "Скачивание завершено!" -ForegroundColor Green
        
        # Распаковываем
        Write-Host "Распаковка..." -ForegroundColor Yellow
        if (Test-Path $installDir) {
            Remove-Item $installDir -Recurse -Force
        }
        Expand-Archive -Path $zipPath -DestinationPath $installDir -Force
        
        Write-Host "Установка завершена в: $installDir" -ForegroundColor Green
        
        # Добавляем в PATH для текущей сессии
        $env:Path += ";$binDir"
        
        # Проверяем
        Write-Host "`nПроверка установки..." -ForegroundColor Yellow
        $protocVersion = & "$binDir\protoc.exe" --version 2>&1
        Write-Host $protocVersion -ForegroundColor Green
        
        Write-Host "`n=== ВАЖНО ===" -ForegroundColor Yellow
        Write-Host "protoc установлен в: $installDir" -ForegroundColor Cyan
        Write-Host "Для постоянного использования добавьте в PATH:" -ForegroundColor Yellow
        Write-Host "$binDir" -ForegroundColor Cyan
        Write-Host "`nИли выполните в каждой новой сессии PowerShell:" -ForegroundColor Yellow
        Write-Host "`$env:Path += `";$binDir`"" -ForegroundColor Cyan
        
        Write-Host "`n=== Следующие шаги ===" -ForegroundColor Green
        Write-Host "1. Установите Go плагины:" -ForegroundColor Yellow
        Write-Host "   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest" -ForegroundColor Cyan
        Write-Host "   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest" -ForegroundColor Cyan
        Write-Host "`n2. Сгенерируйте proto файлы:" -ForegroundColor Yellow
        Write-Host "   .\scripts\generate-proto.ps1" -ForegroundColor Cyan
        
    } catch {
        Write-Host "Ошибка при установке: $_" -ForegroundColor Red
        Write-Host "`nПопробуйте ручную установку (см. INSTALL_PROTOC_WINDOWS.md)" -ForegroundColor Yellow
        exit 1
    } finally {
        # Удаляем временные файлы
        if (Test-Path $tempDir) {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
    
} else {
    Write-Host "`n=== Инструкция для ручной установки ===" -ForegroundColor Green
    Write-Host "1. Перейдите на: https://github.com/protocolbuffers/protobuf/releases" -ForegroundColor Yellow
    Write-Host "2. Скачайте последнюю версию для Windows (protoc-*-win64.zip)" -ForegroundColor Yellow
    Write-Host "3. Распакуйте архив в удобное место (например, C:\tools\protoc)" -ForegroundColor Yellow
    Write-Host "4. Добавьте путь к bin директории в PATH" -ForegroundColor Yellow
    Write-Host "   Например: C:\tools\protoc\bin" -ForegroundColor Cyan
    Write-Host "`nПодробная инструкция в файле: INSTALL_PROTOC_WINDOWS.md" -ForegroundColor Green
}

