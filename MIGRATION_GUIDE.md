# Database Migration Guide

This guide explains how to use the database migration system for the Streamshort API.

## Overview

The migration system consists of:
- **SQL migration files** in the `migrations/` directory
- **Go migration runner** for programmatic execution
- **CLI tool** for command-line operations
- **Migration tracking** to prevent duplicate executions

## Quick Start

### 1. Set up your database URL

```bash
export DATABASE_URL="postgres://username:password@your-neon-host:5432/streamshort?sslmode=require"
```

### 2. Run migrations

```bash
# Using the CLI tool
go run cmd/migrate/main.go -action migrate

# Or using psql directly
psql -d your_database -f migrations/migrate.sql
```

### 3. Check status

```bash
go run cmd/migrate/main.go -action status
```

## Migration Files

### 001_create_users_table.sql
```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    phone VARCHAR(20) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
```

### 002_create_otp_transactions_table.sql
```sql
CREATE TABLE otp_transactions (
    id BIGSERIAL PRIMARY KEY,
    txn_id VARCHAR(50) NOT NULL UNIQUE,
    phone VARCHAR(20) NOT NULL,
    otp VARCHAR(10) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
```

### 003_create_refresh_tokens_table.sql
```sql
CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    token VARCHAR(255) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

## CLI Commands

### Run Migrations
```bash
go run cmd/migrate/main.go -action migrate
```

### Check Status
```bash
go run cmd/migrate/main.go -action status
```

### Rollback (WARNING: drops all data)
```bash
go run cmd/migrate/main.go -action rollback
```

### Custom Database URL
```bash
go run cmd/migrate/main.go -db "postgres://user:pass@host:5432/db" -action migrate
```

## Integration with GORM

The migration system works alongside GORM's AutoMigrate. The current setup:

1. **Development**: Uses GORM AutoMigrate for quick setup
2. **Production**: Uses SQL migrations for better control and versioning

To use SQL migrations in production, update `config/database.go`:

```go
func runMigrations(sqlDB *sql.DB) error {
    runner := migrations.NewMigrationRunner(sqlDB)
    return runner.RunMigrations()
}
```

## Migration Best Practices

### 1. Always use transactions
Migrations are wrapped in transactions to ensure atomicity.

### 2. Use IF NOT EXISTS
All CREATE statements use `IF NOT EXISTS` to prevent errors on re-runs.

### 3. Add proper indexes
Each migration includes relevant indexes for performance.

### 4. Include comments
Tables and columns have descriptive comments for documentation.

### 5. Use soft deletes
All tables include `deleted_at` for soft delete functionality.

## Performance Considerations

### Indexes Created
- **Users**: `phone`, `deleted_at`
- **OTP Transactions**: `txn_id`, `phone`, `expires_at`, `used`, `deleted_at`
- **Refresh Tokens**: `token`, `user_id`, `expires_at`, `revoked`, `deleted_at`

### Composite Indexes
- `otp_transactions(phone, otp, used, expires_at)` for OTP verification
- `refresh_tokens(token, revoked, expires_at)` for token validation

## Troubleshooting

### Migration Already Applied
If you get "migration already applied" errors, check the `schema_migrations` table:

```sql
SELECT * FROM schema_migrations ORDER BY applied_at;
```

### Rollback Issues
If rollback fails, manually drop tables:

```sql
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS otp_transactions CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS schema_migrations CASCADE;
```

### Connection Issues
Ensure your `DATABASE_URL` is correct and the database is accessible:

```bash
psql $DATABASE_URL -c "SELECT 1;"
```

## Adding New Migrations

1. Create a new SQL file: `migrations/004_your_migration.sql`
2. Follow the naming convention: `001_`, `002_`, etc.
3. Include proper comments and documentation
4. Test with the CLI tool before deploying

## Production Deployment

1. **Backup your database** before running migrations
2. **Test migrations** in a staging environment first
3. **Run migrations** during maintenance windows
4. **Monitor** the migration process
5. **Verify** the schema after migration

## Example Migration Workflow

```bash
# 1. Set database URL
export DATABASE_URL="postgres://user:pass@host:5432/streamshort"

# 2. Check current status
go run cmd/migrate/main.go -action status

# 3. Run migrations
go run cmd/migrate/main.go -action migrate

# 4. Verify status
go run cmd/migrate/main.go -action status

# 5. Test the application
go run main.go
```

## Security Notes

- **Never commit** database credentials to version control
- **Use environment variables** for database URLs
- **Limit database access** to necessary permissions only
- **Monitor** migration logs for any issues
- **Backup** before running migrations in production
