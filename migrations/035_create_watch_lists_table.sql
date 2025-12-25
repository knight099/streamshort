-- Create watch_lists table
CREATE TABLE IF NOT EXISTS watch_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    series_id UUID NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, series_id)
);

-- Index for faster lookups by user
CREATE INDEX IF NOT EXISTS idx_watch_lists_user_id ON watch_lists(user_id);
-- Index for faster lookups by series (optional, but good for "who watched this")
CREATE INDEX IF NOT EXISTS idx_watch_lists_series_id ON watch_lists(series_id);
