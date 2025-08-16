#!/bin/bash

# Git BashでUTF-8を強制設定
export LC_ALL=C.UTF-8
export LANG=C.UTF-8

echo "Testing UTF-8 encoding fix"
echo "Current LANG: $LANG"
echo "Current LC_ALL: $LC_ALL"
echo ""

# JSONファイルを作成してから送信
cat > test_deal.json << 'EOF'
{
  "period": "2024-01",
  "dealData": {
    "DealType": "領収書",
    "DealDate": "2025-08-16",
    "DealName": "エンコーディングテスト",
    "DealPartner": "日本語店舗テスト",
    "DealPrice": 5000,
    "DealRemark": "UTF-8テスト用データ",
    "RecStatus": "NEW"
  }
}
EOF

echo "Created JSON file:"
cat test_deal.json
echo ""

echo "Sending with file input method..."
response=$(curl -s -X POST http://localhost:8080/api/v1/deals \
  -H "Content-Type: application/json; charset=utf-8" \
  --data-binary @test_deal.json)

echo "Response:"
echo "$response" | jq .

generated_no=$(echo "$response" | jq -r .dealNo)
echo ""
echo "Generated NO: $generated_no"

echo ""
echo "Checking database (hex):"
sqlite3 data/2024-01/Denchokun.db "SELECT hex(DealPartner) FROM Deals WHERE NO='$generated_no';"

echo ""
echo "Checking database (text):"
sqlite3 data/2024-01/Denchokun.db "SELECT DealPartner FROM Deals WHERE NO='$generated_no';"

echo ""
echo "Expected UTF-8 hex for '日本語店舗テスト':"
echo "E697A5E69CACE8AA9EE5BA97E38386E382B9E38388"

echo ""
echo "Cleanup:"
rm -f test_deal.json