package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"streamshort/models"
	"streamshort/services"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type ContentHandler struct {
	db  *gorm.DB
	rdb *redis.Client
	aws *services.AWSService
}

func NewContentHandler(db *gorm.DB) *ContentHandler {
	return &ContentHandler{db: db}
}

func NewContentHandlerWithServices(db *gorm.DB, rdb *redis.Client, aws *services.AWSService) *ContentHandler {
	return &ContentHandler{db: db, rdb: rdb, aws: aws}
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
	Title           string  `json:"title"`
	SeasonNumber    int     `json:"season_number"`
	EpisodeNumber   int     `json:"episode_number"`
	DurationSeconds int     `json:"duration_seconds"`
	ThumbURL        *string `json:"thumb_url"`
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
	SeasonNumber    int        `json:"season_number"`
	EpisodeNumber   int        `json:"episode_number"`
	DurationSeconds int        `json:"duration_seconds"`
	ThumbURL        *string    `json:"thumb_url"`
	PublishedAt     *time.Time `json:"published_at"`
	CreatedAt       time.Time  `json:"created_at"`
	ViewCount       int64      `json:"view_count"`
	LikeCount       int64      `json:"like_count"`
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
	EpisodeID string `json:"episode_id"` // Optional: link upload to episode
}

type UploadNotifyResponse struct {
	Status string `json:"status"`
}

type ManifestResponse struct {
	ManifestURL string    `json:"manifest_url"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type SeriesDetailResponse struct {
	ID            string         `json:"id"`
	CreatorID     string         `json:"creator_id"`
	CreatorName   *string        `json:"creator_name"`
	Title         string         `json:"title"`
	Synopsis      string         `json:"synopsis"`
	Language      string         `json:"language"`
	CategoryTags  pq.StringArray `json:"category_tags"`
	PriceType     string         `json:"price_type"`
	PriceAmount   *float64       `json:"price_amount"`
	ThumbnailURL  *string        `json:"thumbnail_url"`
	Status        string         `json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	Episodes      []EpisodeBrief `json:"episodes"`
	Seasons       []int          `json:"seasons"`
	TotalSeasons  int            `json:"total_seasons"`
	FollowerCount int64          `json:"follower_count"`
	Following     bool           `json:"following"`
	ViewCount     int64          `json:"view_count"`
	LikeCount     int64          `json:"like_count"`
}

// CreateSeries creates a new series
// CreateSeries godoc
// @Summary Create a new series
// @Description Create a new series entry. Requires creator privileges.
// @Tags content
// @Accept  json
// @Produce  json
// @Param   series  body     CreateSeriesRequest  true  "Series creation request"
// @Success 201 {object} models.Series
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "Forbidden"
// @Failure 500 {string} string "Internal Server Error"
// @Router /content/series [post]
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

	// Best-effort: index series in Elasticsearch via HTTP if configured
	if esURL := os.Getenv("ELASTICSEARCH_URL"); esURL != "" {
		go func(s models.Series) {
			body, _ := json.Marshal(s)
			req, err := http.NewRequest("PUT", fmt.Sprintf("%s/series/_doc/%s", strings.TrimRight(esURL, "/"), s.ID), strings.NewReader(string(body)))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Do(req)
			if err == nil && resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
		}(series)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(series)
}

// ListSeries lists series with optional filters
// ListSeries godoc
// @Summary List series
// @Description Get a list of series with optional filtering
// @Tags content
// @Accept  json
// @Produce  json
// @Param   language     query    string     false        "Language filter"
// @Param   category     query    string     false        "Category filter"
// @Param   page         query    int        false        "Page number"
// @Param   per_page     query    int        false        "Items per page"
// @Success 200 {object} SeriesListResponse
// @Failure 500 {string} string "Internal Server Error"
// @Router /content/series [get]
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
		Preload("Episodes", func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", "published").Order("episode_number")
		})

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
				ViewCount:       ep.ViewCount,
				LikeCount:       ep.LikeCount,
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

