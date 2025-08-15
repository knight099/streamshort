-- Migration: 005_fix_data_types.sql
-- Description: Fix data type mismatches for UUID consistency
-- Created: 2025-08-16

-- This migration fixes the data type issues that caused the previous migration to fail

-- First, enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Drop existing tables if they exist (due to failed migration)
DROP TABLE IF EXISTS creator_analytics CASCADE;
DROP TABLE IF EXISTS payout_details CASCADE;
DROP TABLE IF EXISTS creator_profiles CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS otp_transactions CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Recreate users table with UUID
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    phone VARCHAR(20) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for users
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);

-- Recreate OTP transactions table with UUID
CREATE TABLE IF NOT EXISTS otp_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    txn_id VARCHAR(50) NOT NULL UNIQUE,
    phone VARCHAR(20) NOT NULL,
    otp VARCHAR(10) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for OTP transactions
CREATE INDEX IF NOT EXISTS idx_otp_transactions_deleted_at ON otp_transactions(deleted_at);
CREATE INDEX IF NOT EXISTS idx_otp_transactions_txn_id ON otp_transactions(txn_id);
CREATE INDEX IF NOT EXISTS idx_otp_transactions_phone ON otp_transactions(phone);
CREATE INDEX IF NOT EXISTS idx_otp_transactions_expires_at ON otp_transactions(expires_at);
CREATE INDEX IF NOT EXISTS idx_otp_transactions_used ON otp_transactions(used);
CREATE INDEX IF NOT EXISTS idx_otp_transactions_phone_otp_used_expires 
ON otp_transactions(phone, otp, used, expires_at);

-- Recreate refresh tokens table with UUID
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token VARCHAR(255) NOT NULL UNIQUE,
    user_id UUID NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for refresh tokens
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_deleted_at ON refresh_tokens(deleted_at);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_revoked ON refresh_tokens(revoked);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_revoked_expires 
ON refresh_tokens(token, revoked, expires_at);

-- Add foreign key constraint for refresh tokens
ALTER TABLE refresh_tokens 
ADD CONSTRAINT fk_refresh_tokens_user_id 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Create creator profiles table
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

-- Create indexes for creator profiles
CREATE INDEX IF NOT EXISTS idx_creator_profiles_deleted_at ON creator_profiles(deleted_at);
CREATE INDEX IF NOT EXISTS idx_creator_profiles_user_id ON creator_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_creator_profiles_kyc_status ON creator_profiles(kyc_status);

-- Create payout details table
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

-- Create indexes for payout details
CREATE INDEX IF NOT EXISTS idx_payout_details_deleted_at ON payout_details(deleted_at);
CREATE INDEX IF NOT EXISTS idx_payout_details_creator_id ON payout_details(creator_id);

-- Create creator analytics table
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

-- Create indexes for creator analytics
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
COMMENT ON TABLE users IS 'User accounts for phone-based authentication';
COMMENT ON TABLE otp_transactions IS 'OTP transactions for phone verification';
COMMENT ON TABLE refresh_tokens IS 'Refresh tokens for JWT authentication';
COMMENT ON TABLE creator_profiles IS 'Creator profiles for content creators';
COMMENT ON TABLE payout_details IS 'Bank account details for creator payouts';
COMMENT ON TABLE creator_analytics IS 'Daily analytics for creator performance';
