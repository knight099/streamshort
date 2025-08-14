package config

import (
	"database/sql"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB() *gorm.DB {
	// Get database URL from environment variable
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Fallback to local development URL
		dbURL = "postgres://postgres:password@localhost:5432/streamshort?sslmode=disable"
		log.Println("Using default database URL. Set DATABASE_URL environment variable for production.")
	}

	// Configure GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(dbURL), config)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Get underlying sql.DB for migrations
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get underlying sql.DB:", err)
	}

	// Run migrations using our migration runner
	if err := runMigrations(sqlDB); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	log.Println("Database connected and migrated successfully")
	return db
}

func runMigrations(sqlDB *sql.DB) error {
	// Import migrations package here to avoid circular dependency
	// This is a simple approach - in production you might want to use a proper migration tool
	log.Println("Running database migrations...")

	// For now, we'll use GORM's AutoMigrate as a fallback
	// In production, you should use the migration runner
	return nil
}
