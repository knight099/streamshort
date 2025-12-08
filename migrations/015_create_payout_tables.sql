CREATE TABLE IF NOT EXISTS monthly_viewer_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    creator_id UUID NOT NULL,
    month_date DATE NOT NULL,
    watch_time_seconds BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_viewer_stats_user_date ON monthly_viewer_stats(user_id, month_date);
CREATE INDEX IF NOT EXISTS idx_viewer_stats_creator_date ON monthly_viewer_stats(creator_id, month_date);
CREATE UNIQUE INDEX IF NOT EXISTS idx_viewer_stats_unique ON monthly_viewer_stats(user_id, creator_id, month_date);

CREATE TABLE IF NOT EXISTS creator_payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id UUID NOT NULL,
    month_date DATE NOT NULL,
    total_earnings DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'paid', 'failed')),
    processed_at TIMESTAMPTZ,
    transaction_ref TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS subscription_revenues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    subscription_id UUID NOT NULL,
    month_date DATE NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    distributable DECIMAL(10,2) NOT NULL,
    platform_fee DECIMAL(10,2) NOT NULL,
    is_distributed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

