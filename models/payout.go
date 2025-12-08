package models

import (
	"time"
	// "gorm.io/gorm"
)

// MonthlyViewerStats tracks how much time a specific user watched a specific creator in a given month
type MonthlyViewerStats struct {
	ID               string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID           string    `json:"user_id" gorm:"type:uuid;not null;index:idx_viewer_stats_user_date"`
	CreatorID        string    `json:"creator_id" gorm:"type:uuid;not null;index:idx_viewer_stats_creator_date"`
	MonthDate        time.Time `json:"month_date" gorm:"type:date;not null;index:idx_viewer_stats_user_date;index:idx_viewer_stats_creator_date"` // Stores the 1st of the month (e.g., 2023-10-01)
	WatchTimeSeconds int64     `json:"watch_time_seconds" gorm:"default:0"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CreatorPayout represents a calculated payout for a creator for a specific month
type CreatorPayout struct {
	ID             string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CreatorID      string     `json:"creator_id" gorm:"type:uuid;not null;index"`
	MonthDate      time.Time  `json:"month_date" gorm:"type:date;not null"` // The month being paid out for
	TotalEarnings  float64    `json:"total_earnings" gorm:"type:decimal(10,2);not null"`
	Status         string     `json:"status" gorm:"type:varchar(20);default:'pending';check:status IN ('pending', 'processing', 'paid', 'failed')"`
	ProcessedAt    *time.Time `json:"processed_at,omitempty"`
	TransactionRef string     `json:"transaction_ref,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// SubscriptionRevenue tracks the revenue from a user's subscription to be distributed
type SubscriptionRevenue struct {
	ID             string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID         string    `json:"user_id" gorm:"type:uuid;not null;index"`
	SubscriptionID string    `json:"subscription_id" gorm:"type:uuid;not null"`
	MonthDate      time.Time `json:"month_date" gorm:"type:date;not null"`
	Amount         float64   `json:"amount" gorm:"type:decimal(10,2);not null"`        // Total paid by user
	Distributable  float64   `json:"distributable" gorm:"type:decimal(10,2);not null"` // The 70% pot
	PlatformFee    float64   `json:"platform_fee" gorm:"type:decimal(10,2);not null"`  // The 30% fee
	IsDistributed  bool      `json:"is_distributed" gorm:"default:false"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (MonthlyViewerStats) TableName() string {
	return "monthly_viewer_stats"
}

func (CreatorPayout) TableName() string {
	return "creator_payouts"
}

func (SubscriptionRevenue) TableName() string {
	return "subscription_revenues"
}
