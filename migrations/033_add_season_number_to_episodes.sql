-- Migration: Add season_number to episodes table
-- This enables grouping episodes by season within a series

-- Add season_number column with default 1 for existing episodes
ALTER TABLE episodes ADD COLUMN IF NOT EXISTS season_number INT NOT NULL DEFAULT 1;

-- Create composite index for efficient season-based queries
CREATE INDEX IF NOT EXISTS idx_episodes_series_season ON episodes(series_id, season_number, episode_number);

-- Add comment for documentation
COMMENT ON COLUMN episodes.season_number IS 'Season number within the series, starting from 1';
