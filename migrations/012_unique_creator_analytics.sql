-- Ensure a single row per creator in creator_analytics (daily analytics or similar)
-- Using DO block for safe constraint addition
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'uix_creator_analytics_creator'
          AND table_name = 'creator_analytics'
    ) THEN
        ALTER TABLE creator_analytics
        ADD CONSTRAINT uix_creator_analytics_creator UNIQUE (creator_id);
    END IF;
END $$;
