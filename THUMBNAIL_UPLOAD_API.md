# Thumbnail Upload API - Frontend Integration Guide

## Overview

This guide explains how to implement series thumbnail uploads in the frontend. The upload process uses a three-step flow with AWS S3 presigned URLs.

---

## Upload Flow

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Select    │─────▶│   Request   │─────▶│   Upload    │─────▶│   Notify    │
│   Image     │      │  Upload URL │      │   to S3     │      │  Complete   │
└─────────────┘      └─────────────┘      └─────────────┘      └─────────────┘
   Frontend            Backend API          AWS S3              Backend API
                                                                        │
                                                                        ▼
                                                                ┌─────────────┐
                                                                │   Series    │
                                                                │  Updated    │
                                                                └─────────────┘
```

---

## Step 1: Request Upload URL

### Endpoint
```
POST /api/content/upload-url
```

### Headers
```
Authorization: Bearer {access_token}
Content-Type: application/json
```

### Request Body
```json
{
  "filename": "series-thumbnail.jpg",
  "content_type": "image/jpeg",
  "size_bytes": 245760,
  "metadata": {
    "series_id": "abc123-456def-789ghi",
    "type": "thumbnail"
  }
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `filename` | string | Yes | Name of the file to upload |
| `content_type` | string | Yes | MIME type (must be `image/*`) |
| `size_bytes` | number | Yes | File size in bytes (max 5MB for thumbnails) |
| `metadata.series_id` | string | Yes | UUID of the series |
| `metadata.type` | string | Yes | Must be `"thumbnail"` |

### Response
```json
{
  "upload_id": "550e8400-e29b-41d4-a716-446655440000",
  "presigned_url": "https://bucket.s3.amazonaws.com/thumbnails/550e8400.../thumbnail.jpg?X-Amz-Algorithm=...",
  "expires_in": 3600,
  "upload_headers": {
    "Content-Type": "image/jpeg"
  }
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `upload_id` | string | Unique upload identifier (save for step 3) |
| `presigned_url` | string | S3 URL to upload the file to |
| `expires_in` | number | URL expiration time in seconds (1 hour) |
| `upload_headers` | object | Headers to include in S3 upload request |

---

## Step 2: Upload to S3

### Endpoint
```
PUT {presigned_url}
```

### Headers
```
Content-Type: image/jpeg
```

### Body
```
[Binary image data]
```

### Example (JavaScript)
```javascript
async function uploadToS3(presignedUrl, file, contentType) {
  const response = await fetch(presignedUrl, {
    method: 'PUT',
    headers: {
      'Content-Type': contentType,
    },
    body: file,
  });

  if (!response.ok) {
    throw new Error('S3 upload failed');
  }

  return response;
}
```

---

## Step 3: Notify Upload Complete

### Endpoint
```
POST /api/content/uploads/{upload_id}/notify
```

### Headers
```
Authorization: Bearer {access_token}
Content-Type: application/json
```

### Request Body
```json
{
  "s3_path": "s3://bucket/thumbnails/550e8400-e29b-41d4-a716-446655440000/thumbnail.jpg",
  "size_bytes": 245760
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `s3_path` | string | Yes | Full S3 path (construct from upload_id and filename) |
| `size_bytes` | number | Yes | Final file size in bytes |

### S3 Path Format
```
s3://{bucket}/thumbnails/{upload_id}/{filename}
```

### Response
```json
{
  "status": "completed"
}
```

---

## Complete Implementation Example

### React/TypeScript

```typescript
import { useState } from 'react';

interface UploadUrlResponse {
  upload_id: string;
  presigned_url: string;
  expires_in: number;
  upload_headers: Record<string, string>;
}

interface ThumbnailUploadProps {
  seriesId: string;
  accessToken: string;
  onSuccess: () => void;
  onError: (error: Error) => void;
}

export function useThumbnailUpload({
  seriesId,
  accessToken,
  onSuccess,
  onError,
}: ThumbnailUploadProps) {
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);

  const uploadThumbnail = async (file: File) => {
    try {
      setUploading(true);
      setProgress(10);

      // Validate file
      if (!file.type.startsWith('image/')) {
        throw new Error('File must be an image');
      }
      if (file.size > 5 * 1024 * 1024) {
        throw new Error('File size must not exceed 5MB');
      }

      setProgress(20);

      // Step 1: Request upload URL
      const uploadUrlResponse = await fetch(
        'https://api.example.com/api/content/upload-url',
        {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${accessToken}`,
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            filename: file.name,
            content_type: file.type,
            size_bytes: file.size,
            metadata: {
              series_id: seriesId,
              type: 'thumbnail',
            },
          }),
        }
      );

      if (!uploadUrlResponse.ok) {
        throw new Error('Failed to get upload URL');
      }

      const uploadData: UploadUrlResponse = await uploadUrlResponse.json();
      setProgress(40);

      // Step 2: Upload to S3
      const s3Response = await fetch(uploadData.presigned_url, {
        method: 'PUT',
        headers: {
          'Content-Type': file.type,
        },
        body: file,
      });

      if (!s3Response.ok) {
        throw new Error('Failed to upload to S3');
      }

      setProgress(70);

      // Step 3: Notify completion
      const s3Path = `s3://bucket/thumbnails/${uploadData.upload_id}/${file.name}`;
      
      const notifyResponse = await fetch(
        `https://api.example.com/api/content/uploads/${uploadData.upload_id}/notify`,
        {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${accessToken}`,
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            s3_path: s3Path,
            size_bytes: file.size,
          }),
        }
      );

      if (!notifyResponse.ok) {
        throw new Error('Failed to notify upload completion');
      }

      setProgress(100);
      onSuccess();
    } catch (error) {
      onError(error as Error);
    } finally {
      setUploading(false);
    }
  };

  return { uploadThumbnail, uploading, progress };
}
```

### Usage Example

```typescript
function SeriesThumbnailUploader({ seriesId }: { seriesId: string }) {
  const { accessToken } = useAuth();
  const { uploadThumbnail, uploading, progress } = useThumbnailUpload({
    seriesId,
    accessToken,
    onSuccess: () => {
      console.log('Thumbnail uploaded successfully!');
      // Refresh series data to show new thumbnail
    },
    onError: (error) => {
      console.error('Upload failed:', error.message);
    },
  });

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      uploadThumbnail(file);
    }
  };

  return (
    <div>
      <input
        type="file"
        accept="image/*"
        onChange={handleFileChange}
        disabled={uploading}
      />
      {uploading && <progress value={progress} max={100} />}
    </div>
  );
}
```

---

## Error Handling

### Common Errors

| HTTP Status | Error Message | Description |
|-------------|---------------|-------------|
| 400 | "Thumbnail must be an image file" | Wrong content type |
| 400 | "Thumbnail size must not exceed 5MB" | File too large |
| 403 | "User must be onboarded as a creator first" | User not a creator |
| 404 | "Series not found or access denied" | Invalid series_id or not owned |
| 500 | "Failed to generate upload URL" | AWS configuration issue |

### Error Response Format
```json
{
  "error": "Thumbnail must be an image file"
}
```

### Example Error Handler

```typescript
try {
  await uploadThumbnail(file);
} catch (error) {
  if (error instanceof Response) {
    const data = await error.json();
    switch (error.status) {
      case 400:
        showNotification('Invalid file: ' + data.error, 'error');
        break;
      case 403:
        showNotification('You must be a creator to upload thumbnails', 'error');
        break;
      case 404:
        showNotification('Series not found', 'error');
        break;
      default:
        showNotification('Upload failed. Please try again.', 'error');
    }
  }
}
```

---

## Validation Requirements

### Client-Side Validation

```typescript
function validateThumbnail(file: File): string | null {
  // Check file type
  if (!file.type.startsWith('image/')) {
    return 'Please select an image file';
  }

  // Check file size (5MB)
  const maxSize = 5 * 1024 * 1024;
  if (file.size > maxSize) {
    return 'Image size must not exceed 5MB';
  }

  // Recommended: Check image dimensions
  return new Promise((resolve) => {
    const img = new Image();
    img.onload = () => {
      if (img.width < 300 || img.height < 300) {
        resolve('Image must be at least 300x300 pixels');
      } else {
        resolve(null);
      }
    };
    img.src = URL.createObjectURL(file);
  });
}
```

### Supported Formats
- JPEG (`.jpg`, `.jpeg`)
- PNG (`.png`)
- WebP (`.webp`)
- GIF (`.gif`)

---

## Display Thumbnail

After successful upload, the series `thumbnail_url` is automatically updated. Fetch the series to get the URL:

```typescript
// GET /content/series/{series_id}
const response = await fetch(`https://api.example.com/content/series/${seriesId}`);
const series = await response.json();

console.log(series.thumbnail_url);
// Output: "https://bucket.s3.amazonaws.com/thumbnails/550e8400.../thumbnail.jpg"
```

### Display in UI

```tsx
<img 
  src={series.thumbnail_url || '/default-thumbnail.png'} 
  alt={series.title}
  className="series-thumbnail"
/>
```

---

## Testing Checklist

- [ ] File type validation works (only images allowed)
- [ ] File size validation works (max 5MB)
- [ ] Upload progress indicator displays correctly
- [ ] Error messages display properly
- [ ] Thumbnail displays after upload
- [ ] Upload works for different image formats (JPG, PNG, WebP)
- [ ] Loading states prevent duplicate uploads
- [ ] Series owner can upload (403 for non-owners)

---

## Notes

- **Upload URL expiration:** Presigned URLs expire after 1 hour. Request a new URL if expired.
- **AWS credentials:** If AWS is not configured, the backend returns mock URLs for development.
- **Public URLs:** Thumbnails are served from S3. In production, consider CloudFront for better performance.
- **Concurrent uploads:** Each upload must complete before starting another for the same series.
- **File replacement:** Uploading a new thumbnail replaces the previous one (URL updated).

---

## Quick Reference

### Endpoints Summary

| Step | Method | Endpoint | Auth |
|------|--------|----------|------|
| 1. Request URL | POST | `/api/content/upload-url` | ✓ Required |
| 2. Upload File | PUT | `{presigned_url}` | ✗ None |
| 3. Notify Complete | POST | `/api/content/uploads/{upload_id}/notify` | ✓ Required |

### Request Payloads

**Step 1:**
```json
{"filename": "thumb.jpg", "content_type": "image/jpeg", "size_bytes": 245760, "metadata": {"series_id": "abc123", "type": "thumbnail"}}
```

**Step 3:**
```json
{"s3_path": "s3://bucket/thumbnails/{upload_id}/thumb.jpg", "size_bytes": 245760}
```
