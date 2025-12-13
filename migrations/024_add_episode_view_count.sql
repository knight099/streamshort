-- Migration: Add view_count column to episodes table
-- Tracks unique views per episode (incremented only once per user)

ALTER TABLE episodes ADD COLUMN IF NOT EXISTS view_count BIGINT DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_episodes_view_count ON episodes(view_count);
