#!/bin/bash

# Test CloudFront Signed URL Configuration
# This script tests if CloudFront signing is working correctly

set -e

API_URL="${API_URL:-https://api.episodd.com}"
EPISODE_ID="${1:-386db34f-e43f-42d3-ab83-66ad5e766c0a}"

echo "========================================="
echo "CloudFront Signed URL Test"
echo "========================================="
echo ""

# Test 1: Debug endpoint
echo "Test 1: Checking CloudFront configuration..."
echo "GET $API_URL/debug/cloudfront"
echo ""

RESPONSE=$(curl -s "$API_URL/debug/cloudfront")
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

# Extract signed URL if available
SIGNED_URL=$(echo "$RESPONSE" | jq -r '.signed_url // empty' 2>/dev/null)

if [ -n "$SIGNED_URL" ]; then
    echo "✅ CloudFront signing is configured"
    echo ""
    
    # Test 2: Try to access the signed URL
    echo "Test 2: Testing signed URL access..."
    echo "HEAD $SIGNED_URL"
    echo ""
    
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -I "$SIGNED_URL")
    echo "HTTP Status: $HTTP_CODE"
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo "✅ Signed URL works! CloudFront configuration is correct."
    elif [ "$HTTP_CODE" = "404" ]; then
        echo "⚠️  File not found (expected for test path). Signing mechanism works!"
    elif [ "$HTTP_CODE" = "403" ]; then
        echo "❌ 403 Forbidden - CloudFront configuration issue detected!"
        echo ""
        echo "Possible causes:"
        echo "  1. Key-Pair-Id not in CloudFront's trusted key group"
        echo "  2. Private key doesn't match public key in CloudFront"
        echo "  3. CloudFront behavior not configured for signed URLs"
        echo ""
        echo "See CLOUDFRONT_403_TROUBLESHOOTING.md for detailed steps"
    else
        echo "⚠️  Unexpected status code: $HTTP_CODE"
    fi
else
    echo "❌ CloudFront signing not configured or debug endpoint failed"
    echo "Check AWS environment variables:"
    echo "  - AWS_CLOUDFRONT_KEY_PAIR_ID"
    echo "  - AWS_CLOUDFRONT_PRIVATE_KEY"
    echo "  - AWS_CLOUDFRONT_DOMAIN"
fi

echo ""
echo "========================================="
echo ""

# Test 3: Check actual episode manifest
if [ -n "$EPISODE_ID" ]; then
    echo "Test 3: Checking episode manifest..."
    echo "Note: This requires authentication token"
    echo ""
    
    if [ -n "$AUTH_TOKEN" ]; then
        echo "GET $API_URL/api/episodes/$EPISODE_ID/manifest"
        MANIFEST_RESPONSE=$(curl -s -H "Authorization: Bearer $AUTH_TOKEN" "$API_URL/api/episodes/$EPISODE_ID/manifest")
        echo "$MANIFEST_RESPONSE" | jq '.' 2>/dev/null || echo "$MANIFEST_RESPONSE"
        
        MANIFEST_URL=$(echo "$MANIFEST_RESPONSE" | jq -r '.manifest_url // empty' 2>/dev/null)
        
        if [ -n "$MANIFEST_URL" ]; then
            echo ""
            echo "Testing manifest URL access..."
            HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -I "$MANIFEST_URL")
            echo "HTTP Status: $HTTP_CODE"
            
            if [ "$HTTP_CODE" = "200" ]; then
                echo "✅ Episode manifest URL works!"
            elif [ "$HTTP_CODE" = "403" ]; then
                echo "❌ 403 Forbidden on episode manifest URL"
                echo "This is the issue causing video playback to fail!"
            fi
        fi
    else
        echo "⚠️  Set AUTH_TOKEN environment variable to test episode manifest"
        echo "Example: export AUTH_TOKEN='your_jwt_token'"
    fi
fi

echo ""
echo "========================================="
echo "Summary"
echo "========================================="
echo ""
echo "If you see 403 errors, follow these steps:"
echo "1. Read CLOUDFRONT_403_TROUBLESHOOTING.md"
echo "2. Verify CloudFront behaviors are configured"
echo "3. Check that key pair is in trusted key group"
echo "4. Ensure private key matches public key"
echo ""
