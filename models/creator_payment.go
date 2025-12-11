package models

import (
	"time"

	"gorm.io/gorm"
)

// CreatorEarnings represents an individual earnings transaction for a creator
type CreatorEarnings struct {
	ID             string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CreatorID      string         `json:"creator_id" gorm:"type:uuid;not null;index"`
	Amount         float64        `json:"amount" gorm:"type:decimal(10,2);not null"`
	EarningsType   string         `json:"earnings_type" gorm:"type:varchar(50);not null;check:earnings_type IN ('subscription', 'one_time', 'ad_revenue')"`
	SeriesID       *string        `json:"series_id" gorm:"type:uuid"`
	EpisodeID      *string        `json:"episode_id" gorm:"type:uuid"`
	SubscriptionID *string        `json:"subscription_id" gorm:"type:uuid"`
	Status         string         `json:"status" gorm:"type:varchar(20);default:'pending';check:status IN ('pending', 'paid', 'cancelled')"`
	CreatedAt      time.Time      `json:"created_at"`
	PaidAt         *time.Time     `json:"paid_at"`
	PayoutID       *string        `json:"payout_id" gorm:"type:uuid"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relationships
	Creator      *CreatorProfile `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	Series       *Series         `json:"series,omitempty" gorm:"foreignKey:SeriesID"`
	Episode      *Episode        `json:"episode,omitempty" gorm:"foreignKey:EpisodeID"`
	Subscription *Subscription   `json:"subscription,omitempty" gorm:"foreignKey:SubscriptionID"`
	Payout       *Payout         `json:"payout,omitempty" gorm:"foreignKey:PayoutID"`
}

// Payout represents a payout request to a creator
type Payout struct {
	ID                    string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CreatorID             string         `json:"creator_id" gorm:"type:uuid;not null;index"`
	Amount                float64        `json:"amount" gorm:"type:decimal(10,2);not null"`
	Status                string         `json:"status" gorm:"type:varchar(20);default:'pending';check:status IN ('pending', 'processing', 'completed', 'failed')"`
	RazorpayPayoutID      *string        `json:"razorpay_payout_id" gorm:"type:varchar(255)"`
	RazorpayFundAccountID *string        `json:"razorpay_fund_account_id" gorm:"type:varchar(255)"`
	TransactionReference  *string        `json:"transaction_reference" gorm:"type:varchar(255)"`
	PayoutMethod          string         `json:"payout_method" gorm:"type:varchar(50);default:'bank_transfer';check:payout_method IN ('bank_transfer', 'upi')"`
	FailureReason         *string        `json:"failure_reason" gorm:"type:text"`
	RequestedAt           time.Time      `json:"requested_at" gorm:"default:now()"`
	CompletedAt           *time.Time     `json:"completed_at"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relationships
	Creator  *CreatorProfile   `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	Earnings []CreatorEarnings `json:"earnings,omitempty" gorm:"foreignKey:PayoutID"`
}

