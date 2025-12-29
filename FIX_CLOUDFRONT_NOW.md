# Fix CloudFront 403 Error - Action Required

## Problem Confirmed
✅ Backend is generating signed URLs correctly  
❌ CloudFront is rejecting them with 403 Forbidden  
**Root Cause**: Key-Pair-ID `K22WDLQZV9U4XV` is not in CloudFront's trusted key group

## Your Public Key
```
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAx3dw4E3cFS6E0+cerpXW
qnHFqaBI8hCzUzEBr/K1rgRuoWMim/NyP6XC+eaKOMv+oh/rpjc05IMtqQ8Lfp9n
fV5NO3jIhiPW7Ld+xYYmCZLQl2ghFS7C42zZ0hYuLMEi5FBBZu0Ff/GhpvgdVl5a
fNpL+woewzLeT5/j9SAC2v2rvdFr8ffnocuIT6AI9eS52ACTHQLPEOC1iQFGMJsi
+fD/0G4SSMmCEec66C2FlGXD2CnOTnVBO5fwibWK1FMRt9RvliJonWOQnYL2drlg
kgLe6XipKAqnqD5kt3zwBWTKHsTsan+9SN8heHfC1TbQJlBsyIqwbqCiFbP/+HvV
SwIDAQAB
-----END PUBLIC KEY-----
```

## Step-by-Step Fix (15 minutes)

### Step 1: Check if Public Key Exists (2 min)

1. Go to: https://console.aws.amazon.com/cloudfront/v4/home#/publickey/home
2. Look for a public key with ID: `K22WDLQZV9U4XV`

**If it EXISTS**: Go to Step 2  
**If it DOESN'T EXIST**: Continue below

#### Create the Public Key:
1. Click **"Create public key"**
2. **Name**: `streamshort-signing-key`
3. **Key**: Paste the public key from above (including BEGIN/END lines)
4. Click **"Create"**
5. **IMPORTANT**: Note the Key ID that gets generated
   - If it's different from `K22WDLQZV9U4XV`, you'll need to update your environment variables

### Step 2: Create Key Group (3 min)

1. Go to: https://console.aws.amazon.com/cloudfront/v4/home#/keygrouplist
2. Click **"Create key group"**
3. **Name**: `streamshort-trusted-signers`
4. **Public keys**: Select the key `K22WDLQZV9U4XV` (or the one you just created)
5. Click **"Create key group"**
6. Note the Key Group ID (e.g., `K2XXXXXXXXXXXX`)

### Step 3: Configure Distribution Behaviors (10 min)

1. Go to: https://console.aws.amazon.com/cloudfront/v4/home#/distributions
2. Click on distribution ID: `E35XJJN19W7KFF`
3. Click the **"Behaviors"** tab

#### Create Behavior for `transcoded/*`:

4. Click **"Create behavior"**
5. Fill in:
   - **Path pattern**: `transcoded/*`
   - **Origin**: Select your S3 bucket origin (streamshort-media)
   - **Viewer protocol policy**: Redirect HTTP to HTTPS
   - **Allowed HTTP methods**: GET, HEAD, OPTIONS
   - **Cache policy**: CachingOptimized (or your preferred policy)
   - **Restrict viewer access**: **Yes** ← CRITICAL!
   - **Trusted authorization type**: Trusted key groups
   - **Trusted key groups**: Select `streamshort-trusted-signers` (the one you created in Step 2)
6. Click **"Create behavior"**

#### Create Behavior for `uploads/*`:

7. Click **"Create behavior"** again
8. Fill in (same as above but different path):
   - **Path pattern**: `uploads/*`
   - **Origin**: Select your S3 bucket origin
   - **Viewer protocol policy**: Redirect HTTP to HTTPS
   - **Allowed HTTP methods**: GET, HEAD, OPTIONS
   - **Cache policy**: CachingOptimized
   - **Restrict viewer access**: **Yes** ← CRITICAL!
   - **Trusted authorization type**: Trusted key groups
   - **Trusted key groups**: Select `streamshort-trusted-signers`
9. Click **"Create behavior"**

#### Verify Default Behavior:

10. Check if there's a **Default (*)** behavior
11. If it exists and has "Restrict viewer access" set to Yes, make sure it also uses the same key group
12. If you want public access to other paths, leave Default behavior unrestricted

### Step 4: Wait for Propagation (5-15 min)

1. The distribution status will show **"Deploying"**
2. Wait until it changes to **"Enabled"**
3. This usually takes 5-15 minutes

### Step 5: Test (2 min)

After propagation completes:

```bash
# Test the debug endpoint
curl https://api.streamshort.in/debug/cloudfront | jq

# Get the signed URL and test it
curl https://api.streamshort.in/debug/cloudfront | jq -r '.signed_url' | xargs curl -I
```

**Expected result**: 
- `HTTP/2 404` (file doesn't exist, but signing works!) ✅
- OR `HTTP/2 200` (if the test file exists) ✅

**If still 403**: 
- Wait another 5 minutes for propagation
- Verify the key group is correctly assigned to behaviors
- Check that the Key ID matches exactly

### Step 6: Test Video Playback

1. Open your Flutter app
2. Try playing the video
3. It should work now! 🎉

## Troubleshooting

### Issue: Key ID doesn't match
If the generated Key ID is different from `K22WDLQZV9U4XV`:

1. Update your environment variables in Render:
   - `AWS_CLOUDFRONT_KEY_PAIR_ID` = new Key ID
2. Redeploy your backend
3. Test again

### Issue: Still getting 403 after 15 minutes
1. Go back to CloudFront → Distributions → `E35XJJN19W7KFF` → Behaviors
2. Click on `transcoded/*` behavior → Edit
3. Verify "Restrict viewer access" is **Yes**
4. Verify "Trusted key groups" includes your key group
5. Save and wait another 10 minutes

### Issue: Behavior already exists
If behaviors for `transcoded/*` or `uploads/*` already exist:
1. Click on the behavior → Edit
2. Update the settings as described in Step 3
3. Make sure "Restrict viewer access" is Yes
4. Make sure the correct key group is selected
5. Save changes

## Quick Verification Checklist

- [ ] Public key `K22WDLQZV9U4XV` exists in CloudFront
- [ ] Key group created with this public key
- [ ] `transcoded/*` behavior created with "Restrict viewer access" = Yes
- [ ] `transcoded/*` behavior has the key group assigned
- [ ] `uploads/*` behavior created with "Restrict viewer access" = Yes
- [ ] `uploads/*` behavior has the key group assigned
- [ ] Distribution status is "Enabled" (not "Deploying")
- [ ] Waited at least 10 minutes after saving changes
- [ ] Debug endpoint returns signed URL
- [ ] Signed URL returns 200 or 404 (not 403)

## After Fix

Once this is working:
1. Videos will play in your Flutter app
2. Signed URLs will be valid for 1 hour
3. CloudFront will serve content securely
4. Both transcoded HLS and raw MP4 files will work

## Need Help?

If you're stuck:
1. Take a screenshot of the CloudFront Behaviors tab
2. Run: `curl https://api.streamshort.in/debug/cloudfront | jq`
3. Share the output and screenshot

## Estimated Time
- **Configuration**: 15 minutes
- **Propagation**: 5-15 minutes
- **Total**: ~30 minutes
