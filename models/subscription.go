package models

import (
	"time"

	"gorm.io/gorm"
)

// Subscription represents a user's subscription to content
type Subscription struct {
	ID                      string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID                  string         `json:"user_id" gorm:"type:uuid;not null;index:idx_subscriptions_user_id"`
	TargetType              string         `json:"target_type" gorm:"type:varchar(20);not null;check:target_type IN ('series', 'creator')"`
	TargetID                string         `json:"target_id" gorm:"type:uuid;not null;index:idx_subscriptions_target"`
	RazorpaySubscriptionID  *string        `json:"razorpay_subscription_id" gorm:"type:varchar(255)"`
	Status                  string         `json:"status" gorm:"type:varchar(20);default:'active';check:status IN ('active', 'cancelled', 'expired')"`
	StartDate               time.Time      `json:"start_date" gorm:"not null"`
	EndDate                 time.Time      `json:"end_date" gorm:"not null"`
	AutoRenew               bool           `json:"auto_renew" gorm:"default:true"`
	PlanID                  string         `json:"plan_id" gorm:"type:varchar(255);not null"`
	Amount                  float64        `json:"amount" gorm:"type:decimal(10,2);not null"`
	Currency                string         `json:"currency" gorm:"type:varchar(3);default:'INR'"`
	CreatedAt               time.Time      `json:"created_at"`
	UpdatedAt               time.Time      `json:"updated_at"`
	DeletedAt               gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relationships
	User *User `json:"user" gorm:"foreignKey:UserID"`
}

// SubscriptionPlan represents available subscription plans
type SubscriptionPlan struct {
	ID          string  `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name        string  `json:"name" gorm:"type:varchar(100);not null"`
	Description string  `json:"description" gorm:"type:text"`
	Duration    int     `json:"duration" gorm:"type:int;not null"` // Duration in days
	Amount      float64 `json:"amount" gorm:"type:decimal(10,2);not null"`
	Currency    string  `json:"currency" gorm:"type:varchar(3);default:'INR'"`
	IsActive    bool    `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// IsActive checks if the subscription is currently active
func (s *Subscription) IsActive() bool {
	now := time.Now()
	return s.Status == "active" && now.After(s.StartDate) && now.Before(s.EndDate)
}

// IsExpired checks if the subscription has expired
func (s *Subscription) IsExpired() bool {
	return time.Now().After(s.EndDate)
}

// DaysUntilExpiry returns the number of days until the subscription expires
func (s *Subscription) DaysUntilExpiry() int {
	now := time.Now()
	if now.After(s.EndDate) {
		return 0
	}
	duration := s.EndDate.Sub(now)
	return int(duration.Hours() / 24)
}
