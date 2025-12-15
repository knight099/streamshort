# 🚨 URGENT: Thumbnail Upload Fix Required

## Problems Identified

### 1. ❌ Missing Database Migration
The `file_type` column doesn't exist in production database yet.

**Error:** `Database error` (500) when trying to query `upload_requests.file_type`

### 2. ❌ Missing `series_id` in Frontend Request
Frontend is sending:
```json
"metadata": {"type": "thumbnail"}
```

Should be:
```json
"metadata": {
  "type": "thumbnail",
  "series_id": "{series_id_here}"
}
```

---

## Immediate Fixes Required

### Fix 1: Run Database Migration

**On Production Server:**
```bash
# SSH into your server
ssh your-production-server

# Navigate to project directory
cd /path/to/streamshort

# Run the migration
psql $DATABASE_URL -f migrations/031_add_file_type_to_upload_requests.sql
```

**OR manually execute:**
```sql
-- Add file_type column
ALTER TABLE upload_requests
ADD COLUMN IF NOT EXISTS file_type VARCHAR(20) DEFAULT 'video';

-- Add check constraint
ALTER TABLE upload_requests
ADD CONSTRAINT IF NOT EXISTS check_file_type 
CHECK (file_type IN ('video', 'thumbnail', 'caption'));

-- Update existing records
UPDATE upload_requests
SET file_type = 'video'
WHERE file_type IS NULL;
```

---

### Fix 2: Update Frontend Code

**Flutter - Update Upload Request:**

```dart
// BEFORE (❌ Wrong)
final metadata = {
  'type': 'thumbnail',
};

// AFTER (✅ Correct)
final metadata = {
  'type': 'thumbnail',
  'series_id': seriesId,  // ADD THIS!
};
```

**Complete Flutter Example:**
```dart
Future<void> uploadThumbnail(File imageFile, String seriesId) async {
  // Step 1: Request upload URL
  final uploadUrlResponse = await dio.post(
    '/api/content/upload-url',
    data: {
      'filename': imageFile.path.split('/').last,
      'content_type': 'image/jpeg',
      'size_bytes': await imageFile.length(),
      'metadata': {
        'type': 'thumbnail',
        'series_id': seriesId,  // ← CRITICAL: Include series_id
      },
    },
  );

  final uploadId = uploadUrlResponse.data['upload_id'];
  final presignedUrl = uploadUrlResponse.data['presigned_url'];

  // Step 2: Upload to S3
  await dio.put(
    presignedUrl,
    data: imageFile.openRead(),
    options: Options(
      headers: {'Content-Type': 'image/jpeg'},
    ),
  );

  // Step 3: Notify complete
  await dio.post(
    '/api/content/uploads/$uploadId/notify',
    data: {
      's3_path': 's3://streamshort-media/thumbnails/$uploadId/${imageFile.path.split('/').last}',
      'size_bytes': await imageFile.length(),
    },
  );
}
```

---

## Verification Steps

After applying both fixes:

1. ✅ Database migration applied
2. ✅ Frontend includes `series_id` in metadata
3. ✅ Test upload again
4. ✅ Check series record has `thumbnail_url` updated

---

## Quick Test

```bash
# Test the migration was applied
psql $DATABASE_URL -c "SELECT column_name, data_type FROM information_schema.columns WHERE table_name='upload_requests' AND column_name='file_type';"

# Should return:
# column_name | data_type
# file_type   | character varying
```
