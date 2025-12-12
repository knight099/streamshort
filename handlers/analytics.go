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

// POST /analytics/watch
func (h *AnalyticsHandler) RecordWatch(w http.ResponseWriter, r *http.Request) {
	// Get User ID from context (Auth Middleware required)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		// Fallback for public testing or strict enforcement?
		// For now, let's enforce it. If your Flutter app doesn't send token for this endpoint yet,
		// you might need to relax this or ensure token is sent.
		// http.Error(w, "Unauthorized", http.StatusUnauthorized)
		// return

		// TEMP: Allow anonymous for backward compatibility if needed, but payouts won't work correctly without UserID.
		// For proper Payouts, we NEED userID.
		userID = "anonymous"
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

	// 1. Update Aggregate Creator Stats (Views & Total Watch Time) - Keep this for real-time dashboard
	// Note: We are NO LONGER calculating earnings here.
	// Use ON CONFLICT (creator_id, date) to create one row per creator per day for proper 30-day analytics
	if err := h.db.Exec(`
        INSERT INTO creator_analytics (id, creator_id, date, views, watch_time_seconds, earnings, created_at, updated_at)
        VALUES (gen_random_uuid(), ?, CURRENT_DATE, 1, ?, 0, NOW(), NOW())
        ON CONFLICT (creator_id, date)
        DO UPDATE SET
          views = creator_analytics.views + 1,
          watch_time_seconds = creator_analytics.watch_time_seconds + EXCLUDED.watch_time_seconds,
          updated_at = NOW()`, out.CreatorID, req.WatchedSeconds).Error; err != nil {
		http.Error(w, "Failed to update analytics", http.StatusInternalServerError)
		return
	}

	// 2. Update User-Specific Monthly Stats (For Payout Calculation)
	if userID != "anonymous" {
		// Calculate the first day of the current month
		now := time.Now()
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

		if err := h.db.Exec(`
			INSERT INTO monthly_viewer_stats (id, user_id, creator_id, month_date, watch_time_seconds, created_at, updated_at)
			VALUES (gen_random_uuid(), ?, ?, ?, ?, NOW(), NOW())
			ON CONFLICT (user_id, creator_id, month_date)
			DO UPDATE SET
			  watch_time_seconds = monthly_viewer_stats.watch_time_seconds + EXCLUDED.watch_time_seconds,
			  updated_at = NOW()`, userID, out.CreatorID, firstOfMonth, req.WatchedSeconds).Error; err != nil {
			// Log error but don't fail request?
			// log.Printf("Failed to update viewer stats: %v", err)
		}
	}

	// Return updated aggregate with basic creator profile fields
	var resp AnalyticsResponse
	if err := h.db.Raw(`
        SELECT ca.creator_id,
               ca.views,
               ca.watch_time_seconds,
               json_build_object(
                   'id', cp.id,
                   'user_id', cp.user_id,
                   'display_name', cp.display_name
               ) AS creator_profile
        FROM creator_analytics ca
        JOIN creator_profiles cp ON cp.id = ca.creator_id
        WHERE ca.creator_id = ?
        LIMIT 1`, out.CreatorID).Scan(&resp).Error; err != nil {
		http.Error(w, "Failed to load analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
