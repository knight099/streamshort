package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type AnalyticsHandler struct {
	db *gorm.DB
	// rpm in USD per 1000 minutes
	rpmUSDPer1000Min float64
}

func NewAnalyticsHandler(db *gorm.DB, rpmUSDPer1000Min float64) *AnalyticsHandler {
	return &AnalyticsHandler{db: db, rpmUSDPer1000Min: rpmUSDPer1000Min}
}

type WatchEventRequest struct {
	EpisodeID      string `json:"episode_id"`
	WatchedSeconds int64  `json:"watched_seconds"`
}

type AnalyticsResponse struct {
	CreatorID        string                 `json:"creator_id"`
	Views            int64                  `json:"views"`
	WatchTimeSeconds int64                  `json:"watch_time_seconds"`
	CreatorProfile   map[string]interface{} `json:"creator_profile"`
}

// WatchResponse is the response for the watch tracking endpoint
type WatchResponse struct {
	Success    bool `json:"success"`
	FirstWatch bool `json:"first_watch"`
	Tracked    bool `json:"tracked"`
}

// POST /analytics/watch
func (h *AnalyticsHandler) RecordWatch(w http.ResponseWriter, r *http.Request) {
	// Get User ID from context (Auth Middleware may or may not be present)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		// Anonymous user - don't track for deduplication purposes
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(WatchResponse{
			Success:    true,
			FirstWatch: false,
			Tracked:    false,
		})
		return
	}

	var req WatchEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EpisodeID == "" || req.WatchedSeconds <= 0 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Resolve creator via episode -> series
	type row struct{ CreatorID string }
	var out row
	if err := h.db.Raw(`
        SELECT s.creator_id AS creator_id
        FROM episodes e
        JOIN series s ON s.id = e.series_id
        WHERE e.id = ?
        LIMIT 1`, req.EpisodeID).Scan(&out).Error; err != nil || out.CreatorID == "" {
		http.Error(w, "Episode or creator not found", http.StatusNotFound)
		return
	}

	// Check if user already has a watch record for this episode
	// Try to insert; if conflict (already exists), do nothing
	result := h.db.Exec(`
		INSERT INTO user_episode_watches (id, user_id, episode_id, watched_seconds, first_watched_at)
		VALUES (gen_random_uuid(), ?, ?, ?, NOW())
		ON CONFLICT (user_id, episode_id)
		DO NOTHING
	`, userID, req.EpisodeID, req.WatchedSeconds)

	if result.Error != nil {
		http.Error(w, "Failed to track watch", http.StatusInternalServerError)
		return
	}

	firstWatch := result.RowsAffected > 0

	// Only update analytics if this is a first-time watch
	if firstWatch {
		// Update Aggregate Creator Stats (Views & Total Watch Time)
		if err := h.db.Exec(`
			INSERT INTO creator_analytics (id, creator_id, date, views, watch_time_seconds, earnings, created_at, updated_at)
			VALUES (gen_random_uuid(), ?, CURRENT_DATE, 1, ?, 0, NOW(), NOW())
			ON CONFLICT (creator_id, date)
			DO UPDATE SET
			  views = creator_analytics.views + 1,
			  watch_time_seconds = creator_analytics.watch_time_seconds + EXCLUDED.watch_time_seconds,
			  updated_at = NOW()`, out.CreatorID, req.WatchedSeconds).Error; err != nil {
			// Log but don't fail
		}

		// Update User-Specific Monthly Stats (For Payout Calculation)
		now := time.Now()
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

		h.db.Exec(`
			INSERT INTO monthly_viewer_stats (id, user_id, creator_id, month_date, watch_time_seconds, created_at, updated_at)
			VALUES (gen_random_uuid(), ?, ?, ?, ?, NOW(), NOW())
			ON CONFLICT (user_id, creator_id, month_date)
			DO UPDATE SET
			  watch_time_seconds = monthly_viewer_stats.watch_time_seconds + EXCLUDED.watch_time_seconds,
			  updated_at = NOW()`, userID, out.CreatorID, firstOfMonth, req.WatchedSeconds)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WatchResponse{
		Success:    true,
		FirstWatch: firstWatch,
		Tracked:    true,
	})
}
