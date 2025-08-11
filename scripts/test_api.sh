#!/bin/bash

API_BASE="http://localhost:8080/api/v1"
PERIOD="2024-01"

echo "========================================="
echo "電帳君 API Server Test Script"
echo "========================================="
echo ""

echo "1. Health Check"
echo "---------------"
curl -X GET "$API_BASE/health"
echo -e "\n"

echo "2. Create Period (2024-01)"
echo "-------------------------"
curl -X POST "$API_BASE/periods/$PERIOD/connect"
echo -e "\n"

echo "3. Get Available Periods"
echo "------------------------"
curl -X GET "$API_BASE/periods"
echo -e "\n"

echo "4. Create Test Deal"
echo "-------------------"
curl -X POST "$API_BASE/deals" \
  -H "Content-Type: application/json" \
  -d '{
    "period": "2024-01",
    "dealData": {
      "NO": "D240115001",
      "DealType": "領収書",
      "DealDate": "2024-01-15",
      "DealName": "文房具購入",
      "DealPartner": "オフィス用品店",
      "DealPrice": 1500,
      "DealRemark": "ペン、ノート等",
      "RecStatus": "NEW"
    }
  }'
echo -e "\n"

echo "5. Get Deals List"
echo "-----------------"
curl -X GET "$API_BASE/deals?period=$PERIOD"
echo -e "\n"

echo "6. Get Specific Deal"
echo "--------------------"
curl -X GET "$API_BASE/deals/D240115001?period=$PERIOD"
echo -e "\n"

echo "7. Create Deal Partner"
echo "----------------------"
curl -X POST "$API_BASE/deal-partners?period=$PERIOD" \
  -H "Content-Type: application/json" \
  -d '{"name": "テスト商店"}'
echo -e "\n"

echo "8. Get Deal Partners"
echo "--------------------"
curl -X GET "$API_BASE/deal-partners?period=$PERIOD"
echo -e "\n"

echo "========================================="
echo "Test Complete!"
echo "========================================="