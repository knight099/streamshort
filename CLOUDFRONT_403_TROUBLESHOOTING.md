# CloudFront 403 Error Troubleshooting Guide

## Current Issue
The Flutter app is receiving 403 errors when trying to play videos through CloudFront signed URLs.

## Root Cause Analysis

From the logs, the signed URL being generated is:
```
https://djoxr6aauqbf6.cloudfront.net/uploads/edab3bdd-423b-4f59-b244-2019eadeacf8/VID_20251229_020053_241.mp4
```

**Key Finding**: The video is being served from `uploads/` path (raw MP4) instead of `transcoded/` path (HLS manifest) because:
1. The episode's `hls_manifest_url` field is NULL (transcoding not complete)
2. Backend falls back to serving the raw MP4 file with signed URL
3. CloudFront behavior for `uploads/*` path is likely not configured for signed URLs

## CloudFront Configuration Requirements

### Required Behaviors

Your CloudFront distribution (`E35XJJN19W7KFF`) needs TWO separate behaviors:

#### 1. Behavior for `transcoded/*` (HLS Streaming)
- **Path Pattern**: `transcoded/*`
- **Restrict Viewer Access**: Yes
- **Trusted Key Groups**: Must include key group with public key ID `K22WDLQZV9U4XV`
- **Origin**: S3 bucket `streamshort-media`
- **Viewer Protocol Policy**: Redirect HTTP to HTTPS
- **Allowed HTTP Methods**: GET, HEAD, OPTIONS
- **Cache Policy**: CachingOptimized or custom with appropriate TTL

#### 2. Behavior for `uploads/*` (Raw Files - Optional)
- **Path Pattern**: `uploads/*`
- **Restrict Viewer Access**: Yes (if you want signed URLs) or No (if private via S3)
- **Trusted Key Groups**: Same as above if using signed URLs
- **Origin**: S3 bucket with OAI (Origin Access Identity) to keep S3 private

### Key Pair Configuration

1. **Public Key**: Must be uploaded to CloudFront → Key Management → Public Keys
2. **Key ID**: Should be `K22WDLQZV9U4XV` (already configured)
3. **Key Group**: Create a key group containing the public key
4. **Assign to Behaviors**: Both `transcoded/*` and `uploads/*` behaviors must use this key group

## Verification Steps

### 1. Test the Debug Endpoint

```bash
# Test CloudFront signing configuration
curl https://api.streamshort.in/debug/cloudfront

# Test with specific path
curl "https://api.streamshort.in/debug/cloudfront?path=transcoded/test/index.m3u8"
```

Expected response:
```json
{
  "status": "ok",
  "key_pair_id": "K22WDLQZV9U4XV",
  "domain": "djoxr6aauqbf6.cloudfront.net",
  "test_path": "transcoded/test/index.m3u8",
  "signed_url": "https://...",
  "expires_at": "2025-12-29T12:00:00Z",
  "expires_unix": 1767012000
}
```

### 2. Test the Signed URL Directly

```bash
# Copy the signed_url from above and test it
curl -v "PASTE_SIGNED_URL_HERE"
```

**Expected Results**:
- **200 OK**: CloudFront configuration is correct
- **403 Forbidden**: Key pair not in trusted key group or signature mismatch
- **404 Not Found**: File doesn't exist (expected for test path)

### 3. Check CloudFront Distribution

In AWS Console → CloudFront → Distribution `E35XJJN19W7KFF`:

1. **Behaviors Tab**:
   - Verify `transcoded/*` behavior exists
   - Check "Restrict viewer access" is set to "Yes"
   - Verify "Trusted key groups" includes a key group

2. **Key Management**:
   - Go to CloudFront → Key management → Public keys
   - Verify public key with ID `K22WDLQZV9U4XV` exists
   - Check which key groups contain this key
   - Verify those key groups are assigned to the behaviors

### 4. Verify Private Key Matches Public Key

The private key in your GitHub secret `AWS_CLOUDFRONT_PRIVATE_KEY` must match the public key uploaded to CloudFront.

To verify:
```bash
# Extract public key from private key
openssl rsa -in private_key.pem -pubout -out public_key.pem

# Compare with the public key in CloudFront console
```

## Common Issues and Solutions

### Issue 1: "Missing Key-Pair-Id" Error
**Cause**: Signed URL doesn't include Key-Pair-Id parameter
**Solution**: Check that `AWS_CLOUDFRONT_KEY_PAIR_ID` environment variable is set correctly

### Issue 2: "Unknown Key" Error
**Cause**: Key-Pair-Id not in CloudFront's trusted key group
**Solution**: 
1. Go to CloudFront → Key management → Key groups
2. Create or edit a key group to include public key `K22WDLQZV9U4XV`
3. Assign this key group to the distribution's behaviors

### Issue 3: Consistent 403 on Valid URLs
**Cause**: Private key doesn't match public key in CloudFront
**Solution**: 
1. Regenerate the key pair
2. Upload new public key to CloudFront
3. Update `AWS_CLOUDFRONT_PRIVATE_KEY` secret with new private key
4. Redeploy the application

### Issue 4: 403 Only on `uploads/*` Path
**Cause**: `uploads/*` behavior not configured for signed URLs
**Solution**: 
1. Create a behavior for `uploads/*` path pattern
2. Enable "Restrict viewer access"
3. Assign the same trusted key group
4. **OR** Keep uploads private and only serve transcoded HLS content

## Recommended Solution

### Short-term Fix: Wait for Transcoding
The video needs to be transcoded to HLS format. Once transcoding completes:
1. The `hls_manifest_url` field will be populated
2. Backend will serve from `transcoded/` path instead of `uploads/`
3. If `transcoded/*` behavior is configured correctly, video will play

### Long-term Fix: Configure Both Behaviors
1. Set up `transcoded/*` behavior with signed URLs (for HLS streaming)
2. Keep `uploads/*` private (no signed URLs needed - these are source files)
3. Only serve transcoded content to end users

## Backend Improvements

The backend code has been updated with:
1. Debug logging for CloudFront URL generation
2. Debug endpoint at `/debug/cloudfront` for testing
3. Better error messages

## Testing Checklist

- [ ] Debug endpoint returns valid signed URL
- [ ] Signed URL works when tested with curl
- [ ] CloudFront behavior for `transcoded/*` has "Restrict viewer access" enabled
- [ ] Key group contains public key `K22WDLQZV9U4XV`
- [ ] Key group is assigned to `transcoded/*` behavior
- [ ] Private key in GitHub secrets matches public key in CloudFront
- [ ] Episode has `hls_manifest_url` populated (transcoding complete)
- [ ] Video plays in Flutter app

## Next Steps

1. **Immediate**: Test the debug endpoint to verify signing works
2. **Check**: Verify CloudFront behaviors are configured correctly
3. **Wait**: Allow transcoding to complete for the episode
4. **Test**: Try playing the video again once HLS manifest exists
5. **Monitor**: Check backend logs for CloudFront debug messages

## Support Resources

- [AWS CloudFront Signed URLs Documentation](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-signed-urls.html)
- [Troubleshoot CloudFront 403 Errors](https://repost.aws/knowledge-center/cloudfront-troubleshoot-signed-url-cookies)
- [CloudFront Key Groups](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-trusted-signers.html)
