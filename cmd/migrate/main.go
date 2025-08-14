package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"streamshort/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	var (
		dbURL  = flag.String("db", "", "Database URL (e.g., postgres://user:pass@host:port/db)")
		action = flag.String("action", "migrate", "Action to perform: migrate, status, rollback")
	)
	flag.Parse()

	if *dbURL == "" {
		// Try to get from environment variable
		*dbURL = os.Getenv("DATABASE_URL")
		if *dbURL == "" {
			log.Fatal("Database URL is required. Set -db flag or DATABASE_URL environment variable")
		}
	}

	// Connect to database
	db, err := sql.Open("pgx", *dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Create migration runner
	runner := migrations.NewMigrationRunner(db)

	switch *action {
	case "migrate":
		fmt.Println("Running migrations...")
		if err := runner.RunMigrations(); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Println("Migrations completed successfully")

	case "status":
		fmt.Println("Migration status:")
		status, err := runner.GetMigrationStatus()
		if err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}

		for _, migration := range status {
			if migration.AppliedAt.IsZero() {
				fmt.Printf("  [PENDING] %s\n", migration.Version)
			} else {
				fmt.Printf("  [APPLIED] %s (%s)\n", migration.Version, migration.AppliedAt.Format("2006-01-02 15:04:05"))
			}
		}

	case "rollback":
		fmt.Println("WARNING: Rollback will drop all tables and data!")
		fmt.Print("Are you sure? (type 'yes' to confirm): ")

		var confirmation string
		fmt.Scanln(&confirmation)

		if confirmation != "yes" {
			fmt.Println("Rollback cancelled")
			return
		}

		fmt.Println("Rolling back migrations...")
		// Note: This is a simple rollback that drops all tables
		// In production, you might want more sophisticated rollback logic
		queries := []string{
			"DROP TABLE IF EXISTS refresh_tokens CASCADE",
			"DROP TABLE IF EXISTS otp_transactions CASCADE",
			"DROP TABLE IF EXISTS users CASCADE",
			"DROP TABLE IF EXISTS schema_migrations CASCADE",
		}

		for _, query := range queries {
			if _, err := db.Exec(query); err != nil {
				log.Printf("Warning: Failed to execute %s: %v", query, err)
			}
		}
		fmt.Println("Rollback completed")

	default:
		log.Fatalf("Unknown action: %s. Use migrate, status, or rollback", *action)
	}
}
