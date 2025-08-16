# Content API Implementation Guide

## 🎬 Overview

The Content API provides comprehensive functionality for managing video series, episodes, and uploads in the Streamshort platform. This implementation covers all endpoints defined in the OpenAPI specification.

## 🏗️ Architecture

### Models
- **Series**: Video series with metadata, pricing, and status
- **Episode**: Individual episodes within a series
- **UploadRequest**: Tracks file upload requests and status

### Key Features
- **Creator-only operations**: Series/episode creation requires creator onboarding
- **Public content discovery**: Series listing and details are publicly accessible
- **Protected playback**: Episode manifests require authentication
- **Upload workflow**: Complete upload-to-playback pipeline

## 📋 API Endpoints

### Public Endpoints (No Authentication Required)

#### 1. List Series
```
GET /content/series
```

**Query Parameters:**
- `language` (optional): Filter by language code (e.g., "en", "hi")
- `category` (optional): Filter by category tag
- `page` (optional): Page number (default: 1)
- `per_page` (optional): Items per page (default: 20, max: 100)

**Response:**
```json
{
  "total": 1,
  "items": [
    {
      "id": "series_123",
      "title": "The Short Life",
      "synopsis": "A 5-episode short drama",
      "language": "hi",
      "category_tags": ["drama", "shorts"],
      "price_type": "subscription",
      "price_amount": 99.00,
      "status": "published"
    }
  ]
}
```

#### 2. Get Series Details
```
GET /content/series/{id}
```

**Response:**
```json
{
  "id": "series_123",
  "title": "The Short Life",
  "synopsis": "A 5-episode short drama",
  "language": "hi",
  "category_tags": ["drama", "shorts"],
  "price_type": "subscription",
  "price_amount": 99.00,
  "status": "published",
  "creator": {
    "display_name": "Arjun Films",
    "bio": "Short films in Hindi & Marathi"
  },
  "episodes": [
    {
      "id": "episode_001",
      "title": "Episode 1: The Train Ride",
      "episode_number": 1,
      "duration_seconds": 300,
      "status": "published"
    }
  ]
}
```

### Protected Endpoints (Authentication Required)

#### 3. Create Series
```
POST /api/content/series
```

**Request Body:**
```json
{
  "title": "The Short Life",
  "synopsis": "A 5-episode short drama",
  "language": "hi",
  "category_tags": ["drama", "shorts"],
  "price_type": "subscription",
  "price_amount": 99.00,
  "thumbnail_url": "https://example.com/thumb.jpg"
}
```

**Requirements:**
- User must be onboarded as a creator
- Title, synopsis, and language are required
- Price type must be one of: "free", "subscription", "one_time"

**Response:**
```json
{
  "id": "series_123",
  "title": "The Short Life",
  "status": "draft"
}
```

#### 4. Update Series
```
PUT /api/content/series/{id}
```

**Request Body:** (All fields optional)
```json
{
  "title": "Updated Title",
  "status": "published"
}
```

**Requirements:**
- User must own the series (be the creator)

#### 5. Create Episode
```
POST /api/content/series/{id}/episodes
```

**Request Body:**
```json
{
  "title": "Episode 1: The Train Ride",
  "episode_number": 1,
  "duration_seconds": 300
}
```

**Requirements:**
- User must own the series
- Episode number must be unique within the series
- Duration must be positive

#### 6. Request Upload URL
```
POST /api/content/upload-url
```

**Request Body:**
```json
{
  "filename": "episode1_master.mp4",
  "content_type": "video/mp4",
  "size_bytes": 73400320,
  "metadata": {
    "series_id": "series_123",
    "episode_title": "Episode 1 - The Beginning"
  }
}
```

**Response:**
```json
{
  "upload_id": "upl_94f3d82b",
  "presigned_url": "https://s3.amazonaws.com/bucket/upl_94f3d82b?AWSAccessKeyId=...",
  "expires_in": 3600,
  "upload_headers": {
    "Content-Type": "video/mp4"
  }
}
```

#### 7. Notify Upload Complete
```
POST /api/content/uploads/{upload_id}/notify
```

**Request Body:**
```json
{
  "s3_path": "s3://bucket/upl_94f3d82b/episode1_master.mp4",
  "size_bytes": 73400320
}
```

**Response:**
```json
{
  "status": "queued_for_transcoding"
}
```

#### 8. Get Episode Manifest
```
GET /api/episodes/{id}/manifest
```

