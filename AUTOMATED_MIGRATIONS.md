# Automated Database Migrations Guide

This guide explains how to automate database migrations in Streamshort, eliminating the need to manually write SQL files.

## 🎯 **Overview**

Instead of manually creating SQL migration files, you can now:

1. **Define your models in Go** with GORM tags
2. **Automatically generate SQL** from your models
3. **Use GORM AutoMigrate** for instant schema updates
4. **Generate migration files** on-demand

## 🚀 **Option 1: GORM AutoMigrate (Recommended for Development)**

### **How It Works**
GORM automatically creates/updates database tables based on your Go struct definitions.

### **Benefits**
- ✅ **Zero SQL files needed**
- ✅ **Instant schema updates**
- ✅ **Automatic type mapping**
- ✅ **Index and constraint handling**

### **Usage**
```bash
# Just start the server - tables are created automatically
export DATABASE_URL="your-database-url"
go run main.go
```

### **What Happens**
1. Server starts
2. GORM connects to database
3. Tables are automatically created/updated
4. Schema matches your Go models exactly

## 📝 **Option 2: Generate SQL from Go Models**

### **How It Works**
A code generator analyzes your Go structs and creates SQL migration files automatically.

### **Benefits**
- ✅ **SQL files for review**
- ✅ **Version control friendly**
- ✅ **Production deployment ready**
- ✅ **Customizable templates**

### **Usage**
```bash
# Generate migration from current models
go run cmd/generate/main.go "007_new_feature" "Add new feature tables"

# This creates:
# - migrations/007_new_feature_auto_generated.sql
# - migrations/007_new_feature_auto_generated.go
```

### **Generated SQL Example**
```sql
-- Migration: 007_new_feature_auto_generated.sql
-- Description: Add new feature tables
-- Created: 2025-08-16
-- Auto-generated from Go models

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    phone VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
```

## 🔧 **How to Use**

### **1. Quick Start (AutoMigrate)**
```bash
# Set your database URL
export DATABASE_URL="postgresql://user:pass@host:5432/db"

# Start server - tables created automatically
go run main.go
```

### **2. Generate and Review (SQL Files)**
```bash
# Generate migration
go run cmd/generate/main.go "008_user_profiles" "Add user profile fields"

# Review generated SQL
cat migrations/008_user_profiles_auto_generated.sql

# Apply manually if needed
psql $DATABASE_URL -f migrations/008_user_profiles_auto_generated.sql
```

### **3. Interactive Script**
```bash
# Use the automated script
./scripts/auto_migrate.sh
```

## 🏗️ **Model-Driven Development**

### **Define Your Model**
```go
type UserProfile struct {
    ID          string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    UserID      string         `json:"user_id" gorm:"type:uuid;not null;uniqueIndex"`
    Bio         string         `json:"bio" gorm:"type:text"`
    Avatar      string         `json:"avatar" gorm:"type:varchar(500)"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}
```

### **Automatic SQL Generation**
The generator will create:
```sql
CREATE TABLE IF NOT EXISTS user_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE,
    bio TEXT,
    avatar VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_user_profiles_deleted_at ON user_profiles(deleted_at);
CREATE INDEX IF NOT EXISTS idx_user_profiles_user_id ON user_profiles(user_id);
```

## 📊 **Type Mapping**

### **Go → SQL Type Mapping**
| Go Type | SQL Type | GORM Tag Override |
|---------|----------|-------------------|
| `string` | `VARCHAR(255)` | `type:uuid` → `UUID` |
| `int` | `INTEGER` | `type:bigint` → `BIGINT` |
| `int64` | `BIGINT` | `type:smallint` → `SMALLINT` |
| `float64` | `DECIMAL(10,2)` | `type:real` → `REAL` |
| `bool` | `BOOLEAN` | `type:bit` → `BIT` |
| `time.Time` | `TIMESTAMP WITH TIME ZONE` | `type:date` → `DATE` |

### **Constraint Mapping**
| GORM Tag | SQL Constraint |
|-----------|----------------|
| `primaryKey` | `PRIMARY KEY` |
| `not null` | `NOT NULL` |
| `unique` | `UNIQUE` |
| `default:value` | `DEFAULT value` |
| `index` | `CREATE INDEX` |
| `uniqueIndex` | `CREATE UNIQUE INDEX` |

## 🎨 **Customization**

### **Custom Field Names**
```go
type User struct {
    ID       string `gorm:"column:user_id;primaryKey;type:uuid"`
    Username string `gorm:"column:display_name;not null"`
}
```

### **Custom Table Names**
```go
func (User) TableName() string {
    return "app_users"
}
```

### **Custom Indexes**
```go
type User struct {
    ID       string `gorm:"primaryKey;type:uuid"`
    Email    string `gorm:"uniqueIndex:idx_users_email"`
    Username string `gorm:"uniqueIndex:idx_users_username"`
}
```

## 🚀 **Workflow Examples**

### **Development Workflow**
```bash
# 1. Update your Go models
# 2. Start server - tables auto-created
go run main.go