// SearchSeries godoc
// @Summary Search series
// @Description Search for series by title or synopsis
// @Tags content
// @Accept  json
// @Produce  json
// @Param   q            query    string     true         "Search query"
// @Param   page         query    int        false        "Page number"
// @Param   limit        query    int        false        "Results per page"
// @Success 200 {object} SeriesListResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /content/series/search [get]
func (h *ContentHandler) SearchSeries(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	if q == "" {
		http.Error(w, "q is required", http.StatusBadRequest)
		return
	}

	page := 1
	limit := 20
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	// Cache key
	cacheKey := fmt.Sprintf("series:search:%s:%d:%d", strings.ToLower(q), page, limit)

	// Try Redis cache
	if h.rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		if cached, err := h.rdb.Get(ctx, cacheKey).Bytes(); err == nil && len(cached) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.Write(cached)
			return
		}
	}

	// DB ILIKE search
	var response SeriesListResponse
	var seriesRows []models.Series
	like := "%" + q + "%"

	// Build base query conditions
	baseQuery := h.db.Model(&models.Series{}).
		Where("status = ?", "published").
		Where("title ILIKE ? OR synopsis ILIKE ?", like, like)

	// Count total (use separate query to avoid GORM state mutation)
	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Fetch paginated results (fresh query to avoid Count side effects)
	if err := h.db.Model(&models.Series{}).
		Where("status = ?", "published").
		Where("title ILIKE ? OR synopsis ILIKE ?", like, like).
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&seriesRows).Error; err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	items := make([]SeriesListItem, 0, len(seriesRows))
	for _, s := range seriesRows {
		items = append(items, SeriesListItem{
			ID:           s.ID,
			CreatorID:    s.CreatorID,
			CreatorName:  nil,
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
		})
	}
	response = SeriesListResponse{Total: total, Items: items}

	// Cache response
	if h.rdb != nil {
		if b, err := json.Marshal(response); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_ = h.rdb.Set(ctx, cacheKey, b, 5*time.Minute).Err()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSeries godoc
// @Summary Get series details
// @Description Get detailed information about a series
// @Tags content
// @Accept  json
// @Produce  json
// @Param   id           path     string     true         "Series ID"
// @Success 200 {object} SeriesDetailResponse
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /content/series/{id} [get]
func (h *ContentHandler) GetSeries(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	seriesID := vars["id"]

	var series models.Series
	if err := h.db.Preload("Creator").Preload("Episodes", func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", "published").Order("season_number, episode_number")
	}).Where("id = ?", seriesID).First(&series).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Series not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var creatorName *string
	if series.Creator != nil {
		creatorName = &series.Creator.DisplayName
	}

	eps := make([]EpisodeBrief, 0, len(series.Episodes))
	seasonSet := make(map[int]bool)
	for _, ep := range series.Episodes {
		seasonSet[ep.SeasonNumber] = true
		eps = append(eps, EpisodeBrief{
			ID:              ep.ID,
			Title:           ep.Title,
			SeasonNumber:    ep.SeasonNumber,
			EpisodeNumber:   ep.EpisodeNumber,
			DurationSeconds: ep.DurationSeconds,
			ThumbURL:        ep.ThumbURL,
			PublishedAt:     ep.PublishedAt,
			CreatedAt:       ep.CreatedAt,
			ViewCount:       ep.ViewCount,
			LikeCount:       ep.LikeCount,
		})
	}

	// Build sorted seasons list
	seasons := make([]int, 0, len(seasonSet))
	for s := range seasonSet {
		seasons = append(seasons, s)
	}
	sort.Ints(seasons)

	// follower count
	var followerCount int64
	if err := h.db.Model(&models.CreatorFollow{}).Where("creator_id = ?", series.CreatorID).Count(&followerCount).Error; err != nil {
		followerCount = 0
	}
	// following status (auth optional)
	userID, _ := r.Context().Value("user_id").(string)
	following := false
	if userID != "" {
		var c int64
		_ = h.db.Model(&models.CreatorFollow{}).Where("follower_id = ? AND creator_id = ?", userID, series.CreatorID).Count(&c).Error
		following = c > 0
	}

	// Get total like count for all episodes in this series
	var likeCount int64
	if err := h.db.Model(&models.EpisodeLike{}).
		Joins("JOIN episodes ON episodes.id = episode_likes.episode_id").
		Where("episodes.series_id = ?", series.ID).
		Count(&likeCount).Error; err != nil {
		likeCount = 0
	}

	// Get total view count from series_views table (sum of all view_count values)
	var viewCount int64
	if err := h.db.Model(&models.SeriesView{}).
		Where("series_id = ?", series.ID).
		Select("COALESCE(SUM(view_count), 0)").
		Scan(&viewCount).Error; err != nil {
		// Fallback to series.ViewCount if query fails
		viewCount = series.ViewCount
	}

	resp := SeriesDetailResponse{
		ID:            series.ID,
		CreatorID:     series.CreatorID,
		CreatorName:   creatorName,
		Title:         series.Title,
		Synopsis:      series.Synopsis,
		Language:      series.Language,
		CategoryTags:  series.CategoryTags,
		PriceType:     series.PriceType,
		PriceAmount:   series.PriceAmount,
		ThumbnailURL:  series.ThumbnailURL,
		Status:        series.Status,
		CreatedAt:     series.CreatedAt,
		UpdatedAt:     series.UpdatedAt,
		Episodes:      eps,
		Seasons:       seasons,
		TotalSeasons:  len(seasons),
		FollowerCount: followerCount,
		Following:     following,
		ViewCount:     viewCount,
		LikeCount:     likeCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// UpdateSeries updates a series
// UpdateSeries godoc
// @Summary Update Series
// @Description Update series details.
// @Tags content
// @Accept  json
// @Produce  json
// @Param   id       path     string               true  "Series ID"
// @Param   request  body     UpdateSeriesRequest  true  "Update Request"
// @Success 200 {object} map[string]string
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /content/series/{id} [put]
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

	// Best-effort: reindex series in Elasticsearch
	if h.rdb != nil {
		go func(id string) {
			var s models.Series
			if err := h.db.Where("id = ?", id).First(&s).Error; err == nil {
				_ = s // placeholder for future indexing integration
			}
		}(series.ID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Series updated successfully"})
}

// CreateEpisode creates episode metadata for a series
// CreateEpisode godoc
// @Summary Create Episode
// @Description Create a new episode for a series.
// @Tags content
// @Accept  json
// @Produce  json
// @Param   id       path     string                true  "Series ID"
// @Param   request  body     CreateEpisodeRequest  true  "Create Request"
// @Success 201 {object} models.Episode
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Failure 409 {string} string "Conflict"
// @Failure 500 {string} string "Internal Server Error"
// @Router /content/series/{id}/episodes [post]
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

	// Check if episode number already exists for this season
	var existingEpisode models.Episode
	seasonNum := req.SeasonNumber
	if seasonNum <= 0 {
		seasonNum = 1 // Default to season 1
	}
	if err := h.db.Where("series_id = ? AND season_number = ? AND episode_number = ?", seriesID, seasonNum, req.EpisodeNumber).First(&existingEpisode).Error; err == nil {
		http.Error(w, "Episode number already exists for this season", http.StatusConflict)
		return
	}

	// Create episode
	episode := models.Episode{
		SeriesID:        seriesID,
		Title:           req.Title,
		SeasonNumber:    seasonNum,
		EpisodeNumber:   req.EpisodeNumber,
		DurationSeconds: req.DurationSeconds,
		ThumbURL:        req.ThumbURL,
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
// RequestUploadURL godoc
// @Summary Request Upload URL
// @Description Request a presigned URL for uploading content (video/thumbnail).
// @Tags content
// @Accept  json
// @Produce  json
// @Param   request  body     UploadUrlRequest  true  "Upload Request"
// @Success 200 {object} UploadUrlResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /content/upload-url [post]
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

	// Detect file type from content type or metadata
	fileType := "video" // default
	if req.Metadata != nil {
		if typeVal, ok := req.Metadata["type"].(string); ok {
			fileType = typeVal
		}
	}
	// If not specified in metadata, detect from content type
	if fileType == "video" && strings.HasPrefix(req.ContentType, "image/") {
		fileType = "thumbnail"
	}

	// Validate file type
	if fileType != "video" && fileType != "thumbnail" && fileType != "caption" {
		http.Error(w, "Invalid file type. Must be 'video', 'thumbnail', or 'caption'", http.StatusBadRequest)
		return
	}

	// Additional validation based on file type
	switch fileType {
	case "thumbnail":
		// Validate it's an image
		if !strings.HasPrefix(req.ContentType, "image/") {
			http.Error(w, "Thumbnail must be an image file", http.StatusBadRequest)
			return
		}
		// Check size limit (5MB for thumbnails)
		if req.SizeBytes > 5*1024*1024 {
			http.Error(w, "Thumbnail size must not exceed 5MB", http.StatusBadRequest)
			return
		}
	case "video":
		// Validate it's a video
		if !strings.HasPrefix(req.ContentType, "video/") {
			http.Error(w, "Video file must have video/* content type", http.StatusBadRequest)
			return
		}
	}

	// Create upload request record (ID is auto-generated as UUID by the database)
	uploadReq := models.UploadRequest{
		UserID:      userID,
		Filename:    req.Filename,
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
		FileType:    fileType,
		Metadata:    req.Metadata,
		Status:      "pending",
	}

	if err := h.db.Create(&uploadReq).Error; err != nil {
		// Log detailed error for debugging
		fmt.Printf("ERROR: Failed to create upload request: %v\n", err)
		fmt.Printf("Upload request details: UserID=%s, Filename=%s, ContentType=%s, SizeBytes=%d, FileType=%s\n",
			userID, req.Filename, req.ContentType, req.SizeBytes, fileType)
		if req.Metadata != nil {
			metaJSON, _ := json.Marshal(req.Metadata)
			fmt.Printf("Metadata: %s\n", string(metaJSON))
		}
		http.Error(w, "Failed to create upload request", http.StatusInternalServerError)
		return
	}

	// Use the database-generated UUID as the upload ID
	uploadID := uploadReq.ID

	var presignedURL string
	expiresIn := 3600 // 1 hour

	// Generate real S3 pre-signed URL if AWS is configured
	if h.aws != nil && h.aws.IsConfigured() {
		var err error
		presignedURL, err = h.aws.GenerateUploadPresignedURLWithType(
			r.Context(),
			uploadID,
			req.Filename,
			req.ContentType,
			fileType,
			time.Duration(expiresIn)*time.Second,
		)
		if err != nil {
			http.Error(w, "Failed to generate upload URL", http.StatusInternalServerError)
			return
		}
	} else {
		// Fallback to mock URL for development
		presignedURL = fmt.Sprintf("https://s3.amazonaws.com/bucket/%s/%s?AWSAccessKeyId=mock&Signature=mock", fileType, uploadID)
	}

	response := UploadUrlResponse{
		UploadID:     uploadID,
		PresignedURL: presignedURL,
		ExpiresIn:    expiresIn,
		UploadHeaders: map[string]string{
			"Content-Type": req.ContentType,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// NotifyUploadComplete handles upload completion notification
// NotifyUploadComplete godoc
// @Summary Notify Upload Complete
// @Description Notify the server that a file upload has completed.
// @Tags content
// @Accept  json
// @Produce  json
// @Param   upload_id  path     string               true  "Upload ID"
// @Param   request    body     UploadNotifyRequest  true  "Notify Request"
// @Success 200 {object} UploadNotifyResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /content/uploads/{upload_id}/notify [post]
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

	// Get upload request to determine file type
	var uploadReq models.UploadRequest
	if err := h.db.Where("id = ? AND user_id = ?", uploadID, userID).First(&uploadReq).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Upload request not found", http.StatusNotFound)
			return
		}
		// Log the actual error for debugging
		fmt.Printf("ERROR: Failed to query upload_requests table: %v\n", err)
		fmt.Printf("Upload ID: %s, User ID: %s\n", uploadID, userID)
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	// Update upload request status
	if err := h.db.Model(&uploadReq).
		Updates(map[string]interface{}{
			"status":     "completed",
			"updated_at": time.Now(),
		}).Error; err != nil {
		http.Error(w, "Failed to update upload status", http.StatusInternalServerError)
		return
	}

	// Handle based on file type
	if uploadReq.FileType == "thumbnail" {
		// Check if series_id is in metadata
		if uploadReq.Metadata != nil {
			if seriesIDVal, ok := uploadReq.Metadata["series_id"]; ok {
				seriesID := fmt.Sprintf("%v", seriesIDVal)

				// Verify ownership: series owned by this creator
				var series models.Series
				err := h.db.Joins("JOIN creator_profiles ON series.creator_id = creator_profiles.id").
					Where("series.id = ? AND creator_profiles.user_id = ?", seriesID, userID).
					First(&series).Error

				if err == nil {
					// Generate public URL from S3 path
					// For now, use the S3 path directly; in production, this would be a CloudFront or public S3 URL
					thumbnailURL := req.S3Path
					if h.aws != nil && h.aws.IsConfigured() {
						// Convert s3:// path to HTTPS URL
						// Format: https://bucket-name.s3.region.amazonaws.com/key
						key := h.aws.ExtractS3Key(req.S3Path)
						if key != "" {
							thumbnailURL = fmt.Sprintf("https://%s/%s", h.aws.GetCloudFrontDomain(), key)
						}
					}

					// Update series thumbnail_url
					if err := h.db.Model(&series).Updates(map[string]interface{}{
						"thumbnail_url": thumbnailURL,
						"updated_at":    time.Now(),
					}).Error; err != nil {
						fmt.Printf("Warning: Failed to update series thumbnail_url: %v\n", err)
					}
				}
			}
		}

		// Return success for thumbnail upload
		response := UploadNotifyResponse{
			Status: "completed",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Handle video uploads (existing logic)
	if req.EpisodeID != "" {
		// Verify ownership: episode belongs to a series owned by this creator
		var episode models.Episode
		err := h.db.Joins("JOIN series ON episodes.series_id = series.id").
			Joins("JOIN creator_profiles ON series.creator_id = creator_profiles.id").
			Where("episodes.id = ? AND creator_profiles.user_id = ?", req.EpisodeID, userID).
			First(&episode).Error

		if err == nil {
			// Update episode with the S3 path
			if err := h.db.Model(&episode).Updates(map[string]interface{}{
				"s3_master_path": req.S3Path,
				"status":         "queued_transcode",
				"updated_at":     time.Now(),
			}).Error; err != nil {
				// Log error but don't fail the request
				fmt.Printf("Warning: Failed to update episode s3_master_path: %v\n", err)
			}
		}
	}

	// TODO: In production, trigger transcoding job here for videos
	response := UploadNotifyResponse{
		Status: "queued_for_transcoding",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// GetEpisodeManifest gets signed HLS manifest URL for playback
// GetEpisodeManifest godoc
// @Summary Get Episode Manifest
// @Description Get the HLS manifest or video URL for an episode.
// @Tags content
// @Produce  json
// @Param   id   path     string  true  "Episode ID"
// @Success 200 {object} ManifestResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /episodes/{id}/manifest [get]
func (h *ContentHandler) GetEpisodeManifest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	episodeID := vars["id"]

	// Get user ID from context (for subscription checks)
	userID, ok := r.Context().Value("user_id").(string)
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

	// Check if user has access to this content based on series pricing
	if episode.Series.PriceType != "free" {
		// Allow access if user has any active subscription (global/all-access)
		var subscription models.Subscription
		err := h.db.Where("user_id = ? AND status = ?", userID, "active").
			Order("end_date DESC").First(&subscription).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Subscription required to access this content", http.StatusForbidden)
				return
			}
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Check if subscription is still active (not expired)
		if subscription.IsExpired() {
			// Update status to expired
			h.db.Model(&subscription).Update("status", "expired")
			http.Error(w, "Subscription has expired", http.StatusForbidden)
			return
		}
	}

	var videoURL string
	var expiresAt time.Time
	expiresIn := 1 * time.Hour

	// Check if we have HLS manifest URL (transcoded content)
	if episode.HLSManifestURL != nil && *episode.HLSManifestURL != "" {
		// Use transcoded HLS manifest
		fmt.Printf("[Manifest] Episode %s has HLS manifest, generating signed URL\n", episodeID)
		if h.aws != nil && h.aws.IsCloudFrontConfigured() {
			var err error
			videoURL, expiresAt, err = h.aws.GenerateManifestURL(episodeID, expiresIn)
			if err != nil {
				fmt.Printf("[Manifest Error] Failed to generate manifest URL: %v\n", err)
				http.Error(w, "Failed to generate manifest URL", http.StatusInternalServerError)
				return
			}
		} else {
			expiresAt = time.Now().Add(expiresIn)
			videoURL = *episode.HLSManifestURL
		}
	} else if episode.S3MasterPath != nil && *episode.S3MasterPath != "" {
		// No HLS yet - serve the raw MP4 via CloudFront (fallback for untranscoded content)
		fmt.Printf("[Manifest] Episode %s has no HLS manifest yet, falling back to raw MP4\n", episodeID)
		fmt.Printf("[Manifest] WARNING: Raw MP4 playback may not work if CloudFront 'uploads/*' behavior is not configured for signed URLs\n")
		// No HLS yet - serve the raw MP4 via CloudFront (for development)
		// Extract the path from s3://bucket/path format
		s3Path := *episode.S3MasterPath
		var objectPath string

		if strings.HasPrefix(s3Path, "s3://") {
			// Remove s3:// scheme
			withoutScheme := strings.TrimPrefix(s3Path, "s3://")
			// Find the first slash to separate bucket from key
			slashIndex := strings.Index(withoutScheme, "/")
			if slashIndex != -1 && slashIndex+1 < len(withoutScheme) {
				objectPath = withoutScheme[slashIndex+1:]
			}
		}

		if objectPath != "" {
			// Ensure object path is properly URL encoded if needed, but AWS SDK might handle signing correctly
			// For now, let's just pass the path.
			// Debug log (in production use proper logger)
			fmt.Printf("[Manifest] Generating CloudFront URL for raw MP4 path: %s\n", objectPath)

			if h.aws != nil && h.aws.IsCloudFrontConfigured() {
				var err error
				videoURL, expiresAt, err = h.aws.GenerateCloudFrontSignedURL(objectPath, expiresIn)
				if err != nil {
					fmt.Printf("[Manifest Error] Failed to generate CloudFront signed URL: %v\n", err)
					http.Error(w, "Failed to generate video URL", http.StatusInternalServerError)
					return
				}
				fmt.Printf("[Manifest] Successfully generated signed URL for raw MP4\n")
			} else {
				// Fallback: direct CloudFront URL without signing (for dev)
				expiresAt = time.Now().Add(expiresIn)
				cloudFrontDomain := "djoxr6aauqbf6.cloudfront.net"
				if h.aws != nil && h.aws.GetCloudFrontDomain() != "" {
					cloudFrontDomain = h.aws.GetCloudFrontDomain()
				}
				videoURL = fmt.Sprintf("https://%s/%s", cloudFrontDomain, objectPath)
			}
		} else {
			// Fallback to simple split if s3:// prefix not found or parsing failed
			// This might be for paths stored without s3:// prefix
			pathParts := strings.SplitN(s3Path, "/", 4)
			if len(pathParts) >= 4 {
				// ... existing logic fallback ...
				objectPath = pathParts[3]
				// duplicate logic... let's just fail if path is invalid
			}

			if objectPath == "" {
				// Try to use the path as is if it doesn't look like a full S3 URI
				if !strings.HasPrefix(s3Path, "s3://") {
					objectPath = s3Path
				} else {
					http.Error(w, "Invalid S3 path format", http.StatusInternalServerError)
					return
				}
			}

			// Retry generation with fallback path
			if h.aws != nil && h.aws.IsCloudFrontConfigured() {
				var err error
				videoURL, expiresAt, err = h.aws.GenerateCloudFrontSignedURL(objectPath, expiresIn)
				if err != nil {
					http.Error(w, "Failed to generate video URL", http.StatusInternalServerError)
					return
				}
			} else {
				expiresAt = time.Now().Add(expiresIn)
				cloudFrontDomain := "djoxr6aauqbf6.cloudfront.net"
				if h.aws != nil && h.aws.GetCloudFrontDomain() != "" {
					cloudFrontDomain = h.aws.GetCloudFrontDomain()
				}
				videoURL = fmt.Sprintf("https://%s/%s", cloudFrontDomain, objectPath)
			}
		}
	} else {
		// No video content available
		http.Error(w, "Episode video not available", http.StatusNotFound)
		return
	}

	response := ManifestResponse{
		ManifestURL: videoURL,
		ExpiresAt:   expiresAt,
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
// GetCreatorContent godoc
// @Summary Get Creator Content
// @Description Get all series and episodes created by the authenticated creator.
// @Tags creator
// @Produce  json
// @Success 200 {object} CreatorContentResponse
// @Failure 403 {string} string "Forbidden"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/creators/content [get]
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
// UpdateEpisodeStatus godoc
// @Summary Update Episode Status
// @Description Update the status of an episode.
// @Tags content
// @Accept  json
// @Produce  json
// @Param   id       path     string                      true  "Episode ID"
// @Param   request  body     UpdateEpisodeStatusRequest  true  "Update Request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /content/episodes/{id}/status [put]
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

	// Block publishing if creator KYC is not verified
	if status == "published" {
		var creatorProfile models.CreatorProfile
		if err := h.db.Where("user_id = ?", userID).First(&creatorProfile).Error; err != nil {
			http.Error(w, "Creator profile not found", http.StatusNotFound)
			return
		}
		if creatorProfile.KYCStatus != "verified" {
			http.Error(w, "KYC verification required before publishing. Please wait for admin approval.", http.StatusForbidden)
			return
		}
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
	ThumbURL        *string `json:"thumb_url"`
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
	if req.ThumbURL != nil {
		updates["thumb_url"] = *req.ThumbURL
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

	// Delete S3 objects before deleting from database
	ctx := r.Context()
	if h.aws != nil && h.aws.IsConfigured() {
		// Delete master video file if exists
		if episode.S3MasterPath != nil && *episode.S3MasterPath != "" {
			if err := h.aws.DeleteS3Object(ctx, *episode.S3MasterPath); err != nil {
				// Log error but continue with deletion
				fmt.Printf("Warning: Failed to delete S3 master file %s: %v\n", *episode.S3MasterPath, err)
			}
		}

		// Delete transcoded folder (HLS segments, manifest, etc.)
		transcodedPrefix := fmt.Sprintf("transcoded/%s", episodeID)
		if err := h.aws.DeleteS3ObjectsByPrefix(ctx, transcodedPrefix); err != nil {
			// Log error but continue with deletion
			fmt.Printf("Warning: Failed to delete S3 transcoded folder %s: %v\n", transcodedPrefix, err)
		}

		// Delete thumbnail if exists
		if episode.ThumbURL != nil && *episode.ThumbURL != "" {
			if err := h.aws.DeleteS3Object(ctx, *episode.ThumbURL); err != nil {
				// Log error but continue with deletion
				fmt.Printf("Warning: Failed to delete S3 thumbnail %s: %v\n", *episode.ThumbURL, err)
			}
		}

		// Delete captions if exists
		if episode.CaptionsURL != nil && *episode.CaptionsURL != "" {
			if err := h.aws.DeleteS3Object(ctx, *episode.CaptionsURL); err != nil {
				// Log error but continue with deletion
				fmt.Printf("Warning: Failed to delete S3 captions %s: %v\n", *episode.CaptionsURL, err)
			}
		}
	}

	// Delete from database (soft delete)
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

	// Optional season filter
	seasonStr := r.URL.Query().Get("season")
	var seasonFilter *int
	if seasonStr != "" {
		if s, err := strconv.Atoi(seasonStr); err == nil && s > 0 {
			seasonFilter = &s
		}
	}

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

	// Get all published episodes for this series (with optional season filter)
	var episodes []models.Episode
	query := h.db.Where("series_id = ? AND status = ?", seriesID, "published")
	if seasonFilter != nil {
		query = query.Where("season_number = ?", *seasonFilter)
	}
	if err := query.Order("season_number, episode_number").Find(&episodes).Error; err != nil {
		http.Error(w, "Failed to fetch episodes", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	type EpisodeResponse struct {
		ID              string     `json:"id"`
		Title           string     `json:"title"`
		SeasonNumber    int        `json:"season_number"`
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
			SeasonNumber:    ep.SeasonNumber,
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
	if seasonFilter != nil {
		response["season_number"] = *seasonFilter
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
