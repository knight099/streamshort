-- Add deleted_at column to payouts for GORM soft delete support
ALTER TABLE payouts 
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_payouts_deleted_at 
ON payouts(deleted_at);
