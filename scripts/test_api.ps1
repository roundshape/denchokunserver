$API_BASE = "http://localhost:8080/api/v1"
$PERIOD = "2024-01"

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "電帳君 API Server Test Script" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "1. Health Check" -ForegroundColor Yellow
Write-Host "---------------"
Invoke-RestMethod -Uri "$API_BASE/health" -Method GET | ConvertTo-Json
Write-Host ""

Write-Host "2. Create Period (2024-01)" -ForegroundColor Yellow
Write-Host "-------------------------"
Invoke-RestMethod -Uri "$API_BASE/periods/$PERIOD/connect" -Method POST | ConvertTo-Json
Write-Host ""

Write-Host "3. Get Available Periods" -ForegroundColor Yellow
Write-Host "------------------------"
Invoke-RestMethod -Uri "$API_BASE/periods" -Method GET | ConvertTo-Json
Write-Host ""

Write-Host "4. Create Test Deal" -ForegroundColor Yellow
Write-Host "-------------------"
$dealData = @{
    period = "2024-01"
    dealData = @{
        NO = "D240115001"
        DealType = "領収書"
        DealDate = "2024-01-15"
        DealName = "文房具購入"
        DealPartner = "オフィス用品店"
        DealPrice = 1500
        DealRemark = "ペン、ノート等"
        RecStatus = "NEW"
    }
} | ConvertTo-Json -Depth 3

Invoke-RestMethod -Uri "$API_BASE/deals" -Method POST -Body $dealData -ContentType "application/json" | ConvertTo-Json
Write-Host ""

Write-Host "5. Get Deals List" -ForegroundColor Yellow
Write-Host "-----------------"
Invoke-RestMethod -Uri "$API_BASE/deals?period=$PERIOD" -Method GET | ConvertTo-Json
Write-Host ""

Write-Host "6. Get Specific Deal" -ForegroundColor Yellow
Write-Host "--------------------"
Invoke-RestMethod -Uri "$API_BASE/deals/D240115001?period=$PERIOD" -Method GET | ConvertTo-Json
Write-Host ""

Write-Host "7. Create Deal Partner" -ForegroundColor Yellow
Write-Host "----------------------"
$partnerData = @{
    name = "テスト商店"
} | ConvertTo-Json

Invoke-RestMethod -Uri "$API_BASE/deal-partners?period=$PERIOD" -Method POST -Body $partnerData -ContentType "application/json" | ConvertTo-Json
Write-Host ""

Write-Host "8. Get Deal Partners" -ForegroundColor Yellow
Write-Host "--------------------"
Invoke-RestMethod -Uri "$API_BASE/deal-partners?period=$PERIOD" -Method GET | ConvertTo-Json
Write-Host ""

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Test Complete!" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan