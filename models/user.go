package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Phone     string         `json:"phone" gorm:"uniqueIndex;not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

type OTPTransaction struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	TxnID     string         `json:"txn_id" gorm:"uniqueIndex;not null"`
	Phone     string         `json:"phone" gorm:"not null"`
	OTP       string         `json:"otp" gorm:"not null"`
	ExpiresAt time.Time      `json:"expires_at" gorm:"not null"`
	Used      bool           `json:"used" gorm:"default:false"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

type RefreshToken struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Token        string         `json:"token" gorm:"uniqueIndex;not null"`
	UserID       uint           `json:"user_id" gorm:"not null"`
	ExpiresAt    time.Time      `json:"expires_at" gorm:"not null"`
	Revoked      bool           `json:"revoked" gorm:"default:false"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}
