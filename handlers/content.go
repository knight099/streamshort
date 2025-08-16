package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"streamshort/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type ContentHandler struct {
	db *gorm.DB
}

func NewContentHandler(db *gorm.DB) *ContentHandler {
	return &ContentHandler{db: db}
}

// Request/Response structs matching OpenAPI schema
type CreateSeriesRequest struct {
	Title        string   `json:"title"`
	Synopsis     string   `json:"synopsis"`
	Language     string   `json:"language"`
	CategoryTags []string `json:"category_tags"`
	PriceType    string   `json:"price_type"`
	PriceAmount  *float64 `json:"price_amount"`
	ThumbnailURL *string  `json:"thumbnail_url"`
}

type UpdateSeriesRequest struct {
	Title        *string   `json:"title"`
	Synopsis     *string   `json:"synopsis"`
	Language     *string   `json:"language"`
	CategoryTags *[]string `json:"category_tags"`
	PriceType    *string   `json:"price_type"`
	PriceAmount  *float64  `json:"price_amount"`
	ThumbnailURL *string   `json:"thumbnail_url"`
	Status       *string   `json:"status"`
}

type CreateEpisodeRequest struct {
	Title           string `json:"title"`
	EpisodeNumber   int    `json:"episode_number"`
	DurationSeconds int    `json:"duration_seconds"`
}

type SeriesListResponse struct {
	Total int64           `json:"total"`
	Items []models.Series `json:"items"`
}

