#!/bin/bash

# Test script for Subscription API endpoints
# Make sure the server is running on localhost:8080

BASE_URL="http://localhost:8080"
TOKEN=""

echo "🧪 Testing Subscription API Endpoints"
echo "====================================="

# Function to get auth token
get_auth_token() {
    echo "🔐 Getting authentication token..."
    
    # Send OTP
    OTP_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/otp/send" \
        -H "Content-Type: application/json" \
        -d '{"phone": "+919876543210"}')
    
    echo "OTP Response: $OTP_RESPONSE"
    
    # Extract OTP (in real scenario, user would receive this via SMS)
    # For testing, we'll use a mock OTP
    OTP_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/otp/verify" \
        -H "Content-Type: application/json" \
        -d '{"phone": "+919876543210", "otp": "123456"}')
    
    echo "Verify OTP Response: $OTP_RESPONSE"
    
    # Extract token from response
    TOKEN=$(echo $OTP_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$TOKEN" ]; then
        echo "❌ Failed to get auth token"
        exit 1
    fi
    
    echo "✅ Got auth token: $TOKEN"
}

# Test 1: Get available subscription plans
test_get_plans() {
    echo ""
    echo "📋 Test 1: Get available subscription plans"
    echo "GET $BASE_URL/api/subscriptions/plans"
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/api/subscriptions/plans" \
        -H "Authorization: Bearer $TOKEN")
    
    echo "Response: $RESPONSE"
    echo "Status: $?"
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
            "target_id": "test-series-id",
            "plan_id": "plan_basic_monthly",
            "auto_renew": true
        }')
    
    echo "Response: $RESPONSE"
    echo "Status: $?"
}

# Test 3: Get user subscriptions
test_get_subscriptions() {
    echo ""
    echo "📚 Test 3: Get user subscriptions"
    echo "GET $BASE_URL/api/subscriptions"
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/api/subscriptions" \
        -H "Authorization: Bearer $TOKEN")
    
    echo "Response: $RESPONSE"
    echo "Status: $?"
}

# Test 4: Check subscription status
test_check_status() {
    echo ""
    echo "🔍 Test 4: Check subscription status"
    echo "GET $BASE_URL/api/subscriptions/check?target_type=series&target_id=test-series-id"
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/api/subscriptions/check?target_type=series&target_id=test-series-id" \
        -H "Authorization: Bearer $TOKEN")
    
    echo "Response: $RESPONSE"
    echo "Status: $?"
}

# Test 5: Get specific subscription
test_get_subscription() {
    echo ""
    echo "📖 Test 5: Get specific subscription"
    echo "GET $BASE_URL/api/subscriptions/{id}"
    
    # First get subscriptions to get an ID
    SUBSCRIPTIONS=$(curl -s -X GET "$BASE_URL/api/subscriptions" \
        -H "Authorization: Bearer $TOKEN")
    
    echo "Subscriptions: $SUBSCRIPTIONS"
    
    # Extract first subscription ID (if any)
    SUBSCRIPTION_ID=$(echo $SUBSCRIPTIONS | grep -o '"[^"]*"' | head -1 | tr -d '"')
    
    if [ ! -z "$SUBSCRIPTION_ID" ]; then
        echo "Testing with subscription ID: $SUBSCRIPTION_ID"
        
        RESPONSE=$(curl -s -X GET "$BASE_URL/api/subscriptions/$SUBSCRIPTION_ID" \
            -H "Authorization: Bearer $TOKEN")
        
        echo "Response: $RESPONSE"
    else
        echo "No subscriptions found to test with"
    fi
    
    echo "Status: $?"
}

# Test 6: Cancel subscription
test_cancel_subscription() {
    echo ""
    echo "❌ Test 6: Cancel subscription"
    echo "POST $BASE_URL/api/subscriptions/{id}/cancel"
    
    # Get subscriptions to get an ID
    SUBSCRIPTIONS=$(curl -s -X GET "$BASE_URL/api/subscriptions" \
        -H "Authorization: Bearer $TOKEN")
    
    SUBSCRIPTION_ID=$(echo $SUBSCRIPTIONS | grep -o '"[^"]*"' | head -1 | tr -d '"')
    
    if [ ! -z "$SUBSCRIPTION_ID" ]; then
        echo "Cancelling subscription ID: $SUBSCRIPTION_ID"
        
        RESPONSE=$(curl -s -X POST "$BASE_URL/api/subscriptions/$SUBSCRIPTION_ID/cancel" \
            -H "Authorization: Bearer $TOKEN")
        
        echo "Response: $RESPONSE"
    else
        echo "No subscriptions found to cancel"
    fi
    
    echo "Status: $?"
}

# Test 7: Renew subscription
test_renew_subscription() {
    echo ""
    echo "🔄 Test 7: Renew subscription"
    echo "POST $BASE_URL/api/subscriptions/{id}/renew"
    
    # Get subscriptions to get an ID
    SUBSCRIPTIONS=$(curl -s -X GET "$BASE_URL/api/subscriptions" \
        -H "Authorization: Bearer $TOKEN")
    
    SUBSCRIPTION_ID=$(echo $SUBSCRIPTIONS | grep -o '"[^"]*"' | head -1 | tr -d '"')
    
    if [ ! -z "$SUBSCRIPTION_ID" ]; then
        echo "Renewing subscription ID: $SUBSCRIPTION_ID"
        
        RESPONSE=$(curl -s -X POST "$BASE_URL/api/subscriptions/$SUBSCRIPTION_ID/renew" \
            -H "Authorization: Bearer $TOKEN")
        
        echo "Response: $RESPONSE"
    else
        echo "No subscriptions found to renew"
    fi
    
    echo "Status: $?"
}

# Main test execution
main() {
    echo "🚀 Starting subscription API tests..."
    
    # Get auth token first
    get_auth_token
    
    if [ -z "$TOKEN" ]; then
        echo "❌ Cannot proceed without auth token"
        exit 1
    fi
    
    # Run all tests
    test_get_plans
    test_create_subscription
    test_get_subscriptions
    test_check_status
    test_get_subscription
    test_cancel_subscription
    test_renew_subscription
    
    echo ""
    echo "✅ All subscription API tests completed!"
}

# Run main function
main
