package config

import (
	"log"
	"os"

	"streamshort/models"

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

	// Auto-migrate all models (automatically creates/updates tables)
	log.Println("Running database auto-migration...")
	err = db.AutoMigrate(
		&models.User{},
		&models.OTPTransaction{},
		&models.RefreshToken{},
		&models.CreatorProfile{},
		&models.PayoutDetails{},
		&models.CreatorAnalytics{},
		&models.Series{},
		&models.Episode{},
		&models.UploadRequest{},
	)
	if err != nil {
		log.Fatal("Failed to auto-migrate database:", err)
	}

	log.Println("Database connected and auto-migrated successfully.")
	return db
}
