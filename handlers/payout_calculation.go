package handlers

import (
	"log"
	"time"

	"gorm.io/gorm"
)

type PayoutHandler struct {
	db *gorm.DB
}

func NewPayoutHandler(db *gorm.DB) *PayoutHandler {
	return &PayoutHandler{db: db}
}

// CalculateMonthlyPayouts triggers the calculation for a specific month
// Ideally run this as a cron job or admin triggered endpoint
func (h *PayoutHandler) CalculateMonthlyPayouts(month time.Time) error {
	// 1. Identify valid subscriptions for this month
	// (Simplified: In a real system, you'd check active subs covering this month)
	// For this logic, we assume we have a table 'subscription_revenues' populated when payment is made.
	// If not, we would query active subscriptions and insert into subscription_revenues first.
	
	tx := h.db.Begin()

	// 2. Iterate through each User who has distributable revenue
	type UserRevenue struct {
		UserID        string
		Distributable float64
	}

	var users []UserRevenue
	if err := tx.Raw(`
		SELECT user_id, SUM(distributable) as distributable
		FROM subscription_revenues
		WHERE month_date = ? AND is_distributed = FALSE
		GROUP BY user_id
	`, month).Scan(&users).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, u := range users {
		// 3. Get total watch time for this user in this month
		var totalWatchTime int64
		if err := tx.Raw(`
			SELECT SUM(watch_time_seconds) 
			FROM monthly_viewer_stats 
			WHERE user_id = ? AND month_date = ?
		`, u.UserID, month).Scan(&totalWatchTime).Error; err != nil {
			log.Printf("Error calculating total watch time for user %s: %v", u.UserID, err)
			continue
		}

		if totalWatchTime == 0 {
			// User paid but watched nothing. 
			// Business decision: Keep 100%? Rollover? For now, platform keeps it (unclaimed).
			continue
		}

		// 4. Get breakdown per creator
		type CreatorShare struct {
			CreatorID        string
			WatchTimeSeconds int64
		}
		var creators []CreatorShare
		if err := tx.Raw(`
			SELECT creator_id, watch_time_seconds
			FROM monthly_viewer_stats
			WHERE user_id = ? AND month_date = ?
		`, u.UserID, month).Scan(&creators).Error; err != nil {
			continue
		}

		// 5. Distribute Revenue
		for _, c := range creators {
			shareFraction := float64(c.WatchTimeSeconds) / float64(totalWatchTime)
			payoutAmount := u.Distributable * shareFraction

			// Upsert into CreatorPayout
			if err := tx.Exec(`
				INSERT INTO creator_payouts (id, creator_id, month_date, total_earnings, created_at, updated_at)
				VALUES (gen_random_uuid(), ?, ?, ?, NOW(), NOW())
				ON CONFLICT (creator_id, month_date) -- Assuming unique constraint on (creator, month)
				DO UPDATE SET
					total_earnings = creator_payouts.total_earnings + EXCLUDED.total_earnings,
					updated_at = NOW()
			`, c.CreatorID, month, payoutAmount).Error; err != nil {
				log.Printf("Failed to record payout for creator %s: %v", c.CreatorID, err)
			}
		}

		// 6. Mark revenue as distributed
		if err := tx.Exec(`
			UPDATE subscription_revenues 
			SET is_distributed = TRUE, updated_at = NOW()
			WHERE user_id = ? AND month_date = ?
		`, u.UserID, month).Error; err != nil {
			log.Printf("Failed to mark revenue distributed for user %s: %v", u.UserID, err)
		}
	}

	return tx.Commit().Error
}

