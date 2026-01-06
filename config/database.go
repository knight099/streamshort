package config

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"episodd/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Services struct {
	DB               *gorm.DB
	RDB              *redis.Client
	FirebaseAPIKey   string
	RecaptchaSiteKey string
}

func InitDB() *gorm.DB {
	// Load configuration
	cfg := LoadConfig()

	// Get database URL from configuration
	dbURL := cfg.DatabaseURL
	log.Println("Database URL loaded from configuration")

	if dbURL == "" {
		// Fallback to local development URL
		dbURL = "postgres://postgres:password@localhost:5432/episodd?sslmode=disable"
		log.Println("Using default database URL. Set DATABASE_URL environment variable for production.")
	}

	// Ensure DSN has Neon-friendly flags
	lower := strings.ToLower(dbURL)
	if !strings.Contains(lower, "search_path=") {
		if strings.Contains(dbURL, "?") {
			dbURL += "&search_path=public"
		} else {
			dbURL += "?search_path=public"
		}
	}

	// Parse pgx config from database URL
	pgxConfig, err := pgx.ParseConfig(dbURL)
	if err != nil {
		log.Fatal("Failed to parse database URL:", err)
	}

	// CRITICAL: Disable prepared statement caching to prevent
	// "prepared statement name is already in use" errors with Neon's connection pooler
	// Neon uses PgBouncer in transaction mode which doesn't support prepared statements
	pgxConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// Register the pgx driver with stdlib
	connStr := stdlib.RegisterConnConfig(pgxConfig)

	// Configure GORM with PrepareStmt disabled to avoid prepared statement caching issues
	gormConfig := &gorm.Config{
		// Disable SQL logging (set to Error or Silent to avoid query logs)
		Logger:                                   logger.Default.LogMode(logger.Error),
		DisableForeignKeyConstraintWhenMigrating: true,
		// Disable prepared statement caching to prevent connection pool conflicts
		PrepareStmt: false,
	}

	// Connect to database using the registered pgx config
	db, err := gorm.Open(postgres.New(postgres.Config{
		DriverName: "pgx",
		Conn:       nil,
		DSN:        connStr,
	}), gormConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connected with SimpleProtocol mode (prepared statements disabled)")

	// Configure connection pool settings for Neon's serverless PostgreSQL
	// This prevents connection exhaustion and prepared statement conflicts
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Warning: failed to get underlying sql.DB: %v", err)
	} else {
		// Set maximum number of open connections (Neon free tier allows ~20 concurrent)
		sqlDB.SetMaxOpenConns(10)
		// Set maximum number of idle connections (keep low for serverless)
		sqlDB.SetMaxIdleConns(2)
		// Set maximum lifetime for connections (shorter for serverless to recycle faster)
		sqlDB.SetConnMaxLifetime(3 * time.Minute)
		// Set maximum idle time for connections
		sqlDB.SetConnMaxIdleTime(30 * time.Second)
		log.Println("Database connection pool configured: MaxOpenConns=10, MaxIdleConns=2, ConnMaxLifetime=3m")
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
		// Run SQL migrations from migrations/*.sql first, then GORM automigrate for safety
		log.Println("Running database migrations (SQL) and auto-migration...")

		// Best-effort: execute SQL migrations from @migrations directory using the simple runner logic
		if err := runSQLMigrations(db); err != nil {
			log.Printf("Warning: SQL migrations failed: %v", err)
		}

		// Migrate models one by one to handle errors gracefully
		modelsToMigrate := []interface{}{
			&models.User{},
			&models.OTPTransaction{},
			&models.RefreshToken{},
			&models.CreatorProfile{},
			&models.CreatorFollow{},
			&models.PayoutDetails{},
			&models.CreatorAnalytics{},
			&models.Series{},
			&models.Episode{},
			&models.UploadRequest{},
			// Engagement models
			&models.EpisodeLike{},
			&models.SeriesRating{},
			// Subscription models
			&models.Subscription{},
			&models.SubscriptionPlan{},
			// Payment/earnings models
			&models.CreatorEarnings{},
			&models.Payout{},
			&models.SeriesView{},
			&models.UserEpisodeWatch{},
		}

		for _, model := range modelsToMigrate {
			if err := db.AutoMigrate(model); err != nil {
				log.Printf("Warning: Failed to migrate model %T: %v", model, err)
				// Continue with other models instead of failing completely
			} else {
				log.Printf("Successfully migrated model %T", model)
			}
		}

		// Hard guarantee: ensure users.name column exists
		if err := db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS name TEXT;").Error; err != nil {
			log.Printf("Warning: failed to ensure users.name column: %v", err)
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

		// Safeguard: ensure episodes.series_id FK -> series.id exists with index (even if AutoMigrate skips FKs)
		if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_episodes_series_id ON episodes (series_id);").Error; err != nil {
			log.Printf("Warning: failed to ensure index idx_episodes_series_id: %v", err)
		}
		if err := db.Exec(`
		DO $$
		BEGIN
		    IF NOT EXISTS (
		        SELECT 1
		        FROM information_schema.table_constraints tc
		        WHERE tc.constraint_name = 'fk_episodes_series_id'
		          AND tc.table_name = 'episodes'
		          AND tc.constraint_type = 'FOREIGN KEY'
		    ) THEN
		        ALTER TABLE episodes
		        ADD CONSTRAINT fk_episodes_series_id
		        FOREIGN KEY (series_id)
		        REFERENCES series(id)
		        ON DELETE CASCADE;
		    END IF;
		END $$;`).Error; err != nil {
			log.Printf("Warning: failed to ensure FK fk_episodes_series_id: %v", err)
		}

		// Safeguard: ensure follower_count column exists and backfilled
		if err := db.Exec("ALTER TABLE creator_profiles ADD COLUMN IF NOT EXISTS follower_count BIGINT NOT NULL DEFAULT 0;").Error; err != nil {
			log.Printf("Warning: failed to ensure follower_count column: %v", err)
		}
		if err := db.Exec(`
		UPDATE creator_profiles cp
		SET follower_count = sub.cnt
		FROM (
		  SELECT creator_id, COUNT(*)::BIGINT AS cnt
		  FROM creator_follows
		  WHERE deleted_at IS NULL
		  GROUP BY creator_id
		) AS sub
		WHERE cp.id = sub.creator_id;`).Error; err != nil {
			log.Printf("Warning: failed to backfill follower_count: %v", err)
		}

		// Safeguard: ensure foreign keys on creator_follows
		if err := db.Exec(`
		DO $$
		BEGIN
		    IF NOT EXISTS (
		        SELECT 1 FROM information_schema.table_constraints
		        WHERE constraint_name = 'fk_creator_follows_follower' AND table_name = 'creator_follows'
		    ) THEN
		        ALTER TABLE creator_follows
		        ADD CONSTRAINT fk_creator_follows_follower FOREIGN KEY (follower_id)
		        REFERENCES users(id) ON DELETE CASCADE;
		    END IF;
		    IF NOT EXISTS (
		        SELECT 1 FROM information_schema.table_constraints
		        WHERE constraint_name = 'fk_creator_follows_creator' AND table_name = 'creator_follows'
		    ) THEN
		        ALTER TABLE creator_follows
		        ADD CONSTRAINT fk_creator_follows_creator FOREIGN KEY (creator_id)
		        REFERENCES creator_profiles(id) ON DELETE CASCADE;
		    END IF;
		END $$;`).Error; err != nil {
			log.Printf("Warning: failed to ensure FKs on creator_follows: %v", err)
		}
	}

	log.Println("Database connected and auto-migrated successfully.")
	return db
}

