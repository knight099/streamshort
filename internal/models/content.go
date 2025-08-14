package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Series represents a series/show in the platform
type Series struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CreatorID    uuid.UUID      `json:"creator_id" gorm:"type:uuid;not null"`
	Title        string         `json:"title" gorm:"not null"`
	Description  *string        `json:"description,omitempty"`
	ThumbnailURL *string        `json:"thumbnail_url,omitempty"`
	BannerURL    *string        `json:"banner_url,omitempty"`
	Category     *string        `json:"category,omitempty"`
	Language     *string        `json:"language,omitempty"`
	PriceAmount  *float64       `json:"price_amount,omitempty" gorm:"type:decimal(10,2)"`
	Currency     string         `json:"currency" gorm:"default:'INR'"`
	Status       ContentStatus  `json:"status" gorm:"type:content_status;default:'draft'"`
	Tags         []string       `json:"tags,omitempty" gorm:"type:text[]"`
	Metadata     JSON           `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Creator  CreatorProfile `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	Episodes []Episode      `json:"episodes,omitempty" gorm:"foreignKey:SeriesID"`
}

// Episode represents an episode within a series
type Episode struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SeriesID        uuid.UUID      `json:"series_id" gorm:"type:uuid;not null"`
	Title           string         `json:"title" gorm:"not null"`
	EpisodeNumber   *int           `json:"episode_number,omitempty"`
	Description     *string        `json:"description,omitempty"`
	DurationSeconds *int           `json:"duration_seconds,omitempty"`
	S3Key           *string        `json:"s3_key,omitempty"`
	HLSManifestURL  *string        `json:"hls_manifest_url,omitempty"`
	ThumbnailURL    *string        `json:"thumbnail_url,omitempty"`
	Status          ContentStatus  `json:"status" gorm:"type:content_status;default:'processing'"`
	PublishedAt     *time.Time     `json:"published_at,omitempty"`
	Metadata        JSON           `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Series Series `json:"series,omitempty" gorm:"foreignKey:SeriesID"`
}

// ContentStatus represents the status of content
type ContentStatus string

const (
	ContentStatusDraft      ContentStatus = "draft"
	ContentStatusProcessing ContentStatus = "processing"
	ContentStatusPublished  ContentStatus = "published"
	ContentStatusArchived   ContentStatus = "archived"
)

// UploadRequest represents a request for content upload
type UploadRequest struct {
	ID          uuid.UUID `json:"id"`
	SeriesID    uuid.UUID `json:"series_id"`
	EpisodeID   uuid.UUID `json:"episode_id"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ContentType string    `json:"content_type"`
	UploadURL   string    `json:"upload_url"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// TranscodingJob represents a video transcoding job
type TranscodingJob struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	EpisodeID   uuid.UUID      `json:"episode_id" gorm:"type:uuid;not null"`
	JobID       string         `json:"job_id" gorm:"not null"`
	Status      string         `json:"status" gorm:"default:'pending'"`
	InputPath   string         `json:"input_path" gorm:"not null"`
	OutputPaths JSON           `json:"output_paths" gorm:"type:jsonb;default:'{}'"`
	Progress    int            `json:"progress" gorm:"default:0"`
	Error       *string        `json:"error,omitempty"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Episode Episode `json:"episode,omitempty" gorm:"foreignKey:EpisodeID"`
}

// TableName specifies the table name for Series
func (Series) TableName() string {
	return "series"
}

// TableName specifies the table name for Episode
func (Episode) TableName() string {
	return "episodes"
}

// TableName specifies the table name for TranscodingJob
func (TranscodingJob) TableName() string {
	return "transcoding_jobs"
}
