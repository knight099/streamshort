#!/bin/bash

# Test script for Creator APIs
# Make sure the server is running first: go run main.go

BASE_URL="http://localhost:8080"

echo "🧪 Testing Creator APIs"
echo "========================"

# Test 1: Send OTP to get access token
echo "1. Sending OTP..."
OTP_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/otp/send" \
  -H "Content-Type: application/json" \
  -d '{"phone": "+919876543210"}')

echo "OTP Response: $OTP_RESPONSE"

# Extract txn_id from response (you'll need to manually verify OTP)
echo "Please check the server logs for OTP and verify manually"

# Test 2: Verify OTP to get tokens
echo "2. Verifying OTP..."
echo "Enter the OTP from server logs:"
read OTP

VERIFY_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/otp/verify" \
  -H "Content-Type: application/json" \
  -d "{\"phone\": \"+919876543210\", \"otp\": \"$OTP\"}")

echo "Verify Response: $VERIFY_RESPONSE"

# Extract access token
ACCESS_TOKEN=$(echo $VERIFY_RESPONSE | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ACCESS_TOKEN" ]; then
    echo "❌ Failed to get access token"
    exit 1
fi

echo "✅ Access token obtained: ${ACCESS_TOKEN:0:20}..."

# Test 3: Creator onboarding
echo "3. Testing Creator Onboarding..."
ONBOARD_RESPONSE=$(curl -s -X POST "$BASE_URL/api/creators/onboard" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "display_name": "Arjun Films",
    "bio": "Short films in Hindi & Marathi",
    "kyc_document_s3_path": "s3://uploads/kyc/kyc_doc_1234.jpg"
  }')

echo "Onboard Response: $ONBOARD_RESPONSE"

# Extract creator ID
CREATOR_ID=$(echo $ONBOARD_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$CREATOR_ID" ]; then
    echo "❌ Failed to get creator ID"
    exit 1
fi

echo "✅ Creator ID: $CREATOR_ID"

# Test 4: Get Creator Profile
echo "4. Testing Get Creator Profile..."
PROFILE_RESPONSE=$(curl -s -X GET "$BASE_URL/api/creators/profile" \
  -H "Authorization: Bearer $ACCESS_TOKEN")

echo "Profile Response: $PROFILE_RESPONSE"

# Test 5: Get Creator Dashboard
echo "5. Testing Creator Dashboard..."
DASHBOARD_RESPONSE=$(curl -s -X GET "$BASE_URL/api/creators/$CREATOR_ID/dashboard" \
  -H "Authorization: Bearer $ACCESS_TOKEN")

echo "Dashboard Response: $DASHBOARD_RESPONSE"

# Test 6: Update Creator Profile
echo "6. Testing Update Creator Profile..."
UPDATE_RESPONSE=$(curl -s -X PUT "$BASE_URL/api/creators/profile" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "display_name": "Arjun Films Updated",
    "bio": "Updated bio for Arjun Films"
  }')

echo "Update Response: $UPDATE_RESPONSE"

echo ""
echo "🎉 All Creator API tests completed!"
echo "Check the responses above for any errors."