**Response:**
```json
{
  "manifest_url": "https://cdn.streamshort.com/hls/episode1/index.m3u8?Expires=1723598700&Signature=...",
  "expires_at": "2025-08-15T12:00:00Z"
}
```

**Requirements:**
- Episode must be published
- User must be authenticated (future: check subscription)

## 🗄️ Database Schema

### Series Table
```sql
CREATE TABLE series (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id UUID NOT NULL REFERENCES creator_profiles(id),
    title VARCHAR NOT NULL,
    synopsis TEXT NOT NULL,
    language VARCHAR NOT NULL,
    category_tags TEXT[],
    price_type VARCHAR(20) CHECK (price_type IN ('free', 'subscription', 'one_time')),
    price_amount DECIMAL(10,2),
    thumbnail_url VARCHAR,
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

### Episodes Table
```sql
CREATE TABLE episodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    series_id UUID NOT NULL REFERENCES series(id),
    title VARCHAR NOT NULL,
    episode_number INTEGER NOT NULL,
    duration_seconds INTEGER NOT NULL,
    s3_master_path VARCHAR,
    hls_manifest_url VARCHAR,
    thumb_url VARCHAR,
    captions_url VARCHAR,
    status VARCHAR(30) DEFAULT 'pending_upload' CHECK (status IN ('pending_upload', 'queued_transcode', 'ready', 'published')),
    published_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

### Upload Requests Table
```sql
CREATE TABLE upload_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    filename VARCHAR NOT NULL,
    content_type VARCHAR NOT NULL,
    size_bytes BIGINT NOT NULL,
    metadata JSONB,
    status VARCHAR(30) DEFAULT 'pending' CHECK (status IN ('pending', 'uploading', 'completed', 'failed')),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

## 🔄 Workflow

### 1. Content Creation Flow
```
Creator Onboarding → Series Creation → Episode Creation → Upload → Transcoding → Publishing
```

### 2. Upload Workflow
```
Request Upload URL → Upload to S3 → Notify Completion → Queue Transcoding → Generate HLS → Ready for Playback
```

### 3. Playback Flow
```
Authenticate → Check Subscription → Get Manifest → Stream HLS
```

## 🚀 Testing

### Manual Testing
Use the provided test script:
```bash
./test_content_api.sh
```

### API Testing with curl

#### Create Series
```bash
curl -X POST http://localhost:8080/api/content/series \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Series",
    "synopsis": "Test description",
    "language": "en",
    "category_tags": ["test"],
    "price_type": "free"
  }'
```

#### List Series (Public)
```bash
curl http://localhost:8080/content/series?language=en
```

## 🔧 Configuration

### Environment Variables
- `DATABASE_URL`: PostgreSQL connection string
- `JWT_SECRET`: Secret for JWT token generation

### Database Migration
The API automatically creates tables using GORM's `AutoMigrate`:
```go
db.AutoMigrate(
    &models.Series{},
    &models.Episode{},
    &models.UploadRequest{},
    // ... other models
)
```

## 🚧 Future Enhancements

### 1. AWS S3 Integration
- Replace mock pre-signed URLs with actual S3 integration
- Implement proper file upload handling

### 2. Video Transcoding
- Integrate with AWS MediaConvert or similar service
- Generate HLS manifests and thumbnails

### 3. CDN Integration
- Implement signed URLs with proper expiration
- Add regional CDN support

### 4. Subscription Management
- Check user subscription status before playback
- Implement pay-per-view logic

### 5. Analytics
- Track view counts and watch time
- Generate creator analytics

## 📝 Notes

- **Mock Responses**: Some endpoints return mock data (e.g., S3 URLs) for development
- **Validation**: Basic validation is implemented; enhance for production use
- **Error Handling**: Standard HTTP status codes with JSON error responses
- **Security**: JWT-based authentication with creator ownership verification

## 🐛 Troubleshooting

### Common Issues

1. **"User must be onboarded as a creator first"**
   - Complete creator onboarding before creating content

2. **"Series not found or access denied"**
   - Verify you own the series or it exists

3. **"Episode number already exists"**
   - Use unique episode numbers within a series

4. **Database connection issues**
   - Check `DATABASE_URL` environment variable
   - Ensure PostgreSQL is running

### Debug Mode
Enable detailed logging by setting GORM log level in `config/database.go`:
```go
Logger: logger.Default.LogMode(logger.Info)
```
