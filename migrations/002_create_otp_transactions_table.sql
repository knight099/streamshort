-- Migration: 002_create_otp_transactions_table.sql
-- Description: Create OTP transactions table for phone OTP verification
-- Created: 2025-08-15

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

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_otp_transactions_deleted_at ON otp_transactions(deleted_at);
CREATE INDEX IF NOT EXISTS idx_otp_transactions_txn_id ON otp_transactions(txn_id);
CREATE INDEX IF NOT EXISTS idx_otp_transactions_phone ON otp_transactions(phone);
CREATE INDEX IF NOT EXISTS idx_otp_transactions_expires_at ON otp_transactions(expires_at);
CREATE INDEX IF NOT EXISTS idx_otp_transactions_used ON otp_transactions(used);

-- Create composite index for OTP verification queries
CREATE INDEX IF NOT EXISTS idx_otp_transactions_phone_otp_used_expires 
ON otp_transactions(phone, otp, used, expires_at);

-- Add comment to table
COMMENT ON TABLE otp_transactions IS 'OTP transactions for phone verification';

-- Add comments to columns
COMMENT ON COLUMN otp_transactions.id IS 'Primary key';
COMMENT ON COLUMN otp_transactions.txn_id IS 'Unique transaction ID for OTP request';
COMMENT ON COLUMN otp_transactions.phone IS 'Phone number for OTP';
COMMENT ON COLUMN otp_transactions.otp IS 'One-time password (6-digit code)';
COMMENT ON COLUMN otp_transactions.expires_at IS 'OTP expiration timestamp';
COMMENT ON COLUMN otp_transactions.used IS 'Whether OTP has been used for verification';
COMMENT ON COLUMN otp_transactions.created_at IS 'Record creation timestamp';
COMMENT ON COLUMN otp_transactions.updated_at IS 'Record last update timestamp';
COMMENT ON COLUMN otp_transactions.deleted_at IS 'Soft delete timestamp (NULL if not deleted)';
