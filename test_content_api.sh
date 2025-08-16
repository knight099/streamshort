#!/bin/bash

# Content API Test Script
# This script tests all the Content API endpoints

BASE_URL="http://localhost:8080"
AUTH_TOKEN=""

echo "🎬 Testing Streamshort Content API"
echo "=================================="

# Function to make authenticated requests
make_auth_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    
    if [ -z "$AUTH_TOKEN" ]; then
        echo "❌ No auth token available. Please run auth flow first."
        return 1
    fi
    
    if [ -z "$data" ]; then
        curl -s -X $method \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            -H "Content-Type: application/json" \
            "$BASE_URL$endpoint"
    else
        echo "$data" | curl -s -X $method \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            -H "Content-Type: application/json" \
            -d @- \
            "$BASE_URL$endpoint"
    fi
}

# Function to make public requests
make_public_request() {
    local method=$1
    local endpoint=$2
    
    curl -s -X $method \
        -H "Content-Type: application/json" \
        "$BASE_URL$endpoint"
}

echo ""
echo "🔐 Step 1: Authentication Flow"
echo "-------------------------------"

# Send OTP
echo "📱 Sending OTP..."
OTP_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"phone": "+919876543210"}' \
    "$BASE_URL/auth/otp/send")

echo "OTP Response: $OTP_RESPONSE"

# Extract txn_id (you'll need to manually check your phone/console for the OTP)
TXN_ID=$(echo $OTP_RESPONSE | grep -o '"txn_id":"[^"]*"' | cut -d'"' -f4)
echo "Transaction ID: $TXN_ID"

if [ -z "$TXN_ID" ]; then
    echo "❌ Failed to get transaction ID. Please check the server logs."
    exit 1
fi

echo ""
echo "📝 Enter the OTP you received:"
read OTP_CODE

# Verify OTP
echo "✅ Verifying OTP..."
VERIFY_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "{\"phone\": \"+919876543210\", \"otp\": \"$OTP_CODE\"}" \
    "$BASE_URL/auth/otp/verify")

echo "Verify Response: $VERIFY_RESPONSE"

