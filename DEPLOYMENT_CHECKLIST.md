# Deployment Checklist - Thumbnail Upload Feature

## The Issue
You're getting "Database error" because the production server doesn't have the updated code or database schema.

## What You Need to Deploy

### 1. Database Migration
The production database needs the `file_type` column added.

**On your production server (Render.com or wherever):**

```bash
# Connect to your database
psql $DATABASE_URL

# Run this SQL
ALTER TABLE upload_requests
ADD COLUMN IF NOT EXISTS file_type VARCHAR(20) DEFAULT 'video';

ALTER TABLE upload_requests
ADD CONSTRAINT check_file_type CHECK (file_type IN ('video', 'thumbnail', 'caption'));

UPDATE upload_requests SET file_type = 'video' WHERE file_type IS NULL;

# Verify it worked
\d upload_requests
# Should show file_type column
```

**OR** if your deployment runs migrations automatically:
- Just make sure `migrations/031_add_file_type_to_upload_requests.sql` is in your repo
- The migration will run on next deployment

---

### 2. Deploy Updated Code

You need to deploy the updated Go code to production.

**Updated files:**
- `handlers/content.go` - Enhanced upload handlers
- `services/aws.go` - Type-specific upload methods  
- `models/content.go` - FileType field added
- `migrations/031_add_file_type_to_upload_requests.sql` - Database migration

**Deploy to Render.com:**
```bash
# Commit the changes
git add .
git commit -m "Add thumbnail upload support"
git push origin main

# Render will automatically deploy
# Wait for deployment to complete
```

---

### 3. Verify Deployment

After deployment:

```bash
# Check the server logs for migration message
# Should see: "Applied SQL migration: 031_add_file_type_to_upload_requests"

# Test the API
curl https://api.episodd.com/health
```

---

## Quick Fix (If Urgent)

If you need it working NOW and can't wait for full deployment:

1. **Just run the SQL migration manually** on production database
2. **Restart your production server** (on Render: Manual Deploy > Clear build cache & deploy)

The updated code is backward compatible, so even without deployment, adding the column will fix the immediate error. However, you should still deploy the code updates for the feature to work properly.

---

## For Local Testing

If you want to test locally first:

```bash
# 1. Run migration on local database
psql postgres://postgres:password@localhost:5432/episodd -f migrations/031_add_file_type_to_upload_requests.sql

# 2. Restart your local server
# (Already doing this with the build)

# 3. Test with Flutter app pointing to localhost
```

---

## Current Status

✅ Code updated locally
✅ Migration file created  
❌ **Production database migration NOT applied**
❌ **Production code NOT deployed**

**Next step:** Deploy to production!
