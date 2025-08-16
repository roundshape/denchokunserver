#!/bin/bash

# Ensure UTF-8 encoding
export LC_ALL=C.UTF-8
export LANG=C.UTF-8

echo "Testing multipart with file-based data"
echo "======================================"
echo "Locale: $LANG"
echo ""

# Create dealData as a separate file
cat > dealdata.json << 'EOF'
{
    "period": "2024-01",
    "DealType": "領収書",
    "DealDate": "2025-08-16",
    "DealName": "ファイル経由マルチパート",
    "DealPartner": "UTF-8ファイル経由店",
    "DealPrice": 2500,
    "DealRemark": "multipart file method UTF-8テスト",
    "RecStatus": "NEW"
}
EOF

echo "Created dealData file:"
cat dealdata.json
echo ""

echo "Sending multipart request with file-based dealData..."
response=$(curl -s -X POST http://localhost:8080/api/v1/deals \
  -F "dealData=<dealdata.json" \
  -F "file=@test_receipt.pdf")

echo "Response:"
echo "$response" | jq .

# Extract deal number
dealNo=$(echo "$response" | jq -r .dealNo)
echo ""
echo "Generated Deal Number: $dealNo"

# Check database
echo ""
echo "Database Check:"
echo "Deal Partner (text):"
sqlite3 data/2024-01/Denchokun.db "SELECT DealPartner FROM Deals WHERE NO='$dealNo';"

echo ""
echo "Deal Partner (hex):"
sqlite3 data/2024-01/Denchokun.db "SELECT hex(DealPartner) FROM Deals WHERE NO='$dealNo';"

echo ""
echo "Expected UTF-8 hex for 'UTF-8ファイル経由店':"
echo "5554462D38E38395E382A1E382A4E383ABE7B58CE794B1E5BA97"

echo ""
echo "Cleanup:"
rm -f dealdata.json