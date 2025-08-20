package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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
	Action string `json:"action"` // "like" or "unlike"
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

type CommentRequest struct {
	Content string `json:"content"`
}

type CommentResponse struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	UserID    string    `json:"user_id"`
	EpisodeID string    `json:"episode_id"`
	CreatedAt time.Time `json:"created_at"`
}

// LikeEpisode handles episode likes/unlikes
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

	// Validate action
	if req.Action != "like" && req.Action != "unlike" {
		http.Error(w, "Action must be 'like' or 'unlike'", http.StatusBadRequest)
		return
	}

	// Mock like handling (in real implementation, save to database)
	var likeCount int64 = 42 // Mock count
	isLiked := req.Action == "like"

	// In real implementation, you'd save this to database with userID and episodeID
	_ = userID    // Use userID to avoid linter warning
	_ = episodeID // Use episodeID to avoid linter warning

	if req.Action == "like" {
		likeCount++
	} else {
		if likeCount > 0 {
			likeCount--
		}
	}

	response := LikeResponse{
		Status:    "success",
		LikeCount: likeCount,
		IsLiked:   isLiked,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RateEpisode handles episode ratings
func (h *SocialHandler) RateEpisode(w http.ResponseWriter, r *http.Request) {
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

	// Mock rating handling (in real implementation, save to database)
	_ = userID    // Use userID to avoid linter warning
	_ = episodeID // Use episodeID to avoid linter warning

	// Mock average rating calculation
	averageRating := 4.2
	totalRatings := int64(156)

	response := RatingResponse{
		Status:        "success",
		Rating:        req.Rating,
		AverageRating: averageRating,
		TotalRatings:  totalRatings,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CommentEpisode handles episode comments
func (h *SocialHandler) CommentEpisode(w http.ResponseWriter, r *http.Request) {
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

	var req CommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate content
	if req.Content == "" {
		http.Error(w, "Comment content is required", http.StatusBadRequest)
		return
	}

	// Mock comment creation (in real implementation, save to database)
	commentID := "comment_" + strconv.FormatInt(time.Now().Unix(), 10)
	now := time.Now()

	response := CommentResponse{
		ID:        commentID,
		Content:   req.Content,
		UserID:    userID,
		EpisodeID: episodeID,
		CreatedAt: now,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
