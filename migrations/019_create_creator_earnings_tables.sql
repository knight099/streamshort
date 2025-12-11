-- Creator Earnings Table (tracks individual earnings transactions)
CREATE TABLE IF NOT EXISTS creator_earnings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id UUID NOT NULL REFERENCES creator_profiles(id) ON DELETE CASCADE,
    amount DECIMAL(10, 2) NOT NULL,
    earnings_type VARCHAR(50) NOT NULL CHECK (earnings_type IN ('subscription', 'one_time', 'ad_revenue')),
    series_id UUID REFERENCES series(id) ON DELETE SET NULL,
    episode_id UUID REFERENCES episodes(id) ON DELETE SET NULL,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'cancelled')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    paid_at TIMESTAMPTZ,
    payout_id UUID,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_creator_earnings_creator_id ON creator_earnings(creator_id);
CREATE INDEX IF NOT EXISTS idx_creator_earnings_status ON creator_earnings(status);
CREATE INDEX IF NOT EXISTS idx_creator_earnings_created_at ON creator_earnings(created_at);
CREATE INDEX IF NOT EXISTS idx_creator_earnings_series_id ON creator_earnings(series_id);

-- Payouts Table (tracks payout requests and their status)
CREATE TABLE IF NOT EXISTS payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id UUID NOT NULL REFERENCES creator_profiles(id) ON DELETE CASCADE,
    amount DECIMAL(10, 2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    razorpay_payout_id VARCHAR(255),
    razorpay_fund_account_id VARCHAR(255),
    transaction_reference VARCHAR(255),
    payout_method VARCHAR(50) NOT NULL DEFAULT 'bank_transfer' CHECK (payout_method IN ('bank_transfer', 'upi')),
    failure_reason TEXT,
    requested_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payouts_creator_id ON payouts(creator_id);
CREATE INDEX IF NOT EXISTS idx_payouts_status ON payouts(status);
CREATE INDEX IF NOT EXISTS idx_payouts_requested_at ON payouts(requested_at);

-- Add foreign key from creator_earnings to payouts
ALTER TABLE creator_earnings 
    ADD CONSTRAINT fk_creator_earnings_payout 
    FOREIGN KEY (payout_id) REFERENCES payouts(id) ON DELETE SET NULL;

-- Enhance payout_details table with additional fields
ALTER TABLE payout_details ADD COLUMN IF NOT EXISTS account_type VARCHAR(20) DEFAULT 'savings' CHECK (account_type IN ('savings', 'current'));
ALTER TABLE payout_details ADD COLUMN IF NOT EXISTS upi_id VARCHAR(255);
ALTER TABLE payout_details ADD COLUMN IF NOT EXISTS verified BOOLEAN DEFAULT FALSE;
ALTER TABLE payout_details ADD COLUMN IF NOT EXISTS razorpay_contact_id VARCHAR(255);
ALTER TABLE payout_details ADD COLUMN IF NOT EXISTS razorpay_fund_account_id VARCHAR(255);

-- Series Views Table (for tracking individual views)
CREATE TABLE IF NOT EXISTS series_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    series_id UUID NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    session_id VARCHAR(255),
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_series_views_series_id ON series_views(series_id);
CREATE INDEX IF NOT EXISTS idx_series_views_user_id ON series_views(user_id);
CREATE INDEX IF NOT EXISTS idx_series_views_created_at ON series_views(created_at);

-- Add view_count column to series if it doesn't exist
ALTER TABLE series ADD COLUMN IF NOT EXISTS view_count BIGINT DEFAULT 0;

-- Create index for view_count
CREATE INDEX IF NOT EXISTS idx_series_view_count ON series(view_count);