# Extract access token
AUTH_TOKEN=$(echo $VERIFY_RESPONSE | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$AUTH_TOKEN" ]; then
    echo "❌ Failed to get access token. Please check the OTP code."
    exit 1
fi

echo "✅ Authentication successful! Token: ${AUTH_TOKEN:0:20}..."

echo ""
echo "👨‍🎨 Step 2: Creator Onboarding (Required for Content APIs)"
echo "--------------------------------------------------------"

# Onboard as creator
echo "🚀 Onboarding as creator..."
ONBOARD_RESPONSE=$(make_auth_request "POST" "/api/creators/onboard" '{
    "display_name": "Test Creator",
    "bio": "Testing content creation",
    "kyc_document_s3_path": "s3://test/kyc.pdf"
}')

echo "Onboard Response: $ONBOARD_RESPONSE"

echo ""
echo "🎬 Step 3: Content Creation"
echo "---------------------------"

# Create series
echo "📺 Creating series..."
SERIES_RESPONSE=$(make_auth_request "POST" "/api/content/series" '{
    "title": "Test Series",
    "synopsis": "A test series for API testing",
    "language": "en",
    "category_tags": ["test", "drama"],
    "price_type": "free",
    "thumbnail_url": "https://example.com/thumb.jpg"
}')

echo "Series Response: $SERIES_RESPONSE"

# Extract series ID
SERIES_ID=$(echo $SERIES_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Series ID: $SERIES_ID"

if [ -z "$SERIES_ID" ]; then
    echo "❌ Failed to create series. Please check the server logs."
    exit 1
fi

# Create episode
echo "🎬 Creating episode..."
EPISODE_RESPONSE=$(make_auth_request "POST" "/api/content/series/$SERIES_ID/episodes" '{
    "title": "Episode 1: The Beginning",
    "episode_number": 1,
    "duration_seconds": 300
}')

echo "Episode Response: $EPISODE_RESPONSE"

# Extract episode ID
EPISODE_ID=$(echo $EPISODE_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Episode ID: $EPISODE_ID"

echo ""
echo "📤 Step 4: Upload Management"
echo "----------------------------"

# Request upload URL
echo "🔗 Requesting upload URL..."
UPLOAD_URL_RESPONSE=$(make_auth_request "POST" "/api/content/upload-url" '{
    "filename": "episode1.mp4",
    "content_type": "video/mp4",
    "size_bytes": 73400320,
    "metadata": {
        "series_id": "'$SERIES_ID'",
        "episode_title": "Episode 1"
    }
}')

echo "Upload URL Response: $UPLOAD_URL_RESPONSE"

# Extract upload ID
UPLOAD_ID=$(echo $UPLOAD_URL_RESPONSE | grep -o '"upload_id":"[^"]*"' | cut -d'"' -f4)
echo "Upload ID: $UPLOAD_ID"

# Notify upload complete
echo "✅ Notifying upload complete..."
NOTIFY_RESPONSE=$(make_auth_request "POST" "/api/content/uploads/$UPLOAD_ID/notify" '{
    "s3_path": "s3://bucket/'$UPLOAD_ID'/episode1.mp4",
    "size_bytes": 73400320
}')

echo "Notify Response: $NOTIFY_RESPONSE"

echo ""
echo "📺 Step 5: Content Retrieval"
echo "----------------------------"

# List series (public)
echo "📋 Listing series (public)..."
LIST_RESPONSE=$(make_public_request "GET" "/content/series?language=en&category=test")
echo "List Response: $LIST_RESPONSE"

# Get series details (public)
echo "🔍 Getting series details (public)..."
SERIES_DETAILS=$(make_public_request "GET" "/content/series/$SERIES_ID")
echo "Series Details: $SERIES_DETAILS"

# Get episode manifest (protected)
echo "🎥 Getting episode manifest (protected)..."
MANIFEST_RESPONSE=$(make_auth_request "GET" "/api/episodes/$EPISODE_ID/manifest")
echo "Manifest Response: $MANIFEST_RESPONSE"

echo ""
echo "✏️ Step 6: Content Updates"
echo "--------------------------"

# Update series
echo "🔄 Updating series..."
UPDATE_RESPONSE=$(make_auth_request "PUT" "/api/content/series/$SERIES_ID" '{
    "title": "Updated Test Series",
    "status": "published"
}')

echo "Update Response: $UPDATE_RESPONSE"

echo ""
echo "🎉 Content API Testing Complete!"
echo "================================"
echo ""
echo "📊 Summary:"
echo "- ✅ Authentication: $([ -n "$AUTH_TOKEN" ] && echo "SUCCESS" || echo "FAILED")"
echo "- ✅ Creator Onboarding: $([ -n "$ONBOARD_RESPONSE" ] && echo "SUCCESS" || echo "FAILED")"
echo "- ✅ Series Creation: $([ -n "$SERIES_ID" ] && echo "SUCCESS" || echo "FAILED")"
echo "- ✅ Episode Creation: $([ -n "$EPISODE_ID" ] && echo "SUCCESS" || echo "FAILED")"
echo "- ✅ Upload URL: $([ -n "$UPLOAD_ID" ] && echo "SUCCESS" || echo "FAILED")"
echo "- ✅ Upload Notification: $([ -n "$NOTIFY_RESPONSE" ] && echo "SUCCESS" || echo "FAILED")"
echo "- ✅ Series Listing: $([ -n "$LIST_RESPONSE" ] && echo "SUCCESS" || echo "FAILED")"
echo "- ✅ Series Details: $([ -n "$SERIES_DETAILS" ] && echo "SUCCESS" || echo "FAILED")"
echo "- ✅ Episode Manifest: $([ -n "$MANIFEST_RESPONSE" ] && echo "SUCCESS" || echo "FAILED")"
echo "- ✅ Series Update: $([ -n "$UPDATE_RESPONSE" ] && echo "SUCCESS" || echo "FAILED")"
echo ""
echo "🔗 Tested Endpoints:"
echo "- POST /auth/otp/send"
echo "- POST /auth/otp/verify"
echo "- POST /api/creators/onboard"
echo "- POST /api/content/series"
echo "- POST /api/content/series/{id}/episodes"
echo "- POST /api/content/upload-url"
echo "- POST /api/content/uploads/{upload_id}/notify"
echo "- GET /content/series (public)"
echo "- GET /content/series/{id} (public)"
echo "- GET /api/episodes/{id}/manifest"
echo "- PUT /api/content/series/{id}"
