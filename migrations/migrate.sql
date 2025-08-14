-- Master Migration File
-- Description: Run all migrations for Streamshort database
-- Created: 2025-08-15
-- Usage: psql -d your_database -f migrate.sql

-- Start transaction
BEGIN;

-- Migration 001: Create users table
\i 001_create_users_table.sql

-- Migration 002: Create OTP transactions table
\i 002_create_otp_transactions_table.sql

-- Migration 003: Create refresh tokens table
\i 003_create_refresh_tokens_table.sql

-- Create migration tracking table
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Record migrations
INSERT INTO schema_migrations (version) VALUES 
    ('001_create_users_table'),
    ('002_create_otp_transactions_table'),
    ('003_create_refresh_tokens_table')
ON CONFLICT (version) DO NOTHING;

-- Commit transaction
COMMIT;

-- Display migration status
SELECT version, applied_at FROM schema_migrations ORDER BY applied_at;
