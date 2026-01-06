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

	// Check if subscriptions table exists
	var exists bool
	err := db.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'subscriptions')").Scan(&exists).Error
	if err != nil {
		log.Fatalf("Failed to check if subscriptions table exists: %v", err)
	}

	fmt.Printf("Subscriptions table exists: %v\n", exists)

	// Check if subscription_plans table exists
	err = db.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'subscription_plans')").Scan(&exists).Error
	if err != nil {
		log.Fatalf("Failed to check if subscription_plans table exists: %v", err)
	}

	fmt.Printf("Subscription plans table exists: %v\n", exists)

	// List all tables
	var tables []string
	err = db.Raw("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name").Scan(&tables).Error
	if err != nil {
		log.Fatalf("Failed to list tables: %v", err)
	}

	fmt.Println("\nAll tables in public schema:")
	for _, table := range tables {
		fmt.Printf("  - %s\n", table)
	}

	os.Exit(0)
}
