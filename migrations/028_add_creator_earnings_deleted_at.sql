-- Add deleted_at column to creator_earnings for GORM soft delete support
ALTER TABLE creator_earnings 
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_creator_earnings_deleted_at 
ON creator_earnings(deleted_at);
