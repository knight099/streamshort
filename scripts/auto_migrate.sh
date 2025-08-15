#!/bin/bash

# Auto-migration script for Streamshort
# This script demonstrates the automated migration workflow

echo "🚀 Streamshort Auto-Migration Script"
echo "===================================="

# Check if database URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "❌ DATABASE_URL environment variable is not set"
    echo "Please set it first:"
    echo "export DATABASE_URL='your-database-connection-string'"
    exit 1
fi

echo "✅ Database URL is configured"

# Option 1: Use GORM AutoMigrate (Recommended for development)
echo ""
echo "🔄 Option 1: GORM AutoMigrate (Automatic)"
echo "This will automatically create/update tables from your Go models"
echo ""

read -p "Do you want to use GORM AutoMigrate? (y/n): " use_auto

if [ "$use_auto" = "y" ] || [ "$use_auto" = "Y" ]; then
    echo "🚀 Starting server with AutoMigrate..."
    go run main.go
    exit 0
fi

# Option 2: Generate SQL migrations from models
echo ""
echo "📝 Option 2: Generate SQL migrations from Go models"
echo "This will create SQL files that you can review and run manually"
echo ""

read -p "Do you want to generate SQL migrations? (y/n): " generate_sql

if [ "$generate_sql" = "y" ] || [ "$generate_sql" = "Y" ]; then
    echo "📄 Generating SQL migrations from Go models..."
    
    # Generate migration for current models
    go run cmd/generate/main.go "006_auto_generated" "Auto-generated from current Go models"
    
    echo ""
    echo "✅ SQL migrations generated!"
    echo "📁 Check the migrations/ directory for new files"
    echo ""
    echo "To apply the generated migrations:"
    echo "1. Review the generated SQL files"
    echo "2. Run: psql \$DATABASE_URL -f migrations/006_auto_generated_auto_generated.sql"
    echo "3. Or use: go run cmd/migrate/main.go -action migrate"
fi

# Option 3: Manual migration
echo ""
echo "🔧 Option 3: Manual migration"
echo "Use existing migration files"
echo ""

read -p "Do you want to run manual migrations? (y/n): " manual_migrate

if [ "$manual_migrate" = "y" ] || [ "$manual_migrate" = "Y" ]; then
    echo "📊 Checking migration status..."
    go run cmd/migrate/main.go -action status
    
    echo ""
    echo "🚀 Running migrations..."
    go run cmd/migrate/main.go -action migrate
fi

echo ""
echo "🎉 Migration workflow completed!"
echo ""
echo "Next steps:"
echo "1. Start the server: go run main.go"
echo "2. Test the APIs: ./test_creator_api.sh"
echo "3. Check database: psql \$DATABASE_URL -c '\dt'"
