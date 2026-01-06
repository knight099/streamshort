# CloudFront 403 Error - Fix Summary

## What Was Done

### 1. Added Debug Logging
- Enhanced `services/aws.go` to log CloudFront signing details
- Added logging to `handlers/content.go` for manifest URL generation
- Logs now show: Key-Pair-ID, domain, resource path, expiration time

### 2. Created Debug Endpoint
- New endpoint: `GET /debug/cloudfront`
- Tests CloudFront signing without authentication
- Returns signed URL and configuration details
- Usage: `curl https://api.episodd.com/debug/cloudfront`

### 3. Created Test Script
- `test_cloudfront_signing.sh` - Automated testing script
- Tests CloudFront configuration
- Verifies signed URLs work
- Provides diagnostic information

### 4. Created Troubleshooting Guide
- `CLOUDFRONT_403_TROUBLESHOOTING.md` - Comprehensive guide
- Step-by-step verification process
- Common issues and solutions
- CloudFront configuration requirements

## Root Cause

The 403 error is happening because:

1. **Episode Not Transcoded**: The video hasn't been transcoded to HLS format yet
   - `hls_manifest_url` field is NULL
   - Backend falls back to serving raw MP4 from `uploads/` path

2. **CloudFront Behavior Not Configured**: The `uploads/*` path behavior likely isn't set up for signed URLs
   - Only `transcoded/*` path may be configured
   - Raw MP4 files in `uploads/` can't be accessed with signed URLs

## Immediate Actions Required

### Option A: Wait for Transcoding (Recommended)
1. Wait for the video to be transcoded to HLS format
2. Once `hls_manifest_url` is populated, video will be served from `transcoded/` path
3. If `transcoded/*` behavior is configured correctly, video will play

### Option B: Configure CloudFront for Both Paths
1. Go to AWS Console → CloudFront → Distribution `E35XJJN19W7KFF`
2. Create/verify behavior for `transcoded/*`:
   - Restrict viewer access: Yes
   - Trusted key groups: Include key with ID `K22WDLQZV9U4XV`
3. Create behavior for `uploads/*` (if you want to serve raw MP4):
   - Restrict viewer access: Yes
   - Trusted key groups: Same as above

## Testing Steps

### 1. Test CloudFront Configuration
```bash
# Run the test script
./test_cloudfront_signing.sh

# Or manually test the debug endpoint
curl https://api.episodd.com/debug/cloudfront | jq
```

### 2. Test Signed URL Directly
```bash
# Get a signed URL from the debug endpoint
SIGNED_URL=$(curl -s https://api.episodd.com/debug/cloudfront | jq -r '.signed_url')

# Test it
curl -I "$SIGNED_URL"
```

Expected results:
- **200 OK**: Configuration is correct ✅
- **404 Not Found**: Signing works, file doesn't exist (expected for test) ✅
- **403 Forbidden**: Configuration issue ❌

### 3. Check Backend Logs
After deploying these changes, check your backend logs for:
```
[CloudFront Debug] KeyPairID: K22WDLQZV9U4XV, Domain: djoxr6aauqbf6.cloudfront.net
[CloudFront Debug] Resource path: uploads/...
[Manifest] Episode has no HLS manifest yet, falling back to raw MP4
[Manifest] WARNING: Raw MP4 playback may not work if CloudFront 'uploads/*' behavior is not configured
```

## CloudFront Configuration Checklist

- [ ] Public key `K22WDLQZV9U4XV` exists in CloudFront → Key Management
- [ ] Key is added to a key group
- [ ] Key group is assigned to `transcoded/*` behavior
- [ ] `transcoded/*` behavior has "Restrict viewer access" enabled
- [ ] Private key in GitHub secrets matches public key in CloudFront
- [ ] (Optional) `uploads/*` behavior configured if serving raw MP4s

## Files Modified

1. **services/aws.go**
   - Added debug logging to `GenerateCloudFrontSignedURL()`
   - Added `GetCloudFrontKeyPairID()` method

2. **main.go**
   - Added `/debug/cloudfront` endpoint for testing

3. **handlers/content.go**
   - Enhanced logging for manifest URL generation
   - Better error messages

## Files Created

1. **CLOUDFRONT_403_TROUBLESHOOTING.md** - Detailed troubleshooting guide
2. **CLOUDFRONT_FIX_SUMMARY.md** - This file
3. **test_cloudfront_signing.sh** - Automated test script

## Next Steps

1. **Deploy the changes**:
   ```bash
   git add .
   git commit -m "Add CloudFront debugging and troubleshooting"
   git push
   ```

2. **Test the debug endpoint** once deployed:
   ```bash
   curl https://api.episodd.com/debug/cloudfront
   ```

3. **Verify CloudFront configuration** in AWS Console

4. **Check if transcoding is complete** for the episode:
   - Query the database for `hls_manifest_url` field
   - Or check backend logs when accessing the episode

5. **Test video playback** in Flutter app after transcoding completes

## Support

If issues persist after following the troubleshooting guide:

1. Check CloudFront distribution settings match the requirements
2. Verify the private key is correct (regenerate if needed)
3. Ensure transcoding pipeline is working
4. Review backend logs for detailed error messages

## References

- [AWS CloudFront Signed URLs](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-signed-urls.html)
- [Troubleshoot 403 Errors](https://repost.aws/knowledge-center/cloudfront-troubleshoot-signed-url-cookies)
- [CloudFront Key Groups](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-trusted-signers.html)
