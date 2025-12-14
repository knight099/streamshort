-- Add unique constraint on creator_payouts for (creator_id, month_date)
-- Required for the ON CONFLICT upsert in real-time earnings calculation

CREATE UNIQUE INDEX IF NOT EXISTS idx_creator_payouts_creator_month 
ON creator_payouts(creator_id, month_date);
