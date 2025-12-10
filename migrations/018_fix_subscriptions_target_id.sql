-- Migration: Fix subscriptions table - ensure target_id is dropped for global subscriptions
-- This migration fixes the issue where target_id column still exists with NOT NULL constraint

-- 1) First, make target_id nullable (if it exists) to avoid constraint violations
ALTER TABLE subscriptions
ALTER COLUMN target_id DROP NOT NULL;

-- 2) Drop target_id column if it exists
ALTER TABLE subscriptions
DROP COLUMN IF EXISTS target_id;

-- 3) Drop all CHECK constraints on subscriptions table (to handle inline constraints)
-- This ensures we remove both named and unnamed constraints
DO $$ 
DECLARE
    r RECORD;
BEGIN
    -- Find and drop all CHECK constraints
    FOR r IN (
        SELECT conname 
        FROM pg_constraint 
        WHERE conrelid = 'subscriptions'::regclass 
        AND contype = 'c'
    ) LOOP
        EXECUTE 'ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS ' || quote_ident(r.conname);
    END LOOP;
END $$;

-- 4) Re-add constraints with correct values
-- Target type constraint (includes 'global')
ALTER TABLE subscriptions
ADD CONSTRAINT subscriptions_target_type_check
CHECK (target_type IN ('series', 'creator', 'global'));

-- Status constraint (includes 'pending' and 'halted')
ALTER TABLE subscriptions
ADD CONSTRAINT subscriptions_status_check
CHECK (status IN ('pending', 'active', 'cancelled', 'expired', 'halted'));

-- 5) Ensure key indexes exist
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);

