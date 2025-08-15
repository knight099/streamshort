-- Master Migration File
-- Description: Run all migrations for Streamshort database
-- Created: 2025-08-15
-- Usage: psql -d your_database -f migrate.sql

-- Start transaction
BEGIN;

-- Migration 005: Fix data types and create all tables with UUID consistency
\i 005_fix_data_types.sql

-- Create migration tracking table
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Record migrations
INSERT INTO schema_migrations (version) VALUES 
    ('005_fix_data_types')
ON CONFLICT (version) DO NOTHING;

-- Commit transaction
COMMIT;

-- Display migration status
SELECT version, applied_at FROM schema_migrations ORDER BY applied_at;
