#!/bin/bash

echo "Current LANG: $LANG"
echo "Current locale:"
locale charmap

# Generate unique deal number
dealNo="TEST$(date +%Y%m%d%H%M%S)"
echo "Testing with deal number: $dealNo"

# Create JSON with Japanese text
dealData=$(cat <<EOF
{
    "NO": "$dealNo",
    "period": "2024-01",
    "DealType": "領収書",
    "DealDate": "$(date +%Y-%m-%d)",
    "DealName": "テスト購入",
    "DealPartner": "日本語テスト店",
    "DealPrice": 2000,
    "DealRemark": "UTF-8確認用",
    "RecStatus": "NEW"
}
EOF
)

# Compact JSON
dealData=$(echo "$dealData" | tr -d '\n' | tr -s ' ')

echo "Sending request..."
response=$(curl -X POST http://localhost:8080/api/v1/deals \
  -F "dealData=$dealData" \
  -F "file=@test_receipt.pdf" 2>/dev/null)

echo "Response:"
echo "$response" | jq .

echo ""
echo "Checking database directly:"
sqlite3 data/2024-01/Denchokun.db "SELECT NO, DealPartner FROM Deals WHERE NO='${dealNo}PC000';"

echo ""
echo "Checking as hex:"
sqlite3 data/2024-01/Denchokun.db "SELECT hex(DealPartner) FROM Deals WHERE NO='${dealNo}PC000';"