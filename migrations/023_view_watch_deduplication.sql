-- Migration: View Count and Watch Time Deduplication
-- Track unique episode watches per user (store only first watch)

CREATE TABLE IF NOT EXISTS user_episode_watches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    episode_id UUID NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    watched_seconds INTEGER NOT NULL DEFAULT 0,
    first_watched_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, episode_id)
);

CREATE INDEX IF NOT EXISTS idx_user_episode_watches_user ON user_episode_watches(user_id);
CREATE INDEX IF NOT EXISTS idx_user_episode_watches_episode ON user_episode_watches(episode_id);

-- Note: series_views already has unique index from migration 021:
-- uix_series_views_series_user ON series_views(series_id, user_id) WHERE user_id IS NOT NULL
