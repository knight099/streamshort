# Thumbnail Upload - Quick Start

## Three-Step Process

### 1️⃣ Request Upload URL

```bash
POST /api/content/upload-url
Authorization: Bearer {token}

{
  "filename": "thumbnail.jpg",
  "content_type": "image/jpeg",
  "size_bytes": 245760,
  "metadata": {
    "series_id": "{your_series_id}",
    "type": "thumbnail"
  }
}
```

**Response:**
```json
{
  "upload_id": "abc-123",
  "presigned_url": "https://s3.amazonaws.com/...",
  "expires_in": 3600
}
```

---

### 2️⃣ Upload to S3

```javascript
await fetch(presignedUrl, {
  method: 'PUT',
  headers: { 'Content-Type': 'image/jpeg' },
  body: imageFile
});
```

---

### 3️⃣ Notify Complete

```bash
POST /api/content/uploads/{upload_id}/notify
Authorization: Bearer {token}

{
  "s3_path": "s3://bucket/thumbnails/{upload_id}/thumbnail.jpg",
  "size_bytes": 245760
}
```

---

## Validation Rules

- ✅ File type: `image/*` only
- ✅ Max size: **5 MB**
- ✅ Must be series owner
- ✅ `metadata.series_id` required

---

## Complete React Example

```typescript
async function uploadThumbnail(file: File, seriesId: string, token: string) {
  // Validate
  if (!file.type.startsWith('image/')) throw new Error('Must be an image');
  if (file.size > 5 * 1024 * 1024) throw new Error('Max size 5MB');

  // Step 1: Get upload URL
  const step1 = await fetch('/api/content/upload-url', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      filename: file.name,
      content_type: file.type,
      size_bytes: file.size,
      metadata: { series_id: seriesId, type: 'thumbnail' }
    }),
  });
  const { upload_id, presigned_url } = await step1.json();

  // Step 2: Upload to S3
  await fetch(presigned_url, {
    method: 'PUT',
    headers: { 'Content-Type': file.type },
    body: file,
  });

  // Step 3: Notify complete
  await fetch(`/api/content/uploads/${upload_id}/notify`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      s3_path: `s3://bucket/thumbnails/${upload_id}/${file.name}`,
      size_bytes: file.size,
    }),
  });

  // Done! Series thumbnail_url is now updated
}
```

---

## Common Errors

| Error | Reason | Fix |
|-------|--------|-----|
| 400 "Thumbnail must be an image" | Wrong file type | Use image/* |
| 400 "Size must not exceed 5MB" | File too large | Compress image |
| 403 "Must be onboarded as creator" | Not a creator | Onboard as creator |
| 404 "Series not found" | Wrong series_id | Check series ID |

---

For complete documentation, see [THUMBNAIL_UPLOAD_API.md](file:///Users/vaibhaw/Developer/streamshort/THUMBNAIL_UPLOAD_API.md)
