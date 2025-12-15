package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"streamshort/models"

	"gorm.io/gorm"
)

type UserHandler struct {
	db *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

type UserProfileResponse struct {
	ID           string  `json:"id"`
	Phone        string  `json:"phone"`
	Name         string  `json:"name"`
	IsSubscribed bool    `json:"is_subscribed"`
	PlanName     *string `json:"plan_name,omitempty"`
	ExpiresAt    *string `json:"subscription_expires_at,omitempty"`
}

type UpdateProfileRequest struct {
	Name string `json:"name"`
}

// GetProfile returns the authenticated user's profile with subscription status
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userIDAny := r.Context().Value("user_id")
	if userIDAny == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, _ := userIDAny.(string)

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check subscription status
	var subscription models.Subscription
	isSubscribed := false
	var planName *string
	var expiresAt *string

	err := h.db.Where("user_id = ? AND status = 'active' AND end_date > ?", userID, time.Now()).
		Preload("Plan").
		First(&subscription).Error

	if err == nil {
		isSubscribed = true
		if subscription.Plan != nil {
			planName = &subscription.Plan.Name
		}
		formatted := subscription.EndDate.Format(time.RFC3339)
		expiresAt = &formatted
	}

	resp := UserProfileResponse{
		ID:           user.ID,
		Phone:        user.Phone,
		Name:         user.Name,
		IsSubscribed: isSubscribed,
		PlanName:     planName,
		ExpiresAt:    expiresAt,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// UpdateProfile updates the authenticated user's name
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userIDAny := r.Context().Value("user_id")
	if userIDAny == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, _ := userIDAny.(string)

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Name) == 0 || len(req.Name) > 100 {
		http.Error(w, "Invalid name", http.StatusBadRequest)
		return
	}

	if err := h.db.Model(&models.User{}).Where("id = ?", userID).Update("name", req.Name).Error; err != nil {
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	resp := UserProfileResponse{ID: user.ID, Phone: user.Phone, Name: user.Name}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// WatchHistoryItem represents a single item in the user's watch history
type WatchHistoryItem struct {
	EpisodeID      string  `json:"episode_id"`
	EpisodeTitle   string  `json:"episode_title"`
	EpisodeNumber  int     `json:"episode_number"`
	SeriesID       string  `json:"series_id"`
	SeriesTitle    string  `json:"series_title"`
	ThumbnailURL   *string `json:"thumbnail_url"`
	WatchedSeconds int     `json:"watched_seconds"`
	FirstWatchedAt string  `json:"first_watched_at"`
}

// WatchHistoryResponse is the response for the watch history endpoint
type WatchHistoryResponse struct {
	Items []WatchHistoryItem `json:"items"`
	Total int                `json:"total"`
}

// GetWatchHistory returns the authenticated user's episode watch history
// GET /api/me/watch-history
func (h *UserHandler) GetWatchHistory(w http.ResponseWriter, r *http.Request) {
	userIDAny := r.Context().Value("user_id")
	if userIDAny == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, _ := userIDAny.(string)

	// Query watch history with episode and series details
	var items []WatchHistoryItem
	err := h.db.Raw(`
		SELECT 
			uew.episode_id,
			e.title as episode_title,
			e.episode_number,
			s.id as series_id,
			s.title as series_title,
			COALESCE(e.thumb_url, s.thumbnail_url) as thumbnail_url,
			uew.watched_seconds,
			uew.first_watched_at
		FROM user_episode_watches uew
		JOIN episodes e ON e.id = uew.episode_id
		JOIN series s ON s.id = e.series_id
		WHERE uew.user_id = ?
		ORDER BY uew.first_watched_at DESC
		LIMIT 100
	`, userID).Scan(&items).Error

	if err != nil {
		http.Error(w, "Failed to fetch watch history", http.StatusInternalServerError)
		return
	}

	if items == nil {
		items = []WatchHistoryItem{}
	}

	resp := WatchHistoryResponse{
		Items: items,
		Total: len(items),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
