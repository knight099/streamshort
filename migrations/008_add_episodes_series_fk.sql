BEGIN;

-- Ensure an index exists for efficient joins and deletes
CREATE INDEX IF NOT EXISTS idx_episodes_series_id ON episodes (series_id);

-- Add foreign key constraint from episodes.series_id to series.id with cascade delete
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.table_constraints tc
        WHERE tc.constraint_name = 'fk_episodes_series_id'
          AND tc.table_name = 'episodes'
          AND tc.constraint_type = 'FOREIGN KEY'
    ) THEN
        ALTER TABLE episodes
        ADD CONSTRAINT fk_episodes_series_id
        FOREIGN KEY (series_id)
        REFERENCES series(id)
        ON DELETE CASCADE;
    END IF;
END $$;

COMMIT;


