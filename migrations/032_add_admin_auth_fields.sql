-- Migration: Add admin authentication fields to users table
-- Adds role, username, and password_hash for admin authentication

-- Add role column with default 'user'
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'user';

-- Add username column for admin login (nullable, unique)
ALTER TABLE users ADD COLUMN IF NOT EXISTS username TEXT;

-- Add password_hash for admin authentication (nullable)
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;

-- Create unique index on username (partial - only for non-null values)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_unique 
ON users (username) 
WHERE username IS NOT NULL;

-- Add index on role for filtering
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);

-- Add check constraint for role values
DO $$ BEGIN
    ALTER TABLE users ADD CONSTRAINT chk_users_role 
    CHECK (role IN ('user', 'admin'));
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Make phone nullable (was NOT NULL, needs to be nullable for admin accounts)
ALTER TABLE users ALTER COLUMN phone DROP NOT NULL;
