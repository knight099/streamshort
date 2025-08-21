package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"streamshort/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
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

type SeriesListItem struct {
	ID           string         `json:"id"`
	CreatorID    string         `json:"creator_id"`
	CreatorName  *string        `json:"creator_name"`
	Title        string         `json:"title"`
	Synopsis     string         `json:"synopsis"`
	Language     string         `json:"language"`
	CategoryTags pq.StringArray `json:"category_tags"`
	PriceType    string         `json:"price_type"`
	PriceAmount  *float64       `json:"price_amount"`
	ThumbnailURL *string        `json:"thumbnail_url"`
	Status       string         `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Episodes     []EpisodeBrief `json:"episodes"`
}

type EpisodeBrief struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	EpisodeNumber   int        `json:"episode_number"`
	DurationSeconds int        `json:"duration_seconds"`
	ThumbURL        *string    `json:"thumb_url"`
	PublishedAt     *time.Time `json:"published_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type SeriesListResponse struct {
	Total int64            `json:"total"`
	Items []SeriesListItem `json:"items"`
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
		CategoryTags: pq.StringArray(req.CategoryTags),
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
	query := h.db.Model(&models.Series{}).Where("status = ?", "published").
		Preload("Creator").
		Preload("Episodes", "status = ?", "published")

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
	var seriesRows []models.Series
	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Find(&seriesRows).Error; err != nil {
		http.Error(w, "Failed to fetch series", http.StatusInternalServerError)
		return
	}

	items := make([]SeriesListItem, 0, len(seriesRows))
	for _, s := range seriesRows {
		var creatorName *string
		if s.Creator != nil {
			creatorName = &s.Creator.DisplayName
		}

		eps := make([]EpisodeBrief, 0, len(s.Episodes))
		for _, ep := range s.Episodes {
			eps = append(eps, EpisodeBrief{
				ID:              ep.ID,
				Title:           ep.Title,
				EpisodeNumber:   ep.EpisodeNumber,
				DurationSeconds: ep.DurationSeconds,
				ThumbURL:        ep.ThumbURL,
				PublishedAt:     ep.PublishedAt,
				CreatedAt:       ep.CreatedAt,
			})
		}

		items = append(items, SeriesListItem{
			ID:           s.ID,
			CreatorID:    s.CreatorID,
			CreatorName:  creatorName,
			Title:        s.Title,
			Synopsis:     s.Synopsis,
			Language:     s.Language,
			CategoryTags: s.CategoryTags,
			PriceType:    s.PriceType,
			PriceAmount:  s.PriceAmount,
			ThumbnailURL: s.ThumbnailURL,
			Status:       s.Status,
			CreatedAt:    s.CreatedAt,
			UpdatedAt:    s.UpdatedAt,
			Episodes:     eps,
		})
	}

	response := SeriesListResponse{
		Total: total,
		Items: items,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSeries gets a specific series by ID
func (h *ContentHandler) GetSeries(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	seriesID := vars["id"]

	var series models.Series
	if err := h.db.Preload("Creator").Preload("Episodes", "status = ?", "published").Where("id = ?", seriesID).First(&series).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Series not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	type SeriesDetailResponse struct {
		ID           string         `json:"id"`
		CreatorID    string         `json:"creator_id"`
		CreatorName  *string        `json:"creator_name"`
		Title        string         `json:"title"`
		Synopsis     string         `json:"synopsis"`
		Language     string         `json:"language"`
		CategoryTags pq.StringArray `json:"category_tags"`
		PriceType    string         `json:"price_type"`
		PriceAmount  *float64       `json:"price_amount"`
		ThumbnailURL *string        `json:"thumbnail_url"`
		Status       string         `json:"status"`
		CreatedAt    time.Time      `json:"created_at"`
		UpdatedAt    time.Time      `json:"updated_at"`
		Episodes     []EpisodeBrief `json:"episodes"`
	}

	var creatorName *string
	if series.Creator != nil {
		creatorName = &series.Creator.DisplayName
	}

	eps := make([]EpisodeBrief, 0, len(series.Episodes))
	for _, ep := range series.Episodes {
		eps = append(eps, EpisodeBrief{
			ID:              ep.ID,
			Title:           ep.Title,
			EpisodeNumber:   ep.EpisodeNumber,
			DurationSeconds: ep.DurationSeconds,
			ThumbURL:        ep.ThumbURL,
			PublishedAt:     ep.PublishedAt,
			CreatedAt:       ep.CreatedAt,
		})
	}

	resp := SeriesDetailResponse{
		ID:           series.ID,
		CreatorID:    series.CreatorID,
		CreatorName:  creatorName,
		Title:        series.Title,
		Synopsis:     series.Synopsis,
		Language:     series.Language,
		CategoryTags: series.CategoryTags,
		PriceType:    series.PriceType,
		PriceAmount:  series.PriceAmount,
		ThumbnailURL: series.ThumbnailURL,
		Status:       series.Status,
		CreatedAt:    series.CreatedAt,
		UpdatedAt:    series.UpdatedAt,
		Episodes:     eps,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
		updates["category_tags"] = pq.StringArray(*req.CategoryTags)
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

// CreatorContentResponse represents the response for creator's content
type CreatorContentResponse struct {
	Series []CreatorSeriesResponse `json:"series"`
	Total  int64                   `json:"total"`
}

// CreatorSeriesResponse represents a series with its episodes for creator view
type CreatorSeriesResponse struct {
	ID           string                   `json:"id"`
	Title        string                   `json:"title"`
	Synopsis     string                   `json:"synopsis"`
	Language     string                   `json:"language"`
	CategoryTags pq.StringArray           `json:"category_tags"`
	PriceType    string                   `json:"price_type"`
	PriceAmount  *float64                 `json:"price_amount"`
	ThumbnailURL *string                  `json:"thumbnail_url"`
	Status       string                   `json:"status"`
	CreatedAt    time.Time                `json:"created_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
	Episodes     []CreatorEpisodeResponse `json:"episodes"`
	EpisodeCount int64                    `json:"episode_count"`
}

// CreatorEpisodeResponse represents an episode for creator view
type CreatorEpisodeResponse struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	EpisodeNumber   int        `json:"episode_number"`
	DurationSeconds int        `json:"duration_seconds"`
	Status          string     `json:"status"`
	PublishedAt     *time.Time `json:"published_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// GetCreatorContent fetches all series and episodes created by the authenticated creator
func (h *ContentHandler) GetCreatorContent(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
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

	// Get all series created by this creator
	var series []models.Series
	if err := h.db.Where("creator_id = ?", creatorProfile.ID).Find(&series).Error; err != nil {
		http.Error(w, "Failed to fetch series", http.StatusInternalServerError)
		return
	}

	// Build response with episodes for each series
	var response CreatorContentResponse
	response.Series = make([]CreatorSeriesResponse, 0, len(series))

	for _, s := range series {
		// Get episodes for this series
		var episodes []models.Episode
		if err := h.db.Where("series_id = ?", s.ID).Order("episode_number").Find(&episodes).Error; err != nil {
			http.Error(w, "Failed to fetch episodes for series", http.StatusInternalServerError)
			return
		}

		// Convert episodes to response format
		episodeResponses := make([]CreatorEpisodeResponse, 0, len(episodes))
		for _, ep := range episodes {
			episodeResponses = append(episodeResponses, CreatorEpisodeResponse{
				ID:              ep.ID,
				Title:           ep.Title,
				EpisodeNumber:   ep.EpisodeNumber,
				DurationSeconds: ep.DurationSeconds,
				Status:          ep.Status,
				PublishedAt:     ep.PublishedAt,
				CreatedAt:       ep.CreatedAt,
				UpdatedAt:       ep.UpdatedAt,
			})
		}

		// Convert series to response format
		seriesResponse := CreatorSeriesResponse{
			ID:           s.ID,
			Title:        s.Title,
			Synopsis:     s.Synopsis,
			Language:     s.Language,
			CategoryTags: s.CategoryTags,
			PriceType:    s.PriceType,
			PriceAmount:  s.PriceAmount,
			ThumbnailURL: s.ThumbnailURL,
			Status:       s.Status,
			CreatedAt:    s.CreatedAt,
			UpdatedAt:    s.UpdatedAt,
			Episodes:     episodeResponses,
			EpisodeCount: int64(len(episodeResponses)),
		}

		response.Series = append(response.Series, seriesResponse)
	}

	response.Total = int64(len(response.Series))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type UpdateEpisodeStatusRequest struct {
	Status string `json:"status"`
}

// UpdateEpisodeStatus allows the creator to update the status of an episode
func (h *ContentHandler) UpdateEpisodeStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	episodeID := vars["id"]

	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Verify ownership: episode belongs to a series owned by this creator
	var episode models.Episode
	if err := h.db.Joins("JOIN series ON episodes.series_id = series.id").
		Joins("JOIN creator_profiles ON series.creator_id = creator_profiles.id").
		Where("episodes.id = ? AND creator_profiles.user_id = ?", episodeID, userID).
		First(&episode).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Episode not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var req UpdateEpisodeStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}

	// Normalize and validate status
	status := strings.ToLower(req.Status)
	if status == "publish" {
		status = "published"
	}
	allowed := map[string]bool{
		"pending_upload":   true,
		"queued_transcode": true,
		"ready":            true,
		"published":        true,
	}
	if !allowed[status] {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if status == "published" {
		now := time.Now()
		updates["published_at"] = &now
	}

	if err := h.db.Model(&episode).Updates(updates).Error; err != nil {
		http.Error(w, "Failed to update episode status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Episode status updated successfully",
		"id":      episode.ID,
		"status":  status,
	})
}

type UpdateSeriesStatusRequest struct {
	Status string `json:"status"`
}

// UpdateSeriesStatus allows the creator to update the status of a series
func (h *ContentHandler) UpdateSeriesStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	seriesID := vars["id"]

	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Verify ownership: series belongs to this creator
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

	var req UpdateSeriesStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}

	// Normalize and validate status
	status := strings.ToLower(req.Status)
	if status == "publish" {
		status = "published"
	}
	allowed := map[string]bool{
		"draft":     true,
		"published": true,
	}
	if !allowed[status] {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if err := h.db.Model(&series).Updates(updates).Error; err != nil {
		http.Error(w, "Failed to update series status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Series status updated successfully",
		"id":      series.ID,
		"status":  status,
	})
}

type UpdateEpisodeRequest struct {
	Title           *string `json:"title"`
	EpisodeNumber   *int    `json:"episode_number"`
	DurationSeconds *int    `json:"duration_seconds"`
}

// UpdateEpisode allows the creator to edit episode metadata (title, number, duration)
func (h *ContentHandler) UpdateEpisode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	episodeID := vars["id"]

	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Load episode and verify ownership via series -> creator_profiles
	var episode models.Episode
	if err := h.db.Joins("JOIN series ON episodes.series_id = series.id").
		Joins("JOIN creator_profiles ON series.creator_id = creator_profiles.id").
		Where("episodes.id = ? AND creator_profiles.user_id = ?", episodeID, userID).
		First(&episode).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Episode not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var req UpdateEpisodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.DurationSeconds != nil {
		if *req.DurationSeconds <= 0 {
			http.Error(w, "duration_seconds must be > 0", http.StatusBadRequest)
			return
		}
		updates["duration_seconds"] = *req.DurationSeconds
	}
	if req.EpisodeNumber != nil {
		if *req.EpisodeNumber <= 0 {
			http.Error(w, "episode_number must be > 0", http.StatusBadRequest)
			return
		}
		// Ensure uniqueness within the same series
		var count int64
		if err := h.db.Model(&models.Episode{}).
			Where("series_id = ? AND episode_number = ? AND id <> ?", episode.SeriesID, *req.EpisodeNumber, episode.ID).
			Count(&count).Error; err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		if count > 0 {
			http.Error(w, "Episode number already exists for this series", http.StatusConflict)
			return
		}
		updates["episode_number"] = *req.EpisodeNumber
	}

	if len(updates) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	if err := h.db.Model(&episode).Updates(updates).Error; err != nil {
		http.Error(w, "Failed to update episode", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Episode updated successfully",
		"id":      episode.ID,
	})
}

// DeleteEpisode allows the creator to delete an episode (soft delete)
func (h *ContentHandler) DeleteEpisode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	episodeID := vars["id"]

	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Verify ownership
	var episode models.Episode
	if err := h.db.Joins("JOIN series ON episodes.series_id = series.id").
		Joins("JOIN creator_profiles ON series.creator_id = creator_profiles.id").
		Where("episodes.id = ? AND creator_profiles.user_id = ?", episodeID, userID).
		First(&episode).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Episode not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if err := h.db.Delete(&episode).Error; err != nil {
		http.Error(w, "Failed to delete episode", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Episode deleted successfully",
		"id":      episode.ID,
	})
}

// GetEpisodes fetches all episodes for a specific series
func (h *ContentHandler) GetEpisodes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	seriesID := vars["seriesId"]

	// Check if series exists and is published
	var series models.Series
	if err := h.db.Where("id = ? AND status = ?", seriesID, "published").First(&series).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Series not found or not published", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Get all published episodes for this series
	var episodes []models.Episode
	if err := h.db.Where("series_id = ? AND status = ?", seriesID, "published").
		Order("episode_number").
		Find(&episodes).Error; err != nil {
		http.Error(w, "Failed to fetch episodes", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	type EpisodeResponse struct {
		ID              string     `json:"id"`
		Title           string     `json:"title"`
		EpisodeNumber   int        `json:"episode_number"`
		DurationSeconds int        `json:"duration_seconds"`
		ThumbURL        *string    `json:"thumb_url"`
		PublishedAt     *time.Time `json:"published_at"`
		CreatedAt       time.Time  `json:"created_at"`
	}

	episodeResponses := make([]EpisodeResponse, 0, len(episodes))
	for _, ep := range episodes {
		episodeResponses = append(episodeResponses, EpisodeResponse{
			ID:              ep.ID,
			Title:           ep.Title,
			EpisodeNumber:   ep.EpisodeNumber,
			DurationSeconds: ep.DurationSeconds,
			ThumbURL:        ep.ThumbURL,
			PublishedAt:     ep.PublishedAt,
			CreatedAt:       ep.CreatedAt,
		})
	}

	response := map[string]interface{}{
		"series_id": seriesID,
		"episodes":  episodeResponses,
		"total":     len(episodeResponses),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
