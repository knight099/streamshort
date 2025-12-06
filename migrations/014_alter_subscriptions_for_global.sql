-- Migration: Alter subscriptions table for global all-access model
-- Changes:
-- 1) Allow target_type 'global'
-- 2) Drop target_id column
-- 3) Drop obsolete target index
-- 4) Ensure essential indexes exist

-- 1) Update CHECK constraint to include 'global'
ALTER TABLE subscriptions
DROP CONSTRAINT IF EXISTS subscriptions_target_type_check;

ALTER TABLE subscriptions
ADD CONSTRAINT subscriptions_target_type_check
CHECK (target_type IN ('series', 'creator', 'global'));

-- 2) Drop target_id column if present
ALTER TABLE subscriptions
DROP COLUMN IF EXISTS target_id;

-- 3) Drop obsolete index on (target_type, target_id)
DROP INDEX IF EXISTS idx_subscriptions_target;

-- 4) Ensure key indexes exist
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);


