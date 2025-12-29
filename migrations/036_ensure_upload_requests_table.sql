-- Migration: Ensure upload_requests table exists with all required columns
-- Purpose: Create the upload_requests table if it doesn't exist, with proper constraints

-- Create the table if it doesn't exist
CREATE TABLE IF NOT EXISTS upload_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    file_type VARCHAR(20) DEFAULT 'video',
    metadata JSONB,
    status VARCHAR(30) DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_upload_requests_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Ensure metadata column is JSONB type (convert from json if needed)
DO $$
BEGIN
    -- Check if metadata column exists and is of type json (not jsonb)
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'upload_requests' 
        AND column_name = 'metadata' 
        AND data_type = 'json'
    ) THEN
        -- Convert json to jsonb
        ALTER TABLE upload_requests 
        ALTER COLUMN metadata TYPE JSONB USING metadata::jsonb;
        RAISE NOTICE 'Converted metadata column from json to jsonb';
    END IF;
END $$;

-- Ensure metadata column exists with JSONB type if not present
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'upload_requests' 
        AND column_name = 'metadata'
    ) THEN
        ALTER TABLE upload_requests ADD COLUMN metadata JSONB;
    END IF;
END $$;

-- Create index on deleted_at for soft delete queries
CREATE INDEX IF NOT EXISTS idx_upload_requests_deleted_at ON upload_requests(deleted_at);

-- Create index on user_id for user-based queries
CREATE INDEX IF NOT EXISTS idx_upload_requests_user_id ON upload_requests(user_id);

-- Create index on status for filtering
CREATE INDEX IF NOT EXISTS idx_upload_requests_status ON upload_requests(status);

-- Drop existing constraint if it exists (to avoid duplicate constraint errors)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'check_file_type' AND table_name = 'upload_requests'
    ) THEN
        ALTER TABLE upload_requests DROP CONSTRAINT check_file_type;
    END IF;
END $$;

-- Drop GORM-generated constraint if it exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'chk_upload_requests_file_type' AND table_name = 'upload_requests'
    ) THEN
        ALTER TABLE upload_requests DROP CONSTRAINT chk_upload_requests_file_type;
    END IF;
END $$;

-- Add the check constraint with a consistent name
ALTER TABLE upload_requests
ADD CONSTRAINT chk_upload_requests_file_type 
CHECK (file_type IS NULL OR file_type IN ('video', 'thumbnail', 'caption'));

-- Drop existing status constraint if it exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'chk_upload_requests_status' AND table_name = 'upload_requests'
    ) THEN
        ALTER TABLE upload_requests DROP CONSTRAINT chk_upload_requests_status;
    END IF;
END $$;

-- Add the status check constraint
ALTER TABLE upload_requests
ADD CONSTRAINT chk_upload_requests_status 
CHECK (status IS NULL OR status IN ('pending', 'uploading', 'completed', 'failed'));
