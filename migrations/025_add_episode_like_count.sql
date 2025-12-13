-- Migration: Add like_count column to episodes table
-- Tracks total likes per episode (cached count from episode_likes table)

ALTER TABLE episodes ADD COLUMN IF NOT EXISTS like_count BIGINT DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_episodes_like_count ON episodes(like_count);

-- Backfill like_count from existing episode_likes
UPDATE episodes e
SET like_count = (
    SELECT COUNT(*) 
    FROM episode_likes el 
    WHERE el.episode_id = e.id AND el.deleted_at IS NULL
);