// runSQLMigrations executes all .sql files in the migrations directory, in lexicographic order.
// It records applied versions in schema_migrations to ensure idempotency.
func runSQLMigrations(db *gorm.DB) error {
	// Ensure tracking table exists
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
        version VARCHAR(255) PRIMARY KEY,
        applied_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
    );`).Error; err != nil {
		return err
	}

	// Discover migration files
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	dir := cwd + "/migrations"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	type mf struct{ version, path string }
	files := make([]mf, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		// Basic numeric prefix check (e.g., 001_..., 010_...)
		if len(name) < 3 || name[0] < '0' || name[0] > '9' {
			continue
		}
		files = append(files, mf{version: strings.TrimSuffix(name, ".sql"), path: dir + "/" + name})
	}
	// Sort lexicographically
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if files[j].version < files[i].version {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	// Fetch applied versions
	rows, err := db.Raw("SELECT version FROM schema_migrations").Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	applied := map[string]struct{}{}
	for rows.Next() {
		var v string
		_ = rows.Scan(&v)
		applied[v] = struct{}{}
	}

	// Run pending migrations
	for _, f := range files {
		if _, ok := applied[f.version]; ok {
			continue
		}
		content, err := os.ReadFile(f.path)
		if err != nil {
			return err
		}
		tx := db.Begin()
		if err := tx.Exec(string(content)).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", f.version).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit().Error; err != nil {
			return err
		}
		log.Printf("Applied SQL migration: %s", f.version)
	}
	return nil
}

// InitServices initializes DB, Redis, and Elasticsearch clients
func InitServices() *Services {
	cfg := LoadConfig()

	db := InitDB()

	// Initialize Redis
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Printf("Warning: failed to parse REDIS_URL: %v", err)
	}
	var rdb *redis.Client
	if opt != nil {
		rdb = redis.NewClient(opt)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("Warning: failed to connect to Redis: %v", err)
		} else {
			log.Println("Connected to Redis")
		}
	}

	return &Services{DB: db, RDB: rdb, FirebaseAPIKey: cfg.FirebaseAPIKey, RecaptchaSiteKey: cfg.RecaptchaSiteKey}
}
