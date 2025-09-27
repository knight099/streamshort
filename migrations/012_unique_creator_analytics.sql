-- Ensure a single row per creator in creator_analytics
ALTER TABLE creator_analytics
    ADD CONSTRAINT IF NOT EXISTS uix_creator_analytics_creator UNIQUE (creator_id);

