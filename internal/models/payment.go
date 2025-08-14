package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Subscription represents a user's subscription to a series
type Subscription struct {
	ID                     uuid.UUID          `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID                 uuid.UUID          `json:"user_id" gorm:"type:uuid;not null"`
	SeriesID               uuid.UUID          `json:"series_id" gorm:"type:uuid;not null"`
	RazorpaySubscriptionID *string            `json:"razorpay_subscription_id,omitempty"`
	RazorpayCustomerID     *string            `json:"razorpay_customer_id,omitempty"`
	Status                 SubscriptionStatus `json:"status" gorm:"type:subscription_status;default:'active'"`
	StartsAt               time.Time          `json:"starts_at" gorm:"default:now()"`
	ExpiresAt              *time.Time         `json:"expires_at,omitempty"`
	Amount                 *float64           `json:"amount,omitempty" gorm:"type:decimal(10,2)"`
	Currency               string             `json:"currency" gorm:"default:'INR'"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
	DeletedAt              gorm.DeletedAt     `json:"-" gorm:"index"`

	// Relations
	User   User   `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Series Series `json:"series,omitempty" gorm:"foreignKey:SeriesID"`
}

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusPending   SubscriptionStatus = "pending"
)

// PaymentTransaction represents a payment transaction
type PaymentTransaction struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID            *uuid.UUID     `json:"user_id,omitempty" gorm:"type:uuid"`
	SubscriptionID    *uuid.UUID     `json:"subscription_id,omitempty" gorm:"type:uuid"`
	RazorpayPaymentID *string        `json:"razorpay_payment_id,omitempty"`
	Amount            float64        `json:"amount" gorm:"type:decimal(10,2);not null"`
	Currency          string         `json:"currency" gorm:"default:'INR'"`
	Status            string         `json:"status" gorm:"not null"`
	PaymentMethod     *string        `json:"payment_method,omitempty"`
	GatewayResponse   JSON           `json:"gateway_response" gorm:"type:jsonb;default:'{}'"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	User         *User         `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Subscription *Subscription `json:"subscription,omitempty" gorm:"foreignKey:SubscriptionID"`
}

// CreatorPayout represents a payout to a creator
type CreatorPayout struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CreatorID     uuid.UUID      `json:"creator_id" gorm:"type:uuid;not null"`
	Amount        float64        `json:"amount" gorm:"type:decimal(10,2);not null"`
	Currency      string         `json:"currency" gorm:"default:'INR'"`
	Status        string         `json:"status" gorm:"default:'pending'"`
	PayoutMethod  *string        `json:"payout_method,omitempty"`
	PayoutDetails JSON           `json:"payout_details" gorm:"type:jsonb;default:'{}'"`
	ProcessedAt   *time.Time     `json:"processed_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Creator CreatorProfile `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
}

// PaymentWebhook represents incoming webhook data from Razorpay
type PaymentWebhook struct {
	ID         uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Event      string         `json:"event" gorm:"not null"`
	Data       JSON           `json:"data" gorm:"type:jsonb;default:'{}'"`
	Signature  string         `json:"signature" gorm:"not null"`
	ReceivedAt time.Time      `json:"received_at"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName specifies the table name for Subscription
func (Subscription) TableName() string {
	return "subscriptions"
}

// TableName specifies the table name for PaymentTransaction
func (PaymentTransaction) TableName() string {
	return "payment_transactions"
}

// TableName specifies the table name for CreatorPayout
func (CreatorPayout) TableName() string {
	return "creator_payouts"
}

// TableName specifies the table name for PaymentWebhook
func (PaymentWebhook) TableName() string {
	return "payment_webhooks"
}
