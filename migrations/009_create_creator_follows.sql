BEGIN;

CREATE TABLE IF NOT EXISTS creator_follows (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    follower_id uuid NOT NULL,
    creator_id uuid NOT NULL,
    created_at timestamptz DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT uix_follower_creator UNIQUE (follower_id, creator_id)
);

CREATE INDEX IF NOT EXISTS idx_creator_follows_creator ON creator_follows (creator_id);
CREATE INDEX IF NOT EXISTS idx_creator_follows_follower ON creator_follows (follower_id);
CREATE INDEX IF NOT EXISTS idx_creator_follows_deleted_at ON creator_follows (deleted_at);

COMMIT;
