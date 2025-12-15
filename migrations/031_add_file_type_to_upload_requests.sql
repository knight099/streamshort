-- Migration: Add file_type column to upload_requests table
-- Purpose: Support different upload types (video, thumbnail, caption)

-- Add file_type column with default value 'video' for backward compatibility
ALTER TABLE upload_requests
ADD COLUMN IF NOT EXISTS file_type VARCHAR(20) DEFAULT 'video';

-- Add check constraint to ensure valid file types
ALTER TABLE upload_requests
ADD CONSTRAINT check_file_type CHECK (file_type IN ('video', 'thumbnail', 'caption'));

-- Update any existing records to have 'video' as file type
UPDATE upload_requests
SET file_type = 'video'
WHERE file_type IS NULL;
