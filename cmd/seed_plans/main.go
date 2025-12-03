package main

import (
	"fmt"
	"log"
	"os"
	"streamshort/config"
	"streamshort/models"

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

	// Seed subscription plans
	plans := []models.SubscriptionPlan{
		{
			ID:          "all_access_30d",
			Name:        "All-Access Monthly",
			Description: "Access to all series across all creators for 30 days",
			Duration:    30,
			Amount:      299.00,
			Currency:    "INR",
			IsActive:    true,
		},
		{
			ID:          "all_access_365d",
			Name:        "All-Access Yearly",
			Description: "Access to all series across all creators for 365 days",
			Duration:    365,
			Amount:      2999.00,
			Currency:    "INR",
			IsActive:    true,
		},
	}

	for _, plan := range plans {
		// Check if plan exists
		var existing models.SubscriptionPlan
		err := db.Where("id = ?", plan.ID).First(&existing).Error
		if err != nil {
			// Create new plan
			if err := db.Create(&plan).Error; err != nil {
				log.Printf("Failed to create plan %s: %v", plan.Name, err)
			} else {
				fmt.Printf("✅ Created plan: %s (%.2f %s for %d days)\n", plan.Name, plan.Amount, plan.Currency, plan.Duration)
			}
		} else {
			// Update existing to match new definition and ensure active
			updates := map[string]interface{}{
				"name":        plan.Name,
				"description": plan.Description,
				"duration":    plan.Duration,
				"amount":      plan.Amount,
				"currency":    plan.Currency,
				"is_active":   true,
			}
			if err := db.Model(&existing).Updates(updates).Error; err != nil {
				log.Printf("Failed to update plan %s: %v", plan.Name, err)
			} else {
				fmt.Printf("🔄 Updated plan: %s\n", plan.Name)
			}
		}
	}

	fmt.Println("\n✅ Subscription plans seeded successfully")

	os.Exit(0)
}