// SeriesView represents a single view of a series
type SeriesView struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SeriesID  string    `json:"series_id" gorm:"type:uuid;not null;index"`
	UserID    *string   `json:"user_id" gorm:"type:uuid;index"`
	SessionID *string   `json:"session_id" gorm:"type:varchar(255)"`
	IPAddress *string   `json:"ip_address" gorm:"type:varchar(45)"`
	UserAgent *string   `json:"user_agent" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`

	// Relationships
	Series *Series `json:"series,omitempty" gorm:"foreignKey:SeriesID"`
	User   *User   `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (CreatorEarnings) TableName() string {
	return "creator_earnings"
}

func (Payout) TableName() string {
	return "payouts"
}

func (SeriesView) TableName() string {
	return "series_views"
}

// Constants for platform fee and minimum payout threshold
const (
	PlatformFeePercent     = 0.30  // 30% platform fee
	CreatorSharePercent    = 0.70  // 70% creator share
	MinimumPayoutThreshold = 500.0 // ₹500 minimum payout
)

// EarningsSummary represents the summary of a creator's earnings
type EarningsSummary struct {
	CreatorID          string            `json:"creator_id"`
	TotalEarnings      float64           `json:"total_earnings"`
	PendingEarnings    float64           `json:"pending_earnings"`
	PaidEarnings       float64           `json:"paid_earnings"`
	AvailableForPayout float64           `json:"available_for_payout"`
	MinimumThreshold   float64           `json:"minimum_threshold"`
	CanRequestPayout   bool              `json:"can_request_payout"`
	Breakdown          EarningsBreakdown `json:"breakdown"`
	Currency           string            `json:"currency"`
	LastPayoutDate     *time.Time        `json:"last_payout_date"`
	NextPayoutDate     *time.Time        `json:"next_payout_date"`
}

// EarningsBreakdown represents the breakdown of earnings by type
type EarningsBreakdown struct {
	SubscriptionRevenue SubscriptionRevenueBreakdown `json:"subscription_revenue"`
	OneTimePurchases    OneTimePurchaseBreakdown     `json:"one_time_purchases"`
	AdRevenue           AdRevenueBreakdown           `json:"ad_revenue"`
}

// SubscriptionRevenueBreakdown represents subscription revenue details
type SubscriptionRevenueBreakdown struct {
	Total               float64 `json:"total"`
	ActiveSubscriptions int64   `json:"active_subscriptions"`
	MonthlyRecurring    float64 `json:"monthly_recurring"`
}

// OneTimePurchaseBreakdown represents one-time purchase details
type OneTimePurchaseBreakdown struct {
	Total float64 `json:"total"`
	Count int64   `json:"count"`
}

// AdRevenueBreakdown represents ad revenue details
type AdRevenueBreakdown struct {
	Total float64 `json:"total"`
}

// SeriesEarningsBreakdown represents earnings for a specific series
type SeriesEarningsBreakdown struct {
	SeriesID            string  `json:"series_id"`
	SeriesTitle         string  `json:"series_title"`
	Earnings            float64 `json:"earnings"`
	SubscriptionRevenue float64 `json:"subscription_revenue"`
	OneTimePurchases    float64 `json:"one_time_purchases"`
	ActiveSubscriptions int64   `json:"active_subscriptions"`
	TotalViews          int64   `json:"total_views"`
}

// EpisodeEarningsBreakdown represents earnings for a specific episode
type EpisodeEarningsBreakdown struct {
	EpisodeID        string  `json:"episode_id"`
	EpisodeTitle     string  `json:"episode_title"`
	SeriesID         string  `json:"series_id"`
	Earnings         float64 `json:"earnings"`
	Views            int64   `json:"views"`
	WatchTimeSeconds int64   `json:"watch_time_seconds"`
}

// PayoutRequest represents a payout request from a creator
type PayoutRequest struct {
	Amount       float64 `json:"amount"`
	PayoutMethod string  `json:"payout_method"`
}

// PayoutResponse represents the response after requesting a payout
type PayoutResponse struct {
	PayoutID            string    `json:"payout_id"`
	CreatorID           string    `json:"creator_id"`
	Amount              float64   `json:"amount"`
	Status              string    `json:"status"`
	RequestedAt         time.Time `json:"requested_at"`
	EstimatedCompletion time.Time `json:"estimated_completion"`
	RazorpayPayoutID    *string   `json:"razorpay_payout_id,omitempty"`
}

// PayoutHistoryItem represents a single payout in history
type PayoutHistoryItem struct {
	PayoutID             string     `json:"payout_id"`
	Amount               float64    `json:"amount"`
	Status               string     `json:"status"`
	RequestedAt          time.Time  `json:"requested_at"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	RazorpayPayoutID     *string    `json:"razorpay_payout_id,omitempty"`
	TransactionReference *string    `json:"transaction_reference,omitempty"`
}

// PayoutDetailsRequest represents request to update payout details
type PayoutDetailsRequest struct {
	AccountHolderName string  `json:"account_holder_name"`
	AccountNumber     string  `json:"account_number"`
	IFSCCode          string  `json:"ifsc_code"`
	BankName          string  `json:"bank_name"`
	AccountType       string  `json:"account_type"` // savings, current
	UPIID             *string `json:"upi_id,omitempty"`
}

// PayoutDetailsResponse represents the response with payout details
type PayoutDetailsResponse struct {
	CreatorID     string            `json:"creator_id"`
	PayoutDetails PayoutDetailsInfo `json:"payout_details"`
}

// PayoutDetailsInfo represents detailed payout information
type PayoutDetailsInfo struct {
	AccountHolderName string    `json:"account_holder_name"`
	AccountNumber     string    `json:"account_number"`
	IFSCCode          string    `json:"ifsc_code"`
	BankName          string    `json:"bank_name"`
	AccountType       string    `json:"account_type"`
	UPIID             *string   `json:"upi_id,omitempty"`
	Verified          bool      `json:"verified"`
	UpdatedAt         time.Time `json:"updated_at"`
}
