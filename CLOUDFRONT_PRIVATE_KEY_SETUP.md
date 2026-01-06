# CloudFront Private Key Setup for GitHub Actions

## Issue Fixed
The application was showing "CloudFront signing not configured - using mock URLs for manifests" because the `AWS_CLOUDFRONT_PRIVATE_KEY` environment variable was missing from the GitHub Actions deployment.

## Solution
Added `AWS_CLOUDFRONT_PRIVATE_KEY` to the GitHub Actions workflow and updated the secrets documentation.

## Required Action: Add GitHub Secret

You need to add the CloudFront private key as a GitHub secret:

### Steps:
1. Go to your GitHub repository: https://github.com/knight099/episodd
2. Click **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Name: `AWS_CLOUDFRONT_PRIVATE_KEY`
5. Value: Copy the private key content below (including BEGIN/END lines)

### Private Key Content:
```
-----BEGIN PRIVATE KEY-----

-----END PRIVATE KEY-----
```

## What This Fixes
- CloudFront signed URLs will work properly for HLS manifest streaming
- The application will no longer show "CloudFront signing not configured" warning
- Secure video streaming with proper access control

## Next Steps
1. Add the GitHub secret as described above
2. Push any change to trigger a new deployment
3. The CloudFront signing should work in the deployed application

## Files Modified
- `.github/workflows/deploy.yml` - Added `AWS_CLOUDFRONT_PRIVATE_KEY` environment variable
- `GITHUB_SECRETS_SETUP.md` - Updated to include the new secret requirement