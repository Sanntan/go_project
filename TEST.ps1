# Script for testing services
# UTF-8 encoding fix

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Bank AML System - Testing" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if services are running
Write-Host "[1/4] Checking service availability..." -ForegroundColor Yellow

# Check Ingestion Service
try {
    $ingestionHealth = Invoke-WebRequest -Uri "http://localhost:8080/health" -Method GET -TimeoutSec 2 -UseBasicParsing
    if ($ingestionHealth.StatusCode -eq 200) {
        Write-Host "  OK Ingestion Service is running (port 8080)" -ForegroundColor Green
    }
} catch {
    Write-Host "  ERROR Ingestion Service is not responding on port 8080" -ForegroundColor Red
    Write-Host "    Run: go run cmd/ingestion-service/main.go" -ForegroundColor Yellow
    exit 1
}

# Check Fraud Detection Service
try {
    $fraudHealth = Invoke-WebRequest -Uri "http://localhost:8081/health" -Method GET -TimeoutSec 2 -UseBasicParsing
    if ($fraudHealth.StatusCode -eq 200) {
        Write-Host "  OK Fraud Detection Service is running (port 8081)" -ForegroundColor Green
    }
} catch {
    Write-Host "  ERROR Fraud Detection Service is not responding on port 8081" -ForegroundColor Red
    Write-Host "    Run: go run cmd/fraud-detection-service/main.go" -ForegroundColor Yellow
    exit 1
}

Write-Host ""
Write-Host "[2/4] Sending test transaction..." -ForegroundColor Yellow

# Test transaction
$testTransaction = @{
    transaction_id = "TXN-TEST-$(Get-Date -Format 'yyyyMMddHHmmss')"
    account_number = "ACC123456789"
    amount = 15000.50
    currency = "USD"
    transaction_type = "transfer"
    counterparty_account = "ACC987654321"
    counterparty_bank = "Test Bank"
    counterparty_country = "US"
    channel = "online"
    user_id = "user123"
    branch_id = "branch001"
} | ConvertTo-Json

try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/transactions" `
        -Method POST `
        -ContentType "application/json" `
        -Body $testTransaction `
        -UseBasicParsing

    if ($response.StatusCode -eq 201) {
        $result = $response.Content | ConvertFrom-Json
        Write-Host "  OK Transaction sent successfully!" -ForegroundColor Green
        Write-Host "    Processing ID: $($result.processing_id)" -ForegroundColor Cyan
        Write-Host "    Status: $($result.status)" -ForegroundColor Cyan
        
        $processingId = $result.processing_id
        
        Write-Host ""
        Write-Host "[3/4] Waiting for processing (5 seconds)..." -ForegroundColor Yellow
        Start-Sleep -Seconds 5
        
        Write-Host ""
        Write-Host "[4/4] Checking transaction status..." -ForegroundColor Yellow
        
        try {
            $statusResponse = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/transactions/$processingId" `
                -Method GET `
                -UseBasicParsing
            
            if ($statusResponse.StatusCode -eq 200) {
                $status = $statusResponse.Content | ConvertFrom-Json
                Write-Host "  OK Status received!" -ForegroundColor Green
                Write-Host ""
                Write-Host "  Transaction details:" -ForegroundColor Cyan
                Write-Host "    Transaction ID: $($status.transaction_id)" -ForegroundColor White
                Write-Host "    Status: $($status.status)" -ForegroundColor White
                if ($status.risk_score) {
                    Write-Host "    Risk Score: $($status.risk_score)" -ForegroundColor White
                    Write-Host "    Risk Level: $($status.risk_level)" -ForegroundColor White
                }
                if ($status.flags) {
                    Write-Host "    Flags: $($status.flags -join ', ')" -ForegroundColor White
                }
            }
        } catch {
            Write-Host "  WARNING Could not get status (transaction may still be processing)" -ForegroundColor Yellow
        }
        
    } else {
        Write-Host "  ERROR Failed to send transaction: HTTP $($response.StatusCode)" -ForegroundColor Red
    }
} catch {
    Write-Host "  ERROR Failed to send transaction: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "    Make sure Ingestion Service is running" -ForegroundColor Yellow
    exit 1
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Testing completed!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
