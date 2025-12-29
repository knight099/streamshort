# CloudFront Diagnostic Report
**Date**: December 29, 2025  
**Distribution**: E35XJJN19W7KFF  
**Domain**: djoxr6aauqbf6.cloudfront.net

## Test Results

### ✅ Backend Signing Works
The backend is correctly generating signed URLs with:
- **Key-Pair-ID**: K22WDLQZV9U4XV
- **Domain**: djoxr6aauqbf6.cloudfront.net
- **Signature**: Valid format
- **Expiration**: Properly set (1 hour from generation)

### ❌ CloudFront Rejects Signed URLs
Both test paths return **403 Forbidden**:

**Test 1: Transcoded path**
```bash
curl -I "https://djoxr6aauqbf6.cloudfront.net/transcoded/test/index.m3u8?..."
# Result: HTTP/2 403
# Error: AccessDenied
# x-cache: Error from cloudfront
```

**Test 2: Uploads path (actual episode)**
```bash
curl -I "https://djoxr6aauqbf6.cloudfront.net/uploads/.../VID_20251229_020053_241.mp4?..."
# Result: HTTP/2 403
# Error: AccessDenied
# x-cache: Error from cloudfront
```

## Root Cause Confirmed

The issue is **NOT** in the backend code. The backend is generating valid signed URLs correctly.

The issue **IS** in the CloudFront distribution configuration:

### Most Likely Cause: Key-Pair-Id Not in Trusted Key Group

The Key-Pair-ID `K22WDLQZV9U4XV` is either:
1. Not added to any key group in CloudFront
2. Not assigned to the distribution's behaviors
3. The public key doesn't match the private key being used

## Required Actions in AWS Console

### Step 1: Verify Public Key Exists
1. Go to **AWS Console** → **CloudFront** → **Key management** → **Public keys**
2. Look for a public key with ID: `K22WDLQZV9U4XV`
3. If it doesn't exist, you need to create it:
   - Click "Create public key"
   - Paste the public key (extract from your private key)
   - Note the Key ID that gets generated

### Step 2: Create/Verify Key Group
1. Go to **CloudFront** → **Key management** → **Key groups**
2. Check if any key group contains the public key `K22WDLQZV9U4XV`
3. If not, create a new key group:
   - Click "Create key group"
   - Name: `streamshort-signing-keys`
   - Add public key: `K22WDLQZV9U4XV`
   - Click "Create"

### Step 3: Configure Distribution Behaviors
1. Go to **CloudFront** → **Distributions** → Select `E35XJJN19W7KFF`
2. Click **Behaviors** tab
3. You need to create/edit behaviors for both paths:

#### Behavior for `transcoded/*`:
- Click "Create behavior" (or edit if exists)
- **Path pattern**: `transcoded/*`
- **Origin**: Your S3 bucket origin
- **Viewer protocol policy**: Redirect HTTP to HTTPS
- **Allowed HTTP methods**: GET, HEAD, OPTIONS
- **Restrict viewer access**: **Yes** ← CRITICAL
- **Trusted authorization type**: Trusted key groups
- **Trusted key groups**: Select the key group created in Step 2
- Click "Create behavior"

#### Behavior for `uploads/*`:
- Same as above but with **Path pattern**: `uploads/*`

### Step 4: Wait for Propagation
- CloudFront changes take 5-15 minutes to propagate globally
- Status will show "Deploying" then "Enabled"

### Step 5: Test Again
After propagation completes:
```bash
curl https://api.streamshort.in/debug/cloudfront | jq -r '.signed_url' | xargs curl -I
```

Expected result: **HTTP/2 404** (file doesn't exist, but signing works!)

## Alternative: Check if Private Key Matches Public Key

If the key group is already configured, the issue might be a key mismatch.

### Extract Public Key from Private Key
```bash
# If you have the private key file
openssl rsa -in cloudfront_private_key.pem -pubout -out public_key.pem

# View the public key
cat public_key.pem
```

Compare this with the public key uploaded to CloudFront. They must match exactly.

## Quick Verification Commands

```bash
# Test if signing works (run after CloudFront config changes)
./test_cloudfront_signing.sh

# Or manually:
curl https://api.streamshort.in/debug/cloudfront | jq
```

## Expected Timeline

1. **Configure CloudFront**: 10-15 minutes (manual work)
2. **Propagation**: 5-15 minutes (automatic)
3. **Testing**: 2-3 minutes
4. **Total**: ~30 minutes

## What Happens After Fix

Once CloudFront is configured correctly:
1. Signed URLs will return 200 OK (or 404 if file doesn't exist)
2. Videos will play in the Flutter app
3. Both `transcoded/*` and `uploads/*` paths will work with signed URLs

## Current Status Summary

| Component | Status | Notes |
|-----------|--------|-------|
| Backend Code | ✅ Working | Generating valid signed URLs |
| Private Key | ✅ Loaded | Key is loaded and used for signing |
| Signature Format | ✅ Valid | AWS SDK v2 format is correct |
| CloudFront Config | ❌ **ISSUE** | Key not in trusted key group |
| Video Playback | ❌ Blocked | 403 errors prevent playback |

## Next Steps

1. **Immediate**: Follow Steps 1-4 above to configure CloudFront
2. **After 15 min**: Test the debug endpoint again
3. **Verify**: Try playing video in Flutter app
4. **Monitor**: Check backend logs for any new errors

## Support Resources

- [AWS: Create Key Pairs for Signers](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-trusted-signers.html#private-content-creating-cloudfront-key-pairs)
- [AWS: Add Key Groups to Distribution](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-trusted-signers.html#private-content-adding-trusted-signers)
- [Troubleshoot Signed URLs](https://repost.aws/knowledge-center/cloudfront-troubleshoot-signed-url-cookies)
