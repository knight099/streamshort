-- HOTFIX: Convert metadata column from json to jsonb
-- This is needed because the original table might have been created with json type

-- Convert json to jsonb if the column is of type json
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'upload_requests' 
        AND column_name = 'metadata' 
        AND data_type = 'json'
    ) THEN
        ALTER TABLE upload_requests 
        ALTER COLUMN metadata TYPE JSONB USING metadata::jsonb;
        RAISE NOTICE 'Converted metadata column from json to jsonb';
    ELSIF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'upload_requests' 
        AND column_name = 'metadata' 
        AND data_type = 'jsonb'
    ) THEN
        RAISE NOTICE 'metadata column is already jsonb type';
    ELSE
        -- Add the column if it doesn't exist
        ALTER TABLE upload_requests ADD COLUMN metadata JSONB;
        RAISE NOTICE 'Added metadata column as jsonb type';
    END IF;
END $$;
