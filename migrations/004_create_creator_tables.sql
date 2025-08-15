-- Migration: 004_create_creator_tables.sql
-- Description: Create creator profiles, payout details, and analytics tables
-- Created: 2025-08-15

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create creator_profiles table
CREATE TABLE IF NOT EXISTS creator_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    bio TEXT,
    kyc_document_s3_path VARCHAR(500),
    kyc_status VARCHAR(20) DEFAULT 'pending' CHECK (kyc_status IN ('pending', 'verified', 'rejected')),
    rating DECIMAL(3,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create payout_details table
CREATE TABLE IF NOT EXISTS payout_details (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    creator_id UUID NOT NULL UNIQUE,
    bank_name VARCHAR(255),
    account_number VARCHAR(50),
    ifsc_code VARCHAR(20),
    account_holder VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create creator_analytics table
CREATE TABLE IF NOT EXISTS creator_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    creator_id UUID NOT NULL,
    date DATE NOT NULL,
    views BIGINT DEFAULT 0,
    watch_time_seconds BIGINT DEFAULT 0,
    earnings DECIMAL(10,2) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_creator_profiles_deleted_at ON creator_profiles(deleted_at);
CREATE INDEX IF NOT EXISTS idx_creator_profiles_user_id ON creator_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_creator_profiles_kyc_status ON creator_profiles(kyc_status);

CREATE INDEX IF NOT EXISTS idx_payout_details_deleted_at ON payout_details(deleted_at);
CREATE INDEX IF NOT EXISTS idx_payout_details_creator_id ON payout_details(creator_id);

CREATE INDEX IF NOT EXISTS idx_creator_analytics_deleted_at ON creator_analytics(deleted_at);
CREATE INDEX IF NOT EXISTS idx_creator_analytics_creator_id ON creator_analytics(creator_id);
CREATE INDEX IF NOT EXISTS idx_creator_analytics_date ON creator_analytics(date);
CREATE INDEX IF NOT EXISTS idx_creator_analytics_creator_date ON creator_analytics(creator_id, date);

-- Add foreign key constraints
ALTER TABLE creator_profiles 
ADD CONSTRAINT fk_creator_profiles_user_id 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE payout_details 
ADD CONSTRAINT fk_payout_details_creator_id 
FOREIGN KEY (creator_id) REFERENCES creator_profiles(id) ON DELETE CASCADE;

ALTER TABLE creator_analytics 
ADD CONSTRAINT fk_creator_analytics_creator_id 
FOREIGN KEY (creator_id) REFERENCES creator_profiles(id) ON DELETE CASCADE;

-- Add comments to tables
COMMENT ON TABLE creator_profiles IS 'Creator profiles for content creators';
COMMENT ON TABLE payout_details IS 'Bank account details for creator payouts';
COMMENT ON TABLE creator_analytics IS 'Daily analytics for creator performance';

-- Add comments to columns
COMMENT ON COLUMN creator_profiles.id IS 'Primary key (UUID)';
COMMENT ON COLUMN creator_profiles.user_id IS 'Foreign key to users table';
COMMENT ON COLUMN creator_profiles.display_name IS 'Public display name for the creator';
COMMENT ON COLUMN creator_profiles.bio IS 'Creator biography/description';
COMMENT ON COLUMN creator_profiles.kyc_document_s3_path IS 'S3 path to KYC document';
COMMENT ON COLUMN creator_profiles.kyc_status IS 'KYC verification status';
COMMENT ON COLUMN creator_profiles.rating IS 'Creator rating (0.00 to 5.00)';

COMMENT ON COLUMN payout_details.id IS 'Primary key (UUID)';
COMMENT ON COLUMN payout_details.creator_id IS 'Foreign key to creator_profiles table';
COMMENT ON COLUMN payout_details.bank_name IS 'Name of the bank';
COMMENT ON COLUMN payout_details.account_number IS 'Bank account number';
COMMENT ON COLUMN payout_details.ifsc_code IS 'IFSC code for the bank branch';
COMMENT ON COLUMN payout_details.account_holder IS 'Name of the account holder';

COMMENT ON COLUMN creator_analytics.id IS 'Primary key (UUID)';
COMMENT ON COLUMN creator_analytics.creator_id IS 'Foreign key to creator_profiles table';
COMMENT ON COLUMN creator_analytics.date IS 'Date of the analytics';
COMMENT ON COLUMN creator_analytics.views IS 'Number of views on this date';
COMMENT ON COLUMN creator_analytics.watch_time_seconds IS 'Total watch time in seconds';
COMMENT ON COLUMN creator_analytics.earnings IS 'Earnings for this date';
