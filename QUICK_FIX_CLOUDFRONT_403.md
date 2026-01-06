# Quick Fix: CloudFront 403 Error

## TL;DR

Your video is getting 403 because:
1. Video hasn't been transcoded to HLS yet
2. Backend is serving raw MP4 from `uploads/` path
3. CloudFront `uploads/*` behavior isn't configured for signed URLs

## Quick Test (After Deploying)

```bash
# Test if CloudFront signing works
curl https://api.episodd.com/debug/cloudfront | jq

# If you get a signed_url, test it:
curl -I "PASTE_SIGNED_URL_HERE"
```

**Results**:
- 200/404 = ✅ Signing works
- 403 = ❌ CloudFront config issue

## Quick Fix Options

### Option 1: Wait for Transcoding (Easiest)
Just wait. Once the video is transcoded:
- `hls_manifest_url` will be populated
- Video will be served from `transcoded/` path
- Should work if `transcoded/*` behavior is configured

### Option 2: Fix CloudFront Config (Permanent)
AWS Console → CloudFront → Distribution `E35XJJN19W7KFF`:

1. **Check Behaviors tab**:
   - Look for `transcoded/*` behavior
   - "Restrict viewer access" should be "Yes"
   - "Trusted key groups" should include a key group

2. **Check Key Management**:
   - CloudFront → Key management → Public keys
   - Find key ID `K22WDLQZV9U4XV`
   - Check which key groups contain it
   - Verify those key groups are in the behaviors

3. **If missing**: Create behavior for `transcoded/*`:
   - Path pattern: `transcoded/*`
   - Restrict viewer access: Yes
   - Trusted key groups: Create/select one with key `K22WDLQZV9U4XV`

## Deploy & Test

```bash
# Deploy changes
git add .
git commit -m "Add CloudFront debugging"
git push

# Wait for deployment, then test
curl https://api.episodd.com/debug/cloudfront

# Or run the test script
./test_cloudfront_signing.sh
```

## Check Transcoding Status

```bash
# Check if episode has HLS manifest
# (Replace with your episode ID)
curl -H "Authorization: Bearer YOUR_TOKEN" \
  https://api.episodd.com/api/episodes/386db34f-e43f-42d3-ab83-66ad5e766c0a/manifest
```

If response has `manifest_url` with `transcoded/` in the path → transcoding is done!

## Still Not Working?

Read the detailed guides:
- `CLOUDFRONT_403_TROUBLESHOOTING.md` - Full troubleshooting steps
- `CLOUDFRONT_FIX_SUMMARY.md` - What was changed and why

## Most Likely Issue

**Key-Pair-Id not in trusted key group**:
1. Go to CloudFront → Key management → Key groups
2. Create a key group with public key `K22WDLQZV9U4XV`
3. Assign this key group to `transcoded/*` behavior
4. Wait 5-10 minutes for CloudFront to propagate
5. Test again
