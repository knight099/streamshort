-- Migration: Fix creator_analytics to support per-date rows instead of collapsing all into one row
-- This fixes the dashboard to correctly show last 30 days of analytics

-- Drop the old constraint that only conflicts on creator_id (ignores date)
ALTER TABLE creator_analytics DROP CONSTRAINT IF EXISTS uix_creator_analytics_creator;
ALTER TABLE creator_analytics DROP CONSTRAINT IF EXISTS creator_analytics_creator_id_key;

-- Add the correct composite unique constraint on (creator_id, date)
-- This ensures we get one row per creator per date for proper 30-day dashboard stats
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'uix_creator_analytics_creator_date'
          AND table_name = 'creator_analytics'
    ) THEN
        ALTER TABLE creator_analytics
        ADD CONSTRAINT uix_creator_analytics_creator_date UNIQUE (creator_id, date);
    END IF;
END $$;

-- Add index for efficient date-based queries
CREATE INDEX IF NOT EXISTS idx_creator_analytics_date ON creator_analytics(date);
