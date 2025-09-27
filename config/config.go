package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Port                    string
	DatabaseURL             string
	SkipMigrations          bool
	RedisURL                string
	FirebaseAPIKey          string
	FirebaseProjectID       string
	RecaptchaSiteKey        string
	CreatorRPMUSDPer1000Min float64
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	config := &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/streamshort?sslmode=disable"),
		// Support both SKIP_MIGRATION and SKIP_MIGRATIONS (prefer singular if set)
		SkipMigrations: func() bool {
			singular := os.Getenv("SKIP_MIGRATION")
			if singular != "" {
				return singular == "true"
			}
			return getEnv("SKIP_MIGRATIONS", "false") == "true"
		}(),
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379/0"),
		FirebaseAPIKey:    getEnv("FIREBASE_API_KEY", ""),
		FirebaseProjectID: getEnv("FIREBASE_PROJECT_ID", ""),
		RecaptchaSiteKey:  getEnv("RECAPTCHA_SITE_KEY", ""),
	}

	// Parse earnings RPM (USD per 1000 watch-minutes). Default 1.0
	rpmStr := getEnv("CREATOR_RPM_USD_PER_1000_MIN", "1.0")
	if v, err := strconv.ParseFloat(rpmStr, 64); err == nil {
		config.CreatorRPMUSDPer1000Min = v
	} else {
		log.Printf("Warning: invalid CREATOR_RPM_USD_PER_1000_MIN=%s, using default 1.0", rpmStr)
		config.CreatorRPMUSDPer1000Min = 1.0
	}

	return config
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
