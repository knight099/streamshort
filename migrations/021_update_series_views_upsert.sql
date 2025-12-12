-- Migration: Update series_views to use upsert pattern with view_count
-- Instead of creating new rows per view, aggregate by (series_id, user_id)

-- Add view_count column if it doesn't exist
ALTER TABLE series_views ADD COLUMN IF NOT EXISTS view_count BIGINT DEFAULT 1;

-- Add updated_at column if it doesn't exist
ALTER TABLE series_views ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Step 1: Consolidate existing duplicate (series_id, user_id) rows
-- For each unique (series_id, user_id) combination where user_id is NOT NULL:
-- - Keep the most recent row (by created_at)
-- - Update its view_count to the total count of rows for that combination
-- - Delete the duplicate older rows

-- First, update view_count on the rows we want to keep (most recent per series_id, user_id)
WITH counts AS (
    SELECT series_id, user_id, COUNT(*) as total_views, MAX(created_at) as latest_created_at
    FROM series_views
    WHERE user_id IS NOT NULL
    GROUP BY series_id, user_id
    HAVING COUNT(*) > 1
),
to_keep AS (
    SELECT DISTINCT ON (sv.series_id, sv.user_id) sv.id, c.total_views
    FROM series_views sv
    JOIN counts c ON sv.series_id = c.series_id AND sv.user_id = c.user_id
    ORDER BY sv.series_id, sv.user_id, sv.created_at DESC
)
UPDATE series_views sv
SET view_count = tk.total_views, updated_at = NOW()
FROM to_keep tk
WHERE sv.id = tk.id;

-- Now delete the duplicate rows (keeping only the most recent one per series_id, user_id)
DELETE FROM series_views sv
WHERE user_id IS NOT NULL
  AND id NOT IN (
    SELECT DISTINCT ON (series_id, user_id) id
    FROM series_views
    WHERE user_id IS NOT NULL
    ORDER BY series_id, user_id, created_at DESC
  );

-- Step 2: Create unique constraint for upsert pattern (series_id, user_id)
-- For anonymous users (user_id IS NULL), we keep individual rows since we can't aggregate by NULL
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'uix_series_views_series_user'
    ) THEN
        -- Only create unique constraint for non-null user_id combinations
        CREATE UNIQUE INDEX uix_series_views_series_user 
        ON series_views(series_id, user_id) 
        WHERE user_id IS NOT NULL;
    END IF;
END $$;

-- Add index for efficient queries
CREATE INDEX IF NOT EXISTS idx_series_views_updated_at ON series_views(updated_at);

