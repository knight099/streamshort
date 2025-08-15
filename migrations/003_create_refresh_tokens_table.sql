-- Migration: 003_create_refresh_tokens_table.sql
-- Description: Create refresh tokens table for JWT token refresh
-- Created: 2025-08-15

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

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_deleted_at ON refresh_tokens(deleted_at);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_revoked ON refresh_tokens(revoked);

-- Create composite index for token validation queries
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_revoked_expires 
ON refresh_tokens(token, revoked, expires_at);

-- Add foreign key constraint to users table
ALTER TABLE refresh_tokens 
ADD CONSTRAINT fk_refresh_tokens_user_id 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Add comment to table
COMMENT ON TABLE refresh_tokens IS 'Refresh tokens for JWT authentication';

-- Add comments to columns
COMMENT ON COLUMN refresh_tokens.id IS 'Primary key';
COMMENT ON COLUMN refresh_tokens.token IS 'Unique refresh token string';
COMMENT ON COLUMN refresh_tokens.user_id IS 'Foreign key to users table';
COMMENT ON COLUMN refresh_tokens.expires_at IS 'Token expiration timestamp';
COMMENT ON COLUMN refresh_tokens.revoked IS 'Whether token has been revoked';
COMMENT ON COLUMN refresh_tokens.created_at IS 'Record creation timestamp';
COMMENT ON COLUMN refresh_tokens.updated_at IS 'Record last update timestamp';
COMMENT ON COLUMN refresh_tokens.deleted_at IS 'Soft delete timestamp (NULL if not deleted)';
