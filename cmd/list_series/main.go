package main

import (
	"fmt"
	"log"
	"os"
	"episodd/config"
	"episodd/models"

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

	// Get all series
	var series []models.Series
	if err := db.Find(&series).Error; err != nil {
		log.Fatalf("Failed to fetch series: %v", err)
	}

	if len(series) == 0 {
		fmt.Println("❌ No series found in database")
		fmt.Println("\nTo create a test series, you can:")
		fmt.Println("1. Use the creator API to create a series")
		fmt.Println("2. Or use a UUID generator like: 00000000-0000-0000-0000-000000000001")
	} else {
		fmt.Printf("✅ Found %d series in database:\n\n", len(series))
		for i, s := range series {
			fmt.Printf("%d. ID: %s\n", i+1, s.ID)
			fmt.Printf("   Title: %s\n", s.Title)
			fmt.Printf("   Price Type: %s\n", s.PriceType)
			fmt.Println()
		}

		if len(series) > 0 {
			fmt.Printf("💡 You can use this series ID for testing: %s\n", series[0].ID)
		}
	}

	os.Exit(0)
}
