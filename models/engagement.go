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

// EpisodeRating represents a user's rating for an episode
type EpisodeRating struct {
	ID        string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	EpisodeID string         `json:"episode_id" gorm:"type:uuid;not null;index:idx_episode_rating_episode_user,unique"`
	UserID    string         `json:"user_id" gorm:"type:uuid;not null;index:idx_episode_rating_episode_user,unique"`
	Score     int            `json:"score" gorm:"not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// EpisodeComment represents a comment made by a user on an episode
type EpisodeComment struct {
	ID        string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	EpisodeID string         `json:"episode_id" gorm:"type:uuid;not null;index"`
	UserID    string         `json:"user_id" gorm:"type:uuid;not null;index"`
	Text      string         `json:"text" gorm:"type:text;not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}
