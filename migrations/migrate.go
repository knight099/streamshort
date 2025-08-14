package migrations

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Migration struct {
	Version   string
	Filename  string
	AppliedAt time.Time
}

type MigrationRunner struct {
	db *sql.DB
}

func NewMigrationRunner(db *sql.DB) *MigrationRunner {
	return &MigrationRunner{db: db}
}

// RunMigrations executes all pending migrations
func (mr *MigrationRunner) RunMigrations() error {
	// Create migrations table if it doesn't exist
	if err := mr.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := mr.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Get all migration files
	files, err := mr.getMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to get migration files: %w", err)
	}

	// Find pending migrations
	pending := mr.getPendingMigrations(files, applied)

	if len(pending) == 0 {
		log.Println("No pending migrations")
		return nil
	}

	// Run pending migrations
	for _, migration := range pending {
		if err := mr.runMigration(migration); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", migration.Version, err)
		}
		log.Printf("Applied migration: %s", migration.Version)
	}

	return nil
}

func (mr *MigrationRunner) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := mr.db.Exec(query)
	return err
}

func (mr *MigrationRunner) getAppliedMigrations() (map[string]Migration, error) {
	query := `SELECT version, applied_at FROM schema_migrations ORDER BY applied_at`
	rows, err := mr.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]Migration)
	for rows.Next() {
		var migration Migration
		err := rows.Scan(&migration.Version, &migration.AppliedAt)
		if err != nil {
			return nil, err
		}
		applied[migration.Version] = migration
	}
	return applied, nil
}

func (mr *MigrationRunner) getMigrationFiles() ([]Migration, error) {
	// Get current directory
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Look for migration files in migrations directory
	migrationsDir := filepath.Join(dir, "migrations")
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, err
	}

	var migrations []Migration
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		// Extract version from filename (e.g., "001_create_users_table.sql" -> "001_create_users_table")
		version := strings.TrimSuffix(file.Name(), ".sql")
		if !strings.HasPrefix(version, "00") {
			continue // Skip non-migration files
		}

		migrations = append(migrations, Migration{
			Version:  version,
			Filename: file.Name(),
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func (mr *MigrationRunner) getPendingMigrations(files []Migration, applied map[string]Migration) []Migration {
	var pending []Migration
	for _, file := range files {
		if _, exists := applied[file.Version]; !exists {
			pending = append(pending, file)
		}
	}
	return pending
}

func (mr *MigrationRunner) runMigration(migration Migration) error {
	// Read migration file
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	filepath := filepath.Join(dir, "migrations", migration.Filename)
	content, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read migration file %s: %w", migration.Filename, err)
	}

	// Start transaction
	tx, err := mr.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute migration
	_, err = tx.Exec(string(content))
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
	}

	// Record migration
	_, err = tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", migration.Version)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
	}

	// Commit transaction
	return tx.Commit()
}

// GetMigrationStatus returns the status of all migrations
func (mr *MigrationRunner) GetMigrationStatus() ([]Migration, error) {
	applied, err := mr.getAppliedMigrations()
	if err != nil {
		return nil, err
	}

	files, err := mr.getMigrationFiles()
	if err != nil {
		return nil, err
	}

	var status []Migration
	for _, file := range files {
		if applied, exists := applied[file.Version]; exists {
			status = append(status, applied)
		} else {
			status = append(status, file)
		}
	}

	return status, nil
}
