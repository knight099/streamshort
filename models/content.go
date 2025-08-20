package models

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Series represents a video series
type Series struct {
	ID           string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CreatorID    string         `json:"creator_id" gorm:"type:uuid;not null"`
	Title        string         `json:"title" gorm:"not null"`
	Synopsis     string         `json:"synopsis" gorm:"not null"`
	Language     string         `json:"language" gorm:"not null"`
	CategoryTags pq.StringArray `json:"category_tags" gorm:"type:text[]"`
	PriceType    string         `json:"price_type" gorm:"type:varchar(20);check:price_type IN ('free', 'subscription', 'one_time')"`
	PriceAmount  *float64       `json:"price_amount" gorm:"type:decimal(10,2)"`
	ThumbnailURL *string        `json:"thumbnail_url"`
	Status       string         `json:"status" gorm:"type:varchar(20);default:'draft';check:status IN ('draft', 'published')"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relationships
	Creator  CreatorProfile `json:"creator" gorm:"foreignKey:CreatorID"`
	Episodes []Episode      `json:"episodes" gorm:"foreignKey:SeriesID"`
}

// Episode represents a single episode in a series
type Episode struct {
	ID              string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SeriesID        string         `json:"series_id" gorm:"type:uuid;not null"`
	Title           string         `json:"title" gorm:"not null"`
	EpisodeNumber   int            `json:"episode_number" gorm:"not null"`
	DurationSeconds int            `json:"duration_seconds" gorm:"not null"`
	S3MasterPath    *string        `json:"s3_master_path"`
	HLSManifestURL  *string        `json:"hls_manifest_url"`
	ThumbURL        *string        `json:"thumb_url"`
	CaptionsURL     *string        `json:"captions_url"`
	Status          string         `json:"status" gorm:"type:varchar(30);default:'pending_upload';check:status IN ('pending_upload', 'queued_transcode', 'ready', 'published')"`
	PublishedAt     *time.Time     `json:"published_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relationships
	Series Series `json:"series" gorm:"foreignKey:SeriesID"`
}

// UploadRequest represents a request for upload URL
type UploadRequest struct {
	ID          string                 `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID      string                 `json:"user_id" gorm:"type:uuid;not null"`
	Filename    string                 `json:"filename" gorm:"not null"`
	ContentType string                 `json:"content_type" gorm:"not null"`
	SizeBytes   int64                  `json:"size_bytes" gorm:"not null"`
	Metadata    map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	Status      string                 `json:"status" gorm:"type:varchar(30);default:'pending';check:status IN ('pending', 'uploading', 'completed', 'failed')"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	DeletedAt   gorm.DeletedAt         `json:"deleted_at,omitempty" gorm:"index"`

	// Relationships
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for Series
func (Series) TableName() string {
	return "series"
}

// TableName specifies the table name for Episode
func (Episode) TableName() string {
	return "episodes"
}

// TableName specifies the table name for UploadRequest
func (UploadRequest) TableName() string {
	return "upload_requests"
}
