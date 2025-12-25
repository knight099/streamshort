-- Add current_period_end column to subscriptions table
-- This column tracks the end of the current billing period for recurring subscriptions

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS current_period_end TIMESTAMPTZ;
