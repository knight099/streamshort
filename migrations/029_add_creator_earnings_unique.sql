-- Add unique constraint on creator_earnings for upsert by (creator_id, subscription_id)
-- This allows updating existing earnings records for the same subscription

CREATE UNIQUE INDEX IF NOT EXISTS idx_creator_earnings_creator_subscription 
ON creator_earnings(creator_id, subscription_id) 
WHERE subscription_id IS NOT NULL;
