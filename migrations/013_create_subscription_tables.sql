-- Migration: Create subscription tables
-- Description: Creates tables for managing user subscriptions and subscription plans

-- Create subscription_plans table
CREATE TABLE IF NOT EXISTS subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    duration INTEGER NOT NULL, -- Duration in days
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'INR',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Create subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('series', 'creator')),
    target_id UUID NOT NULL,
    razorpay_subscription_id VARCHAR(255),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'cancelled', 'expired')),
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    auto_renew BOOLEAN DEFAULT true,
    plan_id VARCHAR(255) NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'INR',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_target ON subscriptions(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);

-- Insert default subscription plans
INSERT INTO subscription_plans (id, name, description, duration, amount, currency, is_active)
VALUES 
    ('plan_basic_monthly', 'Basic Monthly', 'Access to one series for 30 days', 30, 99.00, 'INR', true),
    ('plan_premium_monthly', 'Premium Monthly', 'Access to all series from one creator for 30 days', 30, 299.00, 'INR', true),
    ('plan_basic_yearly', 'Basic Yearly', 'Access to one series for 365 days', 365, 999.00, 'INR', true),
    ('plan_premium_yearly', 'Premium Yearly', 'Access to all series from one creator for 365 days', 365, 2999.00, 'INR', true)
ON CONFLICT (id) DO NOTHING;
