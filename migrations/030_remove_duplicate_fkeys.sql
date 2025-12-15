-- Remove duplicate foreign key constraints
-- Keep the named ones (fk_*), drop the auto-generated GORM ones (*_fkey)

-- series_views: keep fk_series_views_*, drop *_fkey
ALTER TABLE series_views DROP CONSTRAINT IF EXISTS series_views_series_id_fkey;
ALTER TABLE series_views DROP CONSTRAINT IF EXISTS series_views_user_id_fkey;

-- user_episode_watches: keep fk_user_episode_watches_*, drop *_fkey
ALTER TABLE user_episode_watches DROP CONSTRAINT IF EXISTS user_episode_watches_episode_id_fkey;
ALTER TABLE user_episode_watches DROP CONSTRAINT IF EXISTS user_episode_watches_user_id_fkey;
