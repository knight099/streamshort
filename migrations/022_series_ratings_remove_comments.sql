-- Migration: Drop episode_comments and episode_ratings tables, create series_ratings table
-- Comments feature is being removed, ratings now apply to series instead of episodes

-- Drop the episode_comments table
DROP TABLE IF EXISTS episode_comments CASCADE;

-- Drop the episode_ratings table  
DROP TABLE IF EXISTS episode_ratings CASCADE;

-- Create series_ratings table
CREATE TABLE IF NOT EXISTS series_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    series_id UUID NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score INTEGER NOT NULL CHECK (score >= 1 AND score <= 5),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Create unique constraint: one rating per user per series
CREATE UNIQUE INDEX IF NOT EXISTS idx_series_rating_series_user 
ON series_ratings(series_id, user_id) 
WHERE deleted_at IS NULL;

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_series_ratings_series_id ON series_ratings(series_id);
CREATE INDEX IF NOT EXISTS idx_series_ratings_user_id ON series_ratings(user_id);
CREATE INDEX IF NOT EXISTS idx_series_ratings_deleted_at ON series_ratings(deleted_at);
