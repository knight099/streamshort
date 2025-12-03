#!/bin/bash

# Test script for Subscription API endpoints with authentication
# Uses phone number and OTP for login

BASE_URL="http://localhost:8080"
PHONE="+919650690943"
OTP="069009"
TOKEN=""

echo "🧪 Testing Subscription API Endpoints"
echo "======================================"
echo ""

# Function to get auth token
get_auth_token() {
    echo "🔐 Authenticating with phone: $PHONE"
    
    # Step 1: Send OTP
    echo "📤 Sending OTP..."
    OTP_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/otp/send" \
        -H "Content-Type: application/json" \
        -d "{\"phone\": \"$PHONE\"}")
    
    echo "OTP Send Response: $OTP_RESPONSE"
    
    # Step 2: Verify OTP
    echo "🔑 Verifying OTP..."
    VERIFY_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/otp/verify" \
        -H "Content-Type: application/json" \
        -d "{\"phone\": \"$PHONE\", \"otp\": \"$OTP\"}")
    
    echo "Verify Response: $VERIFY_RESPONSE"
    
    # Extract token from response
    TOKEN=$(echo $VERIFY_RESPONSE | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$TOKEN" ]; then
        echo "❌ Failed to get auth token"
        echo "Full response: $VERIFY_RESPONSE"
        return 1
    fi
    
    echo "✅ Got auth token: ${TOKEN:0:20}..."
    return 0
}

# Test 1: Get available subscription plans
test_get_plans() {
    echo ""
    echo "📋 Test 1: Get available subscription plans"
    echo "GET $BASE_URL/api/subscriptions/plans"
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/api/subscriptions/plans" \
        -H "Authorization: Bearer $TOKEN")
    
    echo "Response: $RESPONSE"
    
    # Pretty print if jq is available
    if command -v jq &> /dev/null; then
        echo "Formatted:"
        echo "$RESPONSE" | jq '.'
    fi
}

# Test 2: Create a subscription
test_create_subscription() {
    echo ""
    echo "💳 Test 2: Create a subscription"
    echo "POST $BASE_URL/api/payments/create-subscription"
    
    RESPONSE=$(curl -s -X POST "$BASE_URL/api/payments/create-subscription" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "target_type": "series",
            "target_id": "49c9cd71-e83d-4863-8757-f8a58e4b96b8",
            "plan_id": "plan_basic_monthly",
            "auto_renew": true
        }')
    
    echo "Response: $RESPONSE"
    
    if command -v jq &> /dev/null; then
        echo "Formatted:"
        echo "$RESPONSE" | jq '.'
    fi
}

# Test 3: Get user subscriptions
test_get_subscriptions() {
    echo ""
    echo "📚 Test 3: Get user subscriptions"
    echo "GET $BASE_URL/api/subscriptions"
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/api/subscriptions" \
        -H "Authorization: Bearer $TOKEN")
    
    echo "Response: $RESPONSE"
    
    if command -v jq &> /dev/null; then
        echo "Formatted:"
        echo "$RESPONSE" | jq '.'
    fi
}

# Test 4: Check subscription status
test_check_status() {
    echo ""
    echo "🔍 Test 4: Check subscription status"
    echo "GET $BASE_URL/api/subscriptions/check?target_type=series&target_id=49c9cd71-e83d-4863-8757-f8a58e4b96b8"
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/api/subscriptions/check?target_type=series&target_id=49c9cd71-e83d-4863-8757-f8a58e4b96b8" \
        -H "Authorization: Bearer $TOKEN")
    
    echo "Response: $RESPONSE"
    
    if command -v jq &> /dev/null; then
        echo "Formatted:"
        echo "$RESPONSE" | jq '.'
    fi
}

# Test 5: Get specific subscription (if exists)
test_get_subscription() {
    echo ""
    echo "📖 Test 5: Get specific subscription"
    
    # First get all subscriptions to get an ID
    SUBSCRIPTIONS=$(curl -s -X GET "$BASE_URL/api/subscriptions" \
        -H "Authorization: Bearer $TOKEN")
    
    # Try to extract first subscription ID using jq if available
    if command -v jq &> /dev/null; then
        SUBSCRIPTION_ID=$(echo "$SUBSCRIPTIONS" | jq -r '.subscriptions[0].id // empty')
        
        if [ ! -z "$SUBSCRIPTION_ID" ]; then
            echo "GET $BASE_URL/api/subscriptions/$SUBSCRIPTION_ID"
            
            RESPONSE=$(curl -s -X GET "$BASE_URL/api/subscriptions/$SUBSCRIPTION_ID" \
                -H "Authorization: Bearer $TOKEN")
            
            echo "Response:"
            echo "$RESPONSE" | jq '.'
        else
            echo "No subscriptions found to test with"
        fi
    else
        echo "jq not available, skipping this test"
    fi
}

# Main test execution
main() {
    echo "🚀 Starting subscription API tests..."
    echo ""
    
    # Check if server is running
    if ! curl -s "$BASE_URL/health" > /dev/null; then
        echo "❌ Server is not running on $BASE_URL"
        echo "Please start the server with: go run main.go"
        exit 1
    fi
    echo "✅ Server is running"
    echo ""
    
    # Get auth token
    if ! get_auth_token; then
        echo "❌ Cannot proceed without auth token"
        exit 1
    fi
    
    # Run all tests
    test_get_plans
    test_create_subscription
    test_get_subscriptions
    test_check_status
    test_get_subscription
    
    echo ""
    echo "✅ All subscription API tests completed!"
}

# Run main function
main
