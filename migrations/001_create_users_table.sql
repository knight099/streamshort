-- Migration: 001_create_users_table.sql
-- Description: Create users table for phone-based authentication
-- Created: 2025-08-15

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    phone VARCHAR(20) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create index for soft deletes
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- Create unique index on phone (already covered by UNIQUE constraint, but explicit for clarity)
CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);

-- Add comment to table
COMMENT ON TABLE users IS 'User accounts for phone-based authentication';

-- Add comments to columns
COMMENT ON COLUMN users.id IS 'Primary key';
COMMENT ON COLUMN users.phone IS 'Phone number in E.164 format (e.g., +919876543210)';
COMMENT ON COLUMN users.created_at IS 'Record creation timestamp';
COMMENT ON COLUMN users.updated_at IS 'Record last update timestamp';
COMMENT ON COLUMN users.deleted_at IS 'Soft delete timestamp (NULL if not deleted)';
