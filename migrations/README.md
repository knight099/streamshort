# Database Migrations

This directory contains SQL migration files for the Streamshort database schema.

## Migration Files

### 001_create_users_table.sql
Creates the `users` table for phone-based authentication:
- `id`: Primary key (BIGSERIAL)
- `phone`: Phone number in E.164 format (UNIQUE)
- `created_at`, `updated_at`: Timestamps
- `deleted_at`: Soft delete timestamp

### 002_create_otp_transactions_table.sql
Creates the `otp_transactions` table for OTP verification:
- `id`: Primary key (BIGSERIAL)
- `txn_id`: Unique transaction ID
- `phone`: Phone number for OTP
- `otp`: One-time password (6-digit code)
- `expires_at`: OTP expiration timestamp
- `used`: Whether OTP has been used
- `created_at`, `updated_at`: Timestamps
- `deleted_at`: Soft delete timestamp

### 003_create_refresh_tokens_table.sql
Creates the `refresh_tokens` table for JWT token refresh:
- `id`: Primary key (BIGSERIAL)
- `token`: Unique refresh token string
- `user_id`: Foreign key to users table
- `expires_at`: Token expiration timestamp
- `revoked`: Whether token has been revoked
- `created_at`, `updated_at`: Timestamps
- `deleted_at`: Soft delete timestamp

## Running Migrations

### Option 1: Using the CLI Tool

```bash
# Run all pending migrations
go run cmd/migrate/main.go -action migrate

# Check migration status
go run cmd/migrate/main.go -action status

# Rollback all migrations (WARNING: drops all data)
go run cmd/migrate/main.go -action rollback
```

### Option 2: Using psql directly

```bash
# Run all migrations
psql -d your_database -f migrations/migrate.sql

# Rollback all migrations
psql -d your_database -f migrations/rollback.sql
```

### Option 3: Individual migrations

```bash
# Run migrations one by one
psql -d your_database -f migrations/001_create_users_table.sql
psql -d your_database -f migrations/002_create_otp_transactions_table.sql
psql -d your_database -f migrations/003_create_refresh_tokens_table.sql
```

## Environment Variables

Set the `DATABASE_URL` environment variable:

```bash
export DATABASE_URL="postgres://username:password@your-neon-host:5432/streamshort?sslmode=require"
```

## Migration Tracking

The migrations are tracked in the `schema_migrations` table:

```sql
CREATE TABLE schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

## Indexes

The migrations create the following indexes for optimal performance:

### Users Table
- `idx_users_deleted_at`: For soft delete queries
- `idx_users_phone`: For phone number lookups

### OTP Transactions Table
- `idx_otp_transactions_deleted_at`: For soft delete queries
- `idx_otp_transactions_txn_id`: For transaction ID lookups
- `idx_otp_transactions_phone`: For phone number lookups
- `idx_otp_transactions_expires_at`: For expiration queries
- `idx_otp_transactions_used`: For used status queries
- `idx_otp_transactions_phone_otp_used_expires`: Composite index for OTP verification

### Refresh Tokens Table
- `idx_refresh_tokens_deleted_at`: For soft delete queries
- `idx_refresh_tokens_token`: For token lookups
- `idx_refresh_tokens_user_id`: For user ID lookups
- `idx_refresh_tokens_expires_at`: For expiration queries
- `idx_refresh_tokens_revoked`: For revoked status queries
- `idx_refresh_tokens_token_revoked_expires`: Composite index for token validation

## Foreign Key Constraints

- `refresh_tokens.user_id` → `users.id` (CASCADE DELETE)

## Notes

- All tables use soft deletes with `deleted_at` timestamp
- Timestamps use `TIMESTAMP WITH TIME ZONE` for proper timezone handling
- Phone numbers are stored as VARCHAR(20) to accommodate E.164 format
- OTP codes are stored as VARCHAR(10) for flexibility
- Refresh tokens are stored as VARCHAR(255) for security
