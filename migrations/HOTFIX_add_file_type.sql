-- Quick fix: Add file_type column to upload_requests table
-- Run this on production database immediately

-- Step 1: Add the column with default value
ALTER TABLE upload_requests
ADD COLUMN IF NOT EXISTS file_type VARCHAR(20) DEFAULT 'video';

-- Step 2: Add constraint (PostgreSQL 9.6+ syntax)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'check_file_type'
    ) THEN
        ALTER TABLE upload_requests
        ADD CONSTRAINT check_file_type 
        CHECK (file_type IN ('video', 'thumbnail', 'caption'));
    END IF;
END $$;

-- Step 3: Update any NULL values
UPDATE upload_requests
SET file_type = 'video'
WHERE file_type IS NULL;

-- Verify the column exists
SELECT column_name, data_type, column_default
FROM information_schema.columns
WHERE table_name = 'upload_requests' AND column_name = 'file_type';
