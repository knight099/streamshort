package models

import (
	"time"

	"gorm.io/gorm"
)

// EpisodeLike represents a user's like on an episode
type EpisodeLike struct {
	ID        string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	EpisodeID string         `json:"episode_id" gorm:"type:uuid;not null;index:idx_episode_like_episode_user,unique"`
	UserID    string         `json:"user_id" gorm:"type:uuid;not null;index:idx_episode_like_episode_user,unique"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// SeriesRating represents a user's rating for a series (1-5 stars)
type SeriesRating struct {
	ID        string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SeriesID  string         `json:"series_id" gorm:"type:uuid;not null;index:idx_series_rating_series_user,unique"`
	UserID    string         `json:"user_id" gorm:"type:uuid;not null;index:idx_series_rating_series_user,unique"`
	Score     int            `json:"score" gorm:"not null;check:score >= 1 AND score <= 5"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

func (SeriesRating) TableName() string {
	return "series_ratings"
}
