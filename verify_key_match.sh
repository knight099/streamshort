#!/bin/bash

echo "========================================="
echo "Verify CloudFront Key Pair Match"
echo "========================================="
echo ""

PRIVATE_KEY_PATH="/Users/vaibhaw/.aws/cloudfront-keys/private_key.pem"

if [ ! -f "$PRIVATE_KEY_PATH" ]; then
    echo "❌ Private key file not found at: $PRIVATE_KEY_PATH"
    exit 1
fi

echo "Step 1: Extract public key from your private key..."
openssl rsa -in "$PRIVATE_KEY_PATH" -pubout -out /tmp/extracted_public.pem 2>/dev/null

if [ $? -ne 0 ]; then
    echo "❌ Failed to extract public key"
    exit 1
fi

echo "✅ Public key extracted"
echo ""

echo "Step 2: Your public key (from private key):"
echo "========================================="
cat /tmp/extracted_public.pem
echo "========================================="
echo ""

echo "Step 3: Compare with CloudFront"
echo ""
echo "From your screenshot, the public key in CloudFront starts with:"
echo "-----BEGIN PUBLIC KEY-----"
echo "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAx3dw4E3cFS6E0+cerpXW"
echo "..."
echo ""

echo "Your extracted public key:"
head -2 /tmp/extracted_public.pem | tail -1
echo ""

CLOUDFRONT_KEY_START="MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAx3dw4E3cFS6E0+cerpXW"
EXTRACTED_KEY_START=$(head -2 /tmp/extracted_public.pem | tail -1)

if [ "$CLOUDFRONT_KEY_START" = "$EXTRACTED_KEY_START" ]; then
    echo "✅ Keys MATCH! The public key in CloudFront is correct."
    echo ""
    echo "This means the issue is likely:"
    echo "1. CloudFront distribution is still deploying (check AWS console)"
    echo "2. Changes haven't propagated yet (wait 10-15 minutes)"
    echo "3. Browser/CDN cache needs to clear"
else
    echo "❌ Keys DON'T MATCH!"
    echo ""
    echo "The public key in CloudFront doesn't match your private key."
    echo "You need to:"
    echo "1. Delete the current public key in CloudFront"
    echo "2. Create a new one with the correct public key (shown above)"
    echo "3. Update the key group"
    echo "4. Wait for propagation"
fi

echo ""
echo "========================================="
echo "Next Steps"
echo "========================================="
echo ""
echo "1. Check CloudFront distribution status in AWS Console"
echo "   - Go to: https://console.aws.amazon.com/cloudfront/v4/home#/distributions"
echo "   - Look for distribution E35XJJN19W7KFF"
echo "   - Status should be 'Enabled' (not 'Deploying')"
echo ""
echo "2. If status is 'Deploying', wait until it's 'Enabled'"
echo ""
echo "3. After it's enabled, wait 5 more minutes for edge propagation"
echo ""
echo "4. Test again:"
echo "   curl https://api.streamshort.in/debug/cloudfront | jq -r '.signed_url' | xargs curl -I"
echo ""