# 3. Test your changes
curl http://localhost:8080/api/test
```

### **Production Workflow**
```bash
# 1. Update Go models
# 2. Generate migration
go run cmd/generate/main.go "009_production_update" "Production schema update"

# 3. Review generated SQL
cat migrations/009_production_update_auto_generated.sql

# 4. Apply to production
psql $PROD_DATABASE_URL -f migrations/009_production_update_auto_generated.sql
```

### **Team Collaboration**
```bash
# 1. Pull latest models
git pull origin main

# 2. Generate migration for your changes
go run cmd/generate/main.go "010_team_feature" "Team collaboration feature"

# 3. Commit both Go models and generated SQL
git add models/ cmd/ migrations/
git commit -m "Add team collaboration feature"
git push origin feature/team-collab
```

## 🔍 **Best Practices**

### **1. Model Design**
- Use descriptive field names
- Add proper GORM tags
- Include validation tags
- Document complex relationships

### **2. Migration Strategy**
- **Development**: Use AutoMigrate for speed
- **Staging**: Generate SQL for review
- **Production**: Generate SQL for deployment

### **3. Version Control**
- Commit Go models first
- Generate migrations from models
- Review generated SQL
- Commit both models and migrations

### **4. Testing**
- Test with AutoMigrate first
- Generate SQL for production
- Test generated SQL in staging
- Deploy to production

## 🛠️ **Troubleshooting**

### **Common Issues**

#### **1. Type Mismatches**
```bash
# Problem: UUID vs BIGINT mismatch
# Solution: Ensure consistent types in models
type User struct {
    ID string `gorm:"primaryKey;type:uuid"` // Use UUID consistently
}
```

#### **2. Missing Extensions**
```bash
# Problem: uuid_generate_v4() not found
# Solution: Enable UUID extension in migration
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```

#### **3. Constraint Violations**
```bash
# Problem: Foreign key constraint fails
# Solution: Check table creation order and types
```

### **Debug Commands**
```bash
# Check current schema
psql $DATABASE_URL -c "\dt"

# Check table structure
psql $DATABASE_URL -c "\d+ table_name"

# Check migration status
go run cmd/migrate/main.go -action status

# Generate fresh migration
go run cmd/generate/main.go "debug" "Debug migration"
```

## 🎉 **Benefits Summary**

### **For Developers**
- 🚀 **Faster development** - no SQL writing
- 🔄 **Automatic sync** - models ↔ database
- 🐛 **Fewer errors** - type safety
- 📚 **Better documentation** - models are self-documenting

### **For Teams**
- 🤝 **Consistent schemas** - models drive everything
- 📝 **Version control** - track schema changes
- 🔍 **Code review** - review models, not SQL
- 🚀 **Faster deployments** - automated migrations

### **For Production**
- 🛡️ **Safer deployments** - reviewed migrations
- 📊 **Audit trail** - track all schema changes
- 🔄 **Rollback support** - versioned migrations
- 🚀 **Zero downtime** - automated schema updates

## 🚀 **Get Started**

### **1. Try AutoMigrate**
```bash
export DATABASE_URL="your-database-url"
go run main.go
```

### **2. Generate Your First Migration**
```bash
go run cmd/generate/main.go "001_my_feature" "Add my feature"
```

### **3. Use the Interactive Script**
```bash
./scripts/auto_migrate.sh
```

### **4. Customize for Your Needs**
- Modify templates in `cmd/generate/main.go`
- Add custom type mappings
- Extend constraint handling
- Create custom generators

---

**🎯 The future of database migrations is model-driven!** 

No more manual SQL writing - just define your models in Go and let the automation handle the rest! 🚀
