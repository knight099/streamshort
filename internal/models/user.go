package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email         string         `json:"email" gorm:"uniqueIndex;not null"`
	Phone         *string        `json:"phone,omitempty" gorm:"uniqueIndex"`
	PasswordHash  string         `json:"-" gorm:"not null"`
	Role          UserRole       `json:"role" gorm:"type:user_role;default:'user'"`
	Profile       JSON           `json:"profile" gorm:"type:jsonb;default:'{}'"`
	IsActive      bool           `json:"is_active" gorm:"default:true"`
	EmailVerified bool           `json:"email_verified" gorm:"default:false"`
	PhoneVerified bool           `json:"phone_verified" gorm:"default:false"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	CreatorProfile *CreatorProfile  `json:"creator_profile,omitempty" gorm:"foreignKey:UserID"`
	Subscriptions  []Subscription   `json:"subscriptions,omitempty" gorm:"foreignKey:UserID"`
	Preferences    *UserPreferences `json:"preferences,omitempty" gorm:"foreignKey:UserID"`
}

// UserRole represents the role of a user
type UserRole string

const (
	UserRoleUser    UserRole = "user"
	UserRoleCreator UserRole = "creator"
	UserRoleAdmin   UserRole = "admin"
)

// CreatorProfile represents a creator's profile
type CreatorProfile struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID        uuid.UUID      `json:"user_id" gorm:"type:uuid;not null"`
	DisplayName   string         `json:"display_name" gorm:"not null"`
	Bio           *string        `json:"bio,omitempty"`
	AvatarURL     *string        `json:"avatar_url,omitempty"`
	KYCStatus     KYCStatus      `json:"kyc_status" gorm:"type:kyc_status;default:'pending'"`
	PayoutDetails JSON           `json:"payout_details" gorm:"type:jsonb;default:'{}'"`
	SocialLinks   JSON           `json:"social_links" gorm:"type:jsonb;default:'{}'"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	User   User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Series []Series `json:"series,omitempty" gorm:"foreignKey:CreatorID"`
}

// KYCStatus represents the KYC verification status
type KYCStatus string

const (
	KYCStatusPending  KYCStatus = "pending"
	KYCStatusVerified KYCStatus = "verified"
	KYCStatusRejected KYCStatus = "rejected"
)

// UserPreferences represents user preferences and settings
type UserPreferences struct {
	ID                   uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID               uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;uniqueIndex"`
	WatchHistory         JSON           `json:"watch_history" gorm:"type:jsonb;default:'[]'"`
	FavoriteSeries       JSON           `json:"favorite_series" gorm:"type:jsonb;default:'[]'"`
	NotificationSettings JSON           `json:"notification_settings" gorm:"type:jsonb;default:'{}'"`
	LanguagePreference   string         `json:"language_preference" gorm:"default:'en'"`
	QualityPreference    string         `json:"quality_preference" gorm:"default:'720'"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// JSON is a custom type for JSONB fields
type JSON map[string]interface{}

// Value implements the driver.Valuer interface for JSON
func (j JSON) Value() (interface{}, error) {
	if j == nil {
		return nil, nil
	}
	return j, nil
}

// Scan implements the sql.Scanner interface for JSON
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		return fmt.Errorf("cannot scan %T into JSON", value)
	}
}

// TableName specifies the table name for User
func (User) TableName() string {
	return "users"
}

// TableName specifies the table name for CreatorProfile
func (CreatorProfile) TableName() string {
	return "creator_profiles"
}

// TableName specifies the table name for UserPreferences
func (UserPreferences) TableName() string {
	return "user_preferences"
}
