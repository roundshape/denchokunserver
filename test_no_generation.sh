#!/bin/bash

echo "Testing NO generation (server-side only)"
echo "========================================="
echo ""

# Test 1: JSON API without NO
echo "Test 1: JSON API (NO should be auto-generated)"
echo "------------------------------------------------"
response1=$(curl -s -X POST http://localhost:8080/api/v1/deals \
  -H "Content-Type: application/json; charset=utf-8" \
  -d '{
    "period": "2024-01",
    "dealData": {
      "DealType": "領収書",
      "DealDate": "'$(date +%Y-%m-%d)'",
      "DealName": "JSONテスト",
      "DealPartner": "テスト店舗A",
      "DealPrice": 1000,
      "DealRemark": "サーバー生成NO確認",
      "RecStatus": "NEW"
    }
  }')

echo "$response1" | jq .
generated_no1=$(echo "$response1" | jq -r .dealNo)
echo "Generated NO: $generated_no1"
echo ""

# Test 2: JSON API with NO (should be ignored)
echo "Test 2: JSON API with NO='CUSTOM123' (should be ignored)"
echo "--------------------------------------------------------"
response2=$(curl -s -X POST http://localhost:8080/api/v1/deals \
  -H "Content-Type: application/json; charset=utf-8" \
  -d '{
    "period": "2024-01",
    "dealData": {
      "NO": "CUSTOM123",
      "DealType": "請求書",
      "DealDate": "'$(date +%Y-%m-%d)'",
      "DealName": "カスタムNOテスト",
      "DealPartner": "テスト店舗B",
      "DealPrice": 2000,
      "DealRemark": "NOを送信しても無視されるはず",
      "RecStatus": "NEW"
    }
  }')

echo "$response2" | jq .
generated_no2=$(echo "$response2" | jq -r .dealNo)
echo "Generated NO: $generated_no2"
echo ""

# Test 3: Multipart with NO (should be ignored)
echo "Test 3: Multipart with NO='MULTI456' (should be ignored)"
echo "--------------------------------------------------------"
response3=$(curl -s -X POST http://localhost:8080/api/v1/deals \
  -F 'dealData={"NO":"MULTI456","period":"2024-01","DealType":"領収書","DealDate":"'$(date +%Y-%m-%d)'","DealName":"マルチパートテスト","DealPartner":"テスト店舗C","DealPrice":3000,"DealRemark":"multipartでもNO無視","RecStatus":"NEW"}' \
  -F "file=@test_receipt.pdf")

echo "$response3" | jq .
generated_no3=$(echo "$response3" | jq -r .dealNo)
echo "Generated NO: $generated_no3"
echo ""

# Check database
echo "Database Check"
echo "=============="
echo "Checking if custom NOs were saved (should not exist):"
sqlite3 data/2024-01/Denchokun.db "SELECT NO, DealPartner FROM Deals WHERE NO IN ('CUSTOM123', 'MULTI456');"

echo ""
echo "Latest 3 records (all should have server-generated NOs):"
sqlite3 data/2024-01/Denchokun.db "SELECT NO, DealPartner, DealPrice FROM Deals ORDER BY RegDate DESC LIMIT 3;"

echo ""
echo "NO format check (should be YYYYMMDDHHmmssPCXXX[-NN]):"
echo "- $generated_no1"
echo "- $generated_no2"
echo "- $generated_no3"