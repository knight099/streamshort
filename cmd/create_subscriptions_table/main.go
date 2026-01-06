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

	// Create subscriptions table
	createTableSQL := `
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('series', 'creator', 'global')),
    razorpay_subscription_id VARCHAR(255),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'cancelled', 'expired')),
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    auto_renew BOOLEAN DEFAULT true,
    plan_id VARCHAR(255) NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'INR',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_subscriptions_deleted_at ON subscriptions(deleted_at);
`

	if err := db.Exec(createTableSQL).Error; err != nil {
		log.Fatalf("Failed to create subscriptions table: %v", err)
	}

	fmt.Println("✅ Successfully created subscriptions table and indexes")

	// Verify table exists
	var exists bool
	err := db.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'subscriptions')").Scan(&exists).Error
	if err != nil {
		log.Fatalf("Failed to verify subscriptions table: %v", err)
	}

	fmt.Printf("✅ Subscriptions table exists: %v\n", exists)

	os.Exit(0)
}
