package models

import (
	"time"

	"gorm.io/gorm"
)

type CreatorProfile struct {
	ID              string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID          string         `json:"user_id" gorm:"type:uuid;not null;uniqueIndex"`
	DisplayName     string         `json:"display_name" gorm:"not null"`
	Bio             string         `json:"bio"`
	KYCDocumentPath string         `json:"kyc_document_s3_path" gorm:"column:kyc_document_s3_path"`
	KYCStatus       string         `json:"kyc_status" gorm:"default:'pending';check:kyc_status IN ('pending', 'verified', 'rejected')"`
	PayoutDetails   *PayoutDetails `json:"payout_details" gorm:"foreignKey:CreatorID"`
	FollowerCount   int64          `json:"follower_count" gorm:"default:0"`
	Rating          *float64       `json:"rating" gorm:"type:decimal(3,2)"`
	// Derived fields (not stored)
	SeriesCount int64          `json:"series_count" gorm:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relationships
	User *User `json:"user" gorm:"foreignKey:UserID"`
}

// CreatorFollow represents a user following a creator
type CreatorFollow struct {
	ID         string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	FollowerID string         `json:"follower_id" gorm:"type:uuid;not null;uniqueIndex:uix_follower_creator;index:idx_creator_follows_follower"`
	CreatorID  string         `json:"creator_id" gorm:"type:uuid;not null;uniqueIndex:uix_follower_creator;index:idx_creator_follows_creator"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relationships
	Follower *User           `json:"follower" gorm:"foreignKey:FollowerID;references:ID"`
	Creator  *CreatorProfile `json:"creator" gorm:"foreignKey:CreatorID;references:ID"`
}

func (CreatorFollow) TableName() string { return "creator_follows" }

type PayoutDetails struct {
	ID                    string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CreatorID             string         `json:"creator_id" gorm:"type:uuid;not null;uniqueIndex"`
	BankName              string         `json:"bank_name"`
	AccountNumber         string         `json:"account_number"`
	IFSCCode              string         `json:"ifsc_code"`
	AccountHolder         string         `json:"account_holder"`
	AccountType           string         `json:"account_type" gorm:"type:varchar(20);default:'savings'"`
	UPIID                 *string        `json:"upi_id" gorm:"type:varchar(255)"`
	Verified              bool           `json:"verified" gorm:"default:false"`
	RazorpayContactID     *string        `json:"razorpay_contact_id" gorm:"type:varchar(255)"`
	RazorpayFundAccountID *string        `json:"razorpay_fund_account_id" gorm:"type:varchar(255)"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

type CreatorAnalytics struct {
	ID               string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CreatorID        string         `json:"creator_id" gorm:"type:uuid;not null;index"`
	Date             time.Time      `json:"date" gorm:"type:date;not null"`
	Views            int64          `json:"views" gorm:"default:0"`
	WatchTimeSeconds int64          `json:"watch_time_seconds" gorm:"default:0"`
	Earnings         float64        `json:"earnings" gorm:"type:decimal(10,2);default:0"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relationships
	Creator CreatorProfile `json:"creator" gorm:"foreignKey:CreatorID"`
}

// TableName specifies the table name for CreatorProfile
func (CreatorProfile) TableName() string {
	return "creator_profiles"
}

// TableName specifies the table name for PayoutDetails
func (PayoutDetails) TableName() string {
	return "payout_details"
}

// TableName specifies the table name for CreatorAnalytics
func (CreatorAnalytics) TableName() string {
	return "creator_analytics"
}
