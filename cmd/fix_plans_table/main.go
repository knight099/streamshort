package main

import (
	"fmt"
	"log"
	"os"
	"episodd/config"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(".env.local"); err == nil {
		log.Println("Loaded environment from .env.local")
	}
	_ = godotenv.Load()

	// Initialize database
	svcs := config.InitServices()
	db := svcs.DB

	// Drop and recreate subscription_plans table with correct ID type
	dropSQL := `DROP TABLE IF EXISTS subscription_plans CASCADE;`

	createSQL := `
CREATE TABLE subscription_plans (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    duration INTEGER NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'INR',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

INSERT INTO subscription_plans (id, name, description, duration, amount, currency, is_active)
VALUES 
    ('all_access_30d', 'All-Access Monthly', 'Access to all series across all creators for 30 days', 30, 299.00, 'INR', true),
    ('all_access_365d', 'All-Access Yearly', 'Access to all series across all creators for 365 days', 365, 2999.00, 'INR', true);
`

	if err := db.Exec(dropSQL).Error; err != nil {
		log.Fatalf("Failed to drop subscription_plans table: %v", err)
	}

	if err := db.Exec(createSQL).Error; err != nil {
		log.Fatalf("Failed to create and seed subscription_plans table: %v", err)
	}

	fmt.Println("✅ Successfully recreated subscription_plans table with correct ID type")
	fmt.Println("✅ Seeded 2 default subscription plans")

	os.Exit(0)
}
