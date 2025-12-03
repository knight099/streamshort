#!/bin/bash

# Quick Payment Test Script
# This tests the subscription creation flow

echo "🧪 Quick Payment Test"
echo "===================="
echo ""

BASE_URL="http://localhost:8080"

# Step 1: Create OTP
echo "1️⃣ Creating test OTP..."
go run cmd/create_test_otp/main.go > /dev/null 2>&1
echo "✅ OTP created"
echo ""

# Step 2: Authenticate
echo "2️⃣ Authenticating..."
TOKEN=$(curl -s -X POST "$BASE_URL/auth/otp/verify" \
  -H "Content-Type: application/json" \
  -d '{"phone": "+919650690943", "otp": "069009"}' | \
  grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ Failed to get token"
    exit 1
fi

echo "✅ Authenticated (token: ${TOKEN:0:20}...)"
echo ""

# Step 3: Get subscription plans
echo "3️⃣ Fetching subscription plans..."
PLANS=$(curl -s -X GET "$BASE_URL/api/subscriptions/plans" \
  -H "Authorization: Bearer $TOKEN")

if command -v jq &> /dev/null; then
    echo "$PLANS" | jq -r '.[] | "   - \(.name): ₹\(.amount) for \(.duration) days"'
else
    echo "$PLANS"
fi
echo ""

# Step 4: Create subscription
echo "4️⃣ Creating subscription..."
SUB_RESPONSE=$(curl -s -X POST "$BASE_URL/api/payments/create-subscription" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "target_type": "series",
    "target_id": "49c9cd71-e83d-4863-8757-f8a58e4b96b8",
    "plan_id": "plan_basic_monthly",
    "auto_renew": true
  }')

if command -v jq &> /dev/null; then
    echo "$SUB_RESPONSE" | jq '.'
    SUB_ID=$(echo "$SUB_RESPONSE" | jq -r '.subscription_id')
else
    echo "$SUB_RESPONSE"
    SUB_ID=$(echo "$SUB_RESPONSE" | grep -o '"subscription_id":"[^"]*"' | cut -d'"' -f4)
fi

if [ -z "$SUB_ID" ]; then
    echo "❌ Failed to create subscription"
    exit 1
fi

echo ""
echo "✅ Subscription created: $SUB_ID"
echo ""

# Step 5: Verify subscription
echo "5️⃣ Checking subscription status..."
STATUS=$(curl -s -X GET "$BASE_URL/api/subscriptions/check?target_type=series&target_id=49c9cd71-e83d-4863-8757-f8a58e4b96b8" \
  -H "Authorization: Bearer $TOKEN")

if command -v jq &> /dev/null; then
    HAS_ACCESS=$(echo "$STATUS" | jq -r '.has_access')
    echo "   Has Access: $HAS_ACCESS"
    echo "$STATUS" | jq '.'
else
    echo "$STATUS"
fi

echo ""
echo "🎉 Payment test completed successfully!"
echo ""
echo "📝 Next steps:"
echo "   1. Check your database to see the subscription"
echo "   2. Test with Razorpay checkout UI (see PAYMENT_TESTING_GUIDE.md)"
echo "   3. Test webhook events from Razorpay dashboard"