type UploadUrlRequest struct {
	Filename    string                 `json:"filename"`
	ContentType string                 `json:"content_type"`
	SizeBytes   int64                  `json:"size_bytes"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type UploadUrlResponse struct {
	UploadID      string            `json:"upload_id"`
	PresignedURL  string            `json:"presigned_url"`
	ExpiresIn     int               `json:"expires_in"`
	UploadHeaders map[string]string `json:"upload_headers"`
}

type UploadNotifyRequest struct {
	S3Path    string `json:"s3_path"`
	SizeBytes int64  `json:"size_bytes"`
}

type UploadNotifyResponse struct {
	Status string `json:"status"`
}

type ManifestResponse struct {
	ManifestURL string    `json:"manifest_url"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// CreateSeries creates a new series
func (h *ContentHandler) CreateSeries(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	var req CreateSeriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Title == "" || req.Synopsis == "" || req.Language == "" {
		http.Error(w, "Title, synopsis, and language are required", http.StatusBadRequest)
		return
	}

	// Check if user is a creator
	var creatorProfile models.CreatorProfile
	if err := h.db.Where("user_id = ?", userID).First(&creatorProfile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "User must be onboarded as a creator first", http.StatusForbidden)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Create series
	series := models.Series{
		CreatorID:    creatorProfile.ID,
		Title:        req.Title,
		Synopsis:     req.Synopsis,
		Language:     req.Language,
		CategoryTags: req.CategoryTags,
		PriceType:    req.PriceType,
		PriceAmount:  req.PriceAmount,
		ThumbnailURL: req.ThumbnailURL,
		Status:       "draft",
	}

	if err := h.db.Create(&series).Error; err != nil {
		http.Error(w, "Failed to create series", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(series)
}

// ListSeries lists series with optional filters
func (h *ContentHandler) ListSeries(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	language := r.URL.Query().Get("language")
	category := r.URL.Query().Get("category")
	pageStr := r.URL.Query().Get("page")
	perPageStr := r.URL.Query().Get("per_page")

	// Set defaults
	page := 1
	perPage := 20

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	// Build query
	query := h.db.Model(&models.Series{}).Where("status = ?", "published")

	if language != "" {
		query = query.Where("language = ?", language)
	}

	if category != "" {
		query = query.Where("? = ANY(category_tags)", category)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get paginated results
	var series []models.Series
	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Find(&series).Error; err != nil {
		http.Error(w, "Failed to fetch series", http.StatusInternalServerError)
		return
	}

	response := SeriesListResponse{
		Total: total,
		Items: series,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSeries gets a specific series by ID
func (h *ContentHandler) GetSeries(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	seriesID := vars["id"]

	var series models.Series
	if err := h.db.Preload("Creator").Preload("Episodes").Where("id = ?", seriesID).First(&series).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Series not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(series)
}

// UpdateSeries updates a series
func (h *ContentHandler) UpdateSeries(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	seriesID := vars["id"]

	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	var req UpdateSeriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if series exists and user owns it
	var series models.Series
	if err := h.db.Joins("JOIN creator_profiles ON series.creator_id = creator_profiles.id").
		Where("series.id = ? AND creator_profiles.user_id = ?", seriesID, userID).
		First(&series).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Series not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Update fields
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Synopsis != nil {
		updates["synopsis"] = *req.Synopsis
	}
	if req.Language != nil {
		updates["language"] = *req.Language
	}
	if req.CategoryTags != nil {
		updates["category_tags"] = *req.CategoryTags
	}
	if req.PriceType != nil {
		updates["price_type"] = *req.PriceType
	}
	if req.PriceAmount != nil {
		updates["price_amount"] = *req.PriceAmount
	}
	if req.ThumbnailURL != nil {
		updates["thumbnail_url"] = *req.ThumbnailURL
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	updates["updated_at"] = time.Now()

	if err := h.db.Model(&series).Updates(updates).Error; err != nil {
		http.Error(w, "Failed to update series", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Series updated successfully"})
}

// CreateEpisode creates episode metadata for a series
func (h *ContentHandler) CreateEpisode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	seriesID := vars["id"]

	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	var req CreateEpisodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Title == "" || req.EpisodeNumber <= 0 || req.DurationSeconds <= 0 {
		http.Error(w, "Title, episode number, and duration are required", http.StatusBadRequest)
		return
	}

	// Check if series exists and user owns it
	var series models.Series
	if err := h.db.Joins("JOIN creator_profiles ON series.creator_id = creator_profiles.id").
		Where("series.id = ? AND creator_profiles.user_id = ?", seriesID, userID).
		First(&series).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Series not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check if episode number already exists
	var existingEpisode models.Episode
	if err := h.db.Where("series_id = ? AND episode_number = ?", seriesID, req.EpisodeNumber).First(&existingEpisode).Error; err == nil {
		http.Error(w, "Episode number already exists for this series", http.StatusConflict)
		return
	}

	// Create episode
	episode := models.Episode{
		SeriesID:        seriesID,
		Title:           req.Title,
		EpisodeNumber:   req.EpisodeNumber,
		DurationSeconds: req.DurationSeconds,
		Status:          "pending_upload",
	}

	if err := h.db.Create(&episode).Error; err != nil {
		http.Error(w, "Failed to create episode", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(episode)
}

// RequestUploadURL generates a pre-signed upload URL
func (h *ContentHandler) RequestUploadURL(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	var req UploadUrlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Filename == "" || req.ContentType == "" || req.SizeBytes <= 0 {
		http.Error(w, "Filename, content type, and size are required", http.StatusBadRequest)
		return
	}

	// Check if user is a creator
	var creatorProfile models.CreatorProfile
	if err := h.db.Where("user_id = ?", userID).First(&creatorProfile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "User must be onboarded as a creator first", http.StatusForbidden)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Generate upload ID
	uploadID := fmt.Sprintf("upl_%s", uuid.New().String()[:8])

	// Create upload request record
	uploadReq := models.UploadRequest{
		UserID:      userID,
		Filename:    req.Filename,
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
		Metadata:    req.Metadata,
		Status:      "pending",
	}

	if err := h.db.Create(&uploadReq).Error; err != nil {
		http.Error(w, "Failed to create upload request", http.StatusInternalServerError)
		return
	}

	// TODO: In production, integrate with AWS S3 to generate actual pre-signed URL
	// For now, return a mock response
	response := UploadUrlResponse{
		UploadID:     uploadID,
		PresignedURL: fmt.Sprintf("https://s3.amazonaws.com/bucket/%s?AWSAccessKeyId=mock&Signature=mock", uploadID),
		ExpiresIn:    3600,
		UploadHeaders: map[string]string{
			"Content-Type": req.ContentType,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// NotifyUploadComplete handles upload completion notification
func (h *ContentHandler) NotifyUploadComplete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uploadID := vars["upload_id"]

	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	var req UploadNotifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.S3Path == "" || req.SizeBytes <= 0 {
		http.Error(w, "S3 path and size are required", http.StatusBadRequest)
		return
	}

	// Update upload request status
	if err := h.db.Model(&models.UploadRequest{}).
		Where("id = ? AND user_id = ?", uploadID, userID).
		Updates(map[string]interface{}{
			"status":     "completed",
			"updated_at": time.Now(),
		}).Error; err != nil {
		http.Error(w, "Failed to update upload status", http.StatusInternalServerError)
		return
	}

	// TODO: In production, trigger transcoding job here
	response := UploadNotifyResponse{
		Status: "queued_for_transcoding",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// GetEpisodeManifest gets signed HLS manifest URL for playback
func (h *ContentHandler) GetEpisodeManifest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	episodeID := vars["id"]

	// Get user ID from context (for future subscription checks)
	_, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Get episode with series info
	var episode models.Episode
	if err := h.db.Preload("Series").Where("id = ?", episodeID).First(&episode).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Episode not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check if episode is ready for playback
	if episode.Status != "published" {
		http.Error(w, "Episode not ready for playback", http.StatusBadRequest)
		return
	}

	// TODO: In production, check user subscription status
	// For now, allow access to all authenticated users

	// TODO: In production, generate actual signed URL with expiration
	// For now, return a mock response
	response := ManifestResponse{
		ManifestURL: fmt.Sprintf("https://cdn.streamshort.com/hls/%s/index.m3u8?Expires=%d&Signature=mock", episodeID, time.Now().Add(1*time.Hour).Unix()),
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
