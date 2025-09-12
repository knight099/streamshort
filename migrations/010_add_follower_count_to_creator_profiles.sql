BEGIN;

ALTER TABLE creator_profiles
ADD COLUMN IF NOT EXISTS follower_count BIGINT NOT NULL DEFAULT 0;

UPDATE creator_profiles cp
SET follower_count = sub.cnt
FROM (
  SELECT creator_id, COUNT(*)::BIGINT AS cnt
  FROM creator_follows
  WHERE deleted_at IS NULL
  GROUP BY creator_id
) AS sub
WHERE cp.id = sub.creator_id;

COMMIT;
