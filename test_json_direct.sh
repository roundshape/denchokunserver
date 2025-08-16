#!/bin/bash

echo "Testing with direct JSON API (not multipart)..."

# Generate unique deal number
dealNo="JSON$(date +%Y%m%d%H%M%S)"
echo "Deal number: $dealNo"

# Create test data
jsonData=$(cat <<EOF
{
  "period": "2024-01",
  "dealData": {
    "NO": "$dealNo",
    "DealType": "領収書",
    "DealDate": "$(date +%Y-%m-%d)",
    "DealName": "テスト購入",
    "DealPartner": "日本語テスト店",
    "DealPrice": 3000,
    "DealRemark": "JSON直接送信テスト",
    "RecStatus": "NEW"
  }
}
EOF
)

echo "Sending JSON request..."
echo "$jsonData" | jq .

response=$(curl -X POST http://localhost:8080/api/v1/deals \
  -H "Content-Type: application/json; charset=utf-8" \
  -d "$jsonData" \
  2>/dev/null)

echo ""
echo "Response:"
echo "$response" | jq .

echo ""
echo "Checking database:"
sqlite3 data/2024-01/Denchokun.db "SELECT NO, DealPartner FROM Deals WHERE NO='${dealNo}PC000';"

echo ""
echo "Hex check:"
sqlite3 data/2024-01/Denchokun.db "SELECT hex(DealPartner) FROM Deals WHERE NO='${dealNo}PC000';"