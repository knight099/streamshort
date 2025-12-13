-- Migration: Add foreign keys for episode_likes and user_episode_watches

-- Add foreign key from episode_likes to episodes
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_episode_likes_episode' AND table_name = 'episode_likes'
    ) THEN
        ALTER TABLE episode_likes
        ADD CONSTRAINT fk_episode_likes_episode
        FOREIGN KEY (episode_id) REFERENCES episodes(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Add foreign key from episode_likes to users
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_episode_likes_user' AND table_name = 'episode_likes'
    ) THEN
        ALTER TABLE episode_likes
        ADD CONSTRAINT fk_episode_likes_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Add foreign key from user_episode_watches to episodes
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_user_episode_watches_episode' AND table_name = 'user_episode_watches'
    ) THEN
        ALTER TABLE user_episode_watches
        ADD CONSTRAINT fk_user_episode_watches_episode
        FOREIGN KEY (episode_id) REFERENCES episodes(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Add foreign key from user_episode_watches to users
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_user_episode_watches_user' AND table_name = 'user_episode_watches'
    ) THEN
        ALTER TABLE user_episode_watches
        ADD CONSTRAINT fk_user_episode_watches_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Add foreign key from series_views to series
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_series_views_series' AND table_name = 'series_views'
    ) THEN
        ALTER TABLE series_views
        ADD CONSTRAINT fk_series_views_series
        FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Add foreign key from series_views to users (nullable)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_series_views_user' AND table_name = 'series_views'
    ) THEN
        ALTER TABLE series_views
        ADD CONSTRAINT fk_series_views_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;
