-- Migration: Update series_views to use upsert pattern with view_count
-- Instead of creating new rows per view, aggregate by (series_id, user_id)

-- Add view_count column if it doesn't exist
ALTER TABLE series_views ADD COLUMN IF NOT EXISTS view_count BIGINT DEFAULT 1;

-- Add updated_at column if it doesn't exist
ALTER TABLE series_views ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Create unique constraint for upsert pattern (series_id, user_id)
-- For anonymous users (user_id IS NULL), we keep individual rows since we can't aggregate by NULL
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'uix_series_views_series_user'
          AND table_name = 'series_views'
    ) THEN
        -- Only create unique constraint for non-null user_id combinations
        CREATE UNIQUE INDEX IF NOT EXISTS uix_series_views_series_user 
        ON series_views(series_id, user_id) 
        WHERE user_id IS NOT NULL;
    END IF;
END $$;

-- Add index for efficient queries
CREATE INDEX IF NOT EXISTS idx_series_views_updated_at ON series_views(updated_at);
