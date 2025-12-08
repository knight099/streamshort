package main

import (
	"log"
	"time"

	"streamshort/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// seedSubscriptionPlans creates default subscription plans
func seedSubscriptionPlans(db *gorm.DB) error {
	log.Println("Seeding subscription plans...")

	plans := []models.SubscriptionPlan{
		{
			ID:          "all_access_30d",
			Name:        "All-Access Monthly",
			Description: "Access to all series across all creators for 30 days",
			Duration:    30,
			Amount:      299.0,
			Currency:    "INR",
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "all_access_365d",
			Name:        "All-Access Yearly",
			Description: "Access to all series across all creators for 365 days",
			Duration:    365,
			Amount:      2999.0,
			Currency:    "INR",
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, plan := range plans {
		if err := db.Where("id = ?", plan.ID).Assign(map[string]interface{}{
			"name":        plan.Name,
			"description": plan.Description,
			"duration":    plan.Duration,
			"amount":      plan.Amount,
			"currency":    plan.Currency,
			"is_active":   plan.IsActive,
			"updated_at":  time.Now(),
		}).FirstOrCreate(&models.SubscriptionPlan{ID: plan.ID}).Error; err != nil {
			log.Printf("Failed to create plan %s: %v", plan.Name, err)
			continue
		}
		log.Printf("Created subscription plan: %s", plan.Name)
	}

	return nil
}

// seedTestSubscriptions creates test subscriptions for existing users
func seedTestSubscriptions(db *gorm.DB) error {
	log.Println("Seeding test subscriptions...")

	// Get existing users
	var users []models.User
	if err := db.Limit(5).Find(&users).Error; err != nil {
		return err
	}

	// Get existing series
	var series []models.Series
	if err := db.Limit(3).Find(&series).Error; err != nil {
		return err
	}

	if len(users) == 0 || len(series) == 0 {
		log.Println("No users or series found, skipping subscription seeding")
		return nil
	}

	// Create test subscriptions
	for i, user := range users {
		if i >= len(series) {
			break
		}

		subscription := models.Subscription{
			ID:         uuid.New().String(),
			UserID:     user.ID,
			TargetType: "series",
			Status:     "active",
			StartDate:  time.Now().AddDate(0, 0, -15), // Started 15 days ago
			EndDate:    time.Now().AddDate(0, 0, 15),  // Expires in 15 days
			AutoRenew:  true,
			PlanID:     "plan_basic_monthly",
			Amount:     30.0,
			Currency:   "INR",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if err := db.Create(&subscription).Error; err != nil {
			log.Printf("Failed to create subscription for user %s: %v", user.ID, err)
			continue
		}
		log.Printf("Created subscription for user %s to series %s", user.ID, series[i].Title)
	}

	return nil
}
