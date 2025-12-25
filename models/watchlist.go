package models

import (
	"time"
)

// WatchList represents a series added to a user's watch list
type WatchList struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    string    `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:idx_user_series_unique"`
	SeriesID  string    `json:"series_id" gorm:"type:uuid;not null;uniqueIndex:idx_user_series_unique"`
	CreatedAt time.Time `json:"created_at"`

	// Relationships
	User   *User   `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Series *Series `json:"series,omitempty" gorm:"foreignKey:SeriesID"`
}
