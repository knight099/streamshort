-- Rollback Migration File
-- Description: Rollback all migrations for Streamshort database
-- Created: 2025-08-15
-- Usage: psql -d your_database -f rollback.sql
-- WARNING: This will drop all tables and data!

-- Start transaction
BEGIN;

-- Drop all tables (due to foreign key constraints)
DROP TABLE IF EXISTS creator_analytics CASCADE;
DROP TABLE IF EXISTS payout_details CASCADE;
DROP TABLE IF EXISTS creator_profiles CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS otp_transactions CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS schema_migrations CASCADE;

-- Commit transaction
COMMIT;

-- Display confirmation
SELECT 'All tables dropped successfully' as status;
