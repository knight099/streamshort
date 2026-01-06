package handlers

import (
	"encoding/json"
	"net/http"
	"episodd/models"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type SocialHandler struct {
	db *gorm.DB
}

func NewSocialHandler(db *gorm.DB) *SocialHandler {
	return &SocialHandler{db: db}
}

// Request/Response structs matching OpenAPI schema
type LikeRequest struct {
	Action string `json:"action"` // "like" or "unlike" (legacy format)
	Like   *bool  `json:"like"`   // true = like, false = unlike (new format)
}

type LikeResponse struct {
	Status    string `json:"status"`
	LikeCount int64  `json:"like_count"`
	IsLiked   bool   `json:"is_liked"`
}

type RatingRequest struct {
	Rating int `json:"rating"` // 1-5 stars
}

type RatingResponse struct {
	Status        string  `json:"status"`
	Rating        int     `json:"rating"`
	AverageRating float64 `json:"average_rating"`
	TotalRatings  int64   `json:"total_ratings"`
}

// LikedEpisodeItem represents a single liked episode with details
type LikedEpisodeItem struct {
	EpisodeID     string  `json:"episode_id"`
	EpisodeTitle  string  `json:"episode_title"`
	EpisodeNumber int     `json:"episode_number"`
	SeriesID      string  `json:"series_id"`
	SeriesTitle   string  `json:"series_title"`
	ThumbnailURL  *string `json:"thumbnail_url"`
	LikeCount     int64   `json:"like_count"`
	LikedAt       string  `json:"liked_at"`
}

// LikedEpisodesResponse is the response for the liked episodes endpoint
type LikedEpisodesResponse struct {
	Items []LikedEpisodeItem `json:"items"`
	Total int                `json:"total"`
}

// GetLikeStatus returns the like status for an episode
// GET /api/episodes/{id}/like
// GetLikeStatus godoc
// @Summary Get Like Status
// @Description Get the like status and count for an episode.
// @Tags social
// @Produce  json
// @Param   id   path     string  true  "Episode ID"
// @Success 200 {object} LikeResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/episodes/{id}/like [get]
func (h *SocialHandler) GetLikeStatus(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Get episode ID from URL
	vars := mux.Vars(r)
	episodeID := vars["id"]
	if episodeID == "" {
		http.Error(w, "Episode ID is required", http.StatusBadRequest)
		return
	}

	// Get like count from episode table
	var episode models.Episode
	if err := h.db.Select("like_count").Where("id = ?", episodeID).First(&episode).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Episode not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check if user has liked
	var userLikeCount int64
	h.db.Model(&models.EpisodeLike{}).Where("user_id = ? AND episode_id = ?", userID, episodeID).Count(&userLikeCount)
	isLiked := userLikeCount > 0

	response := LikeResponse{
		Status:    "success",
		LikeCount: episode.LikeCount,
		IsLiked:   isLiked,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// LikeEpisode handles episode likes/unlikes
// LikeEpisode godoc
// @Summary Like/Unlike Episode
// @Description Like or unlike an episode.
// @Tags social
// @Accept  json
// @Produce  json
// @Param   id       path     string       true  "Episode ID"
// @Param   request  body     LikeRequest  true  "Like Action"
// @Success 200 {object} LikeResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/episodes/{id}/like [post]
func (h *SocialHandler) LikeEpisode(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Get episode ID from URL
	vars := mux.Vars(r)
	episodeID := vars["id"]
	if episodeID == "" {
		http.Error(w, "Episode ID is required", http.StatusBadRequest)
		return
	}

	var req LikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Normalize the action: accept both "action" string and "like" boolean
	var isLikeAction bool
	if req.Like != nil {
		// New format: {"like": true/false}
		isLikeAction = *req.Like
	} else if req.Action == "like" {
		isLikeAction = true
	} else if req.Action == "unlike" {
		isLikeAction = false
	} else {
		http.Error(w, "Action must be 'like' or 'unlike', or provide 'like': true/false", http.StatusBadRequest)
		return
	}

	// Handle like/unlike in database
	if isLikeAction {
		// Check if already liked
		var existingLike models.EpisodeLike
		err := h.db.Where("user_id = ? AND episode_id = ?", userID, episodeID).First(&existingLike).Error
		if err == gorm.ErrRecordNotFound {
			// Create new like
			like := models.EpisodeLike{
				UserID:    userID,
				EpisodeID: episodeID,
			}
			if err := h.db.Create(&like).Error; err != nil {
				http.Error(w, "Failed to like episode", http.StatusInternalServerError)
				return
			}
			// Increment episode like_count
			h.db.Exec("UPDATE episodes SET like_count = like_count + 1 WHERE id = ?", episodeID)
		}
		// If already exists, do nothing (idempotent)
	} else {
		// Unlike: delete the like record
		result := h.db.Where("user_id = ? AND episode_id = ?", userID, episodeID).Delete(&models.EpisodeLike{})
		if result.Error != nil {
			http.Error(w, "Failed to unlike episode", http.StatusInternalServerError)
			return
		}
		// Decrement episode like_count only if a row was actually deleted
		if result.RowsAffected > 0 {
			h.db.Exec("UPDATE episodes SET like_count = GREATEST(like_count - 1, 0) WHERE id = ?", episodeID)
		}
	}

	// Get updated like count from episode table
	var episode models.Episode
	h.db.Select("like_count").Where("id = ?", episodeID).First(&episode)

	// Check if user has liked
	var userLikeCount int64
	h.db.Model(&models.EpisodeLike{}).Where("user_id = ? AND episode_id = ?", userID, episodeID).Count(&userLikeCount)
	isLiked := userLikeCount > 0

	response := LikeResponse{
		Status:    "success",
		LikeCount: episode.LikeCount,
		IsLiked:   isLiked,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RateSeries handles series ratings (1-5 stars)
// RateSeries godoc
// @Summary Rate Series
// @Description Rate a series (1-5 stars).
// @Tags social
// @Accept  json
// @Produce  json
// @Param   id       path     string         true  "Series ID"
// @Param   request  body     RatingRequest  true  "Rating"
// @Success 200 {object} RatingResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/series/{id}/rating [post]
func (h *SocialHandler) RateSeries(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Get series ID from URL
	vars := mux.Vars(r)
	seriesID := vars["id"]
	if seriesID == "" {
		http.Error(w, "Series ID is required", http.StatusBadRequest)
		return
	}

	var req RatingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate rating (1-5 stars)
	if req.Rating < 1 || req.Rating > 5 {
		http.Error(w, "Rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	// Upsert rating: update if exists, create if not
	var existingRating models.SeriesRating
	err := h.db.Where("user_id = ? AND series_id = ?", userID, seriesID).First(&existingRating).Error
	switch err {
	case gorm.ErrRecordNotFound:
		// Create new rating
		rating := models.SeriesRating{
			UserID:   userID,
			SeriesID: seriesID,
			Score:    req.Rating,
		}
		if err := h.db.Create(&rating).Error; err != nil {
			http.Error(w, "Failed to create rating", http.StatusInternalServerError)
			return
		}
	case nil:
		// Update existing rating
		if err := h.db.Model(&existingRating).Update("score", req.Rating).Error; err != nil {
			http.Error(w, "Failed to update rating", http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Calculate average rating for this series
	var avgResult struct {
		AvgScore float64
		Count    int64
	}
	h.db.Model(&models.SeriesRating{}).
		Where("series_id = ?", seriesID).
		Select("AVG(score) as avg_score, COUNT(*) as count").
		Scan(&avgResult)

	response := RatingResponse{
		Status:        "success",
		Rating:        req.Rating,
		AverageRating: avgResult.AvgScore,
		TotalRatings:  avgResult.Count,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetLikedEpisodes returns all episodes liked by the authenticated user
// GET /api/me/liked-episodes
// GetLikedEpisodes godoc
// @Summary Get Liked Episodes
// @Description Get list of episodes liked by the user.
// @Tags social
// @Produce  json
// @Success 200 {object} LikedEpisodesResponse
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/me/liked-episodes [get]
func (h *SocialHandler) GetLikedEpisodes(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Query liked episodes with episode and series details
	var items []LikedEpisodeItem
	err := h.db.Raw(`
		SELECT 
			el.episode_id,
			e.title as episode_title,
			e.episode_number,
			s.id as series_id,
			s.title as series_title,
			COALESCE(e.thumb_url, s.thumbnail_url) as thumbnail_url,
			e.like_count,
			el.created_at as liked_at
		FROM episode_likes el
		JOIN episodes e ON e.id = el.episode_id
		JOIN series s ON s.id = e.series_id
		WHERE el.user_id = ? AND el.deleted_at IS NULL
		ORDER BY el.created_at DESC
		LIMIT 100
	`, userID).Scan(&items).Error

	if err != nil {
		http.Error(w, "Failed to fetch liked episodes", http.StatusInternalServerError)
		return
	}

	if items == nil {
		items = []LikedEpisodeItem{}
	}

	response := LikedEpisodesResponse{
		Items: items,
		Total: len(items),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
