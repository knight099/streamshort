package main

import (
	"fmt"
	"log"
	"os"
	"time"

	// "github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectDB() (*gorm.DB, error){
	dsn := os.Getenv("NEON_DATABASE_URL") // e.g. postgres://user:pass@ep-xxx-yyy.ap-southeast-1.aws.neon.tech/db?sslmode=require
	if dsn == "" {
		log.Fatal("NEON_DATABASE_URL is not set")
	}


	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Neon: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB from GORM: %w", err)
	}
	sqlDB.SetMaxIdleConns(5)                   // keep low for Neon
	sqlDB.SetMaxOpenConns(20)                  // Neon connection limit friendly
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // refresh connections
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)  // drop idle connections

	log.Println("✅ Connected to Neon PostgreSQL")
	return db, nil
}


// AutoMigrate runs GORM migrations for provided models
// Usage: AutoMigrate(db, &User{}, &Product{})
func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	log.Println("📦 Running database migrations...")
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	log.Println("✅ Migrations completed successfully")
	return nil
}