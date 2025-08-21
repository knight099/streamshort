package config

import (
	"log"
	"strings"

	"streamshort/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB() *gorm.DB {
	// Load configuration
	cfg := LoadConfig()

	// Get database URL from configuration
	dbURL := cfg.DatabaseURL
	log.Println("Database URL loaded from configuration")

	if dbURL == "" {
		// Fallback to local development URL
		dbURL = "postgres://postgres:password@localhost:5432/streamshort?sslmode=disable"
		log.Println("Using default database URL. Set DATABASE_URL environment variable for production.")
	}

	// Ensure DSN has Neon-friendly flags
	lower := strings.ToLower(dbURL)
	if !strings.Contains(lower, "prefer_simple_protocol") {
		if strings.Contains(dbURL, "?") {
			dbURL += "&prefer_simple_protocol=true"
		} else {
			dbURL += "?prefer_simple_protocol=true"
		}
	}
	if !strings.Contains(lower, "search_path=") {
		if strings.Contains(dbURL, "?") {
			dbURL += "&search_path=public"
		} else {
			dbURL += "?search_path=public"
		}
	}

	// Configure GORM
	config := &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Info),
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(dbURL), config)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Ensure required extensions exist
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto;").Error; err != nil {
		log.Printf("Warning: failed to create extension pgcrypto: %v", err)
	}
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";").Error; err != nil {
		log.Printf("Warning: failed to create extension uuid-ossp: %v", err)
	}

	// Check if migrations should be skipped
	skipMigrations := cfg.SkipMigrations

	if skipMigrations {
		log.Println("Skipping database migrations (SKIP_MIGRATIONS=true)")
	} else {
		// Auto-migrate all models (automatically creates/updates tables)
		log.Println("Running database auto-migration...")

		// Migrate models one by one to handle errors gracefully
		modelsToMigrate := []interface{}{
			&models.User{},
			&models.OTPTransaction{},
			&models.RefreshToken{},
			&models.CreatorProfile{},
			&models.PayoutDetails{},
			&models.CreatorAnalytics{},
			&models.Series{},
			&models.Episode{},
			&models.UploadRequest{},
			// Engagement models
			&models.EpisodeLike{},
			&models.EpisodeRating{},
			&models.EpisodeComment{},
		}

		for _, model := range modelsToMigrate {
			if err := db.AutoMigrate(model); err != nil {
				log.Printf("Warning: Failed to migrate model %T: %v", model, err)
				// Continue with other models instead of failing completely
			} else {
				log.Printf("Successfully migrated model %T", model)
			}
		}

		// Hard guarantee: ensure content tables exist even if AutoMigrate hit benign index errors
		if !db.Migrator().HasTable(&models.Series{}) {
			if err := db.Migrator().CreateTable(&models.Series{}); err != nil {
				log.Printf("Warning: failed to create table for models.Series explicitly: %v", err)
			}
		}
		if !db.Migrator().HasTable(&models.Episode{}) {
			if err := db.Migrator().CreateTable(&models.Episode{}); err != nil {
				log.Printf("Warning: failed to create table for models.Episode explicitly: %v", err)
			}
		}
		if !db.Migrator().HasTable(&models.UploadRequest{}) {
			if err := db.Migrator().CreateTable(&models.UploadRequest{}); err != nil {
				log.Printf("Warning: failed to create table for models.UploadRequest explicitly: %v", err)
			}
		}
	}

	log.Println("Database connected and auto-migrated successfully.")
	return db
}
