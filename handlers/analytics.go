package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type AnalyticsHandler struct {
	db *gorm.DB
}

func NewAnalyticsHandler(db *gorm.DB) *AnalyticsHandler {
	return &AnalyticsHandler{db: db}
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

// POST /api/analytics/watch
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

	// Verify user exists in database (foreign key will fail otherwise)
	var userExists bool
	h.db.Raw(`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&userExists)
	if !userExists {
		log.Printf("User %s not found in database - may need to complete profile first", userID)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Resolve creator and episode duration via episode -> series
	type episodeInfo struct {
		CreatorID       string
		DurationSeconds int64
	}
	var info episodeInfo
	if err := h.db.Raw(`
        SELECT s.creator_id AS creator_id, e.duration_seconds
        FROM episodes e
        JOIN series s ON s.id = e.series_id
        WHERE e.id = ?
        LIMIT 1`, req.EpisodeID).Scan(&info).Error; err != nil || info.CreatorID == "" {
		http.Error(w, "Episode or creator not found", http.StatusNotFound)
		return
	}

	// Upsert: insert on first watch, ACCUMULATE watched_seconds on subsequent watches
	// Frontend sends incremental watch time (e.g., 10s each heartbeat)
	// CAP at episode duration to prevent gaming (can't earn more than video length)
	// RETURNING is_new tells us if this was an insert (first watch) or update
	var isNewWatch bool
	var err error

	// If duration is 0 or unknown, don't cap (allow unlimited for now)
	if info.DurationSeconds > 0 {
		err = h.db.Raw(`
			INSERT INTO user_episode_watches (id, user_id, episode_id, watched_seconds, first_watched_at)
			VALUES (gen_random_uuid(), $1, $2, LEAST($3::integer, $4::integer), NOW())
			ON CONFLICT (user_id, episode_id)
			DO UPDATE SET
				watched_seconds = LEAST(user_episode_watches.watched_seconds + EXCLUDED.watched_seconds, $4::integer)
			RETURNING (xmax = 0) AS is_new
		`, userID, req.EpisodeID, req.WatchedSeconds, info.DurationSeconds).Scan(&isNewWatch).Error
	} else {
		// No duration cap - just accumulate
		err = h.db.Raw(`
			INSERT INTO user_episode_watches (id, user_id, episode_id, watched_seconds, first_watched_at)
			VALUES (gen_random_uuid(), $1, $2, $3::integer, NOW())
			ON CONFLICT (user_id, episode_id)
			DO UPDATE SET
				watched_seconds = user_episode_watches.watched_seconds + EXCLUDED.watched_seconds
			RETURNING (xmax = 0) AS is_new
		`, userID, req.EpisodeID, req.WatchedSeconds).Scan(&isNewWatch).Error
	}

	if err != nil {
		log.Printf("Failed to track watch for user %s, episode %s: %v", userID, req.EpisodeID, err)
		http.Error(w, "Failed to track watch", http.StatusInternalServerError)
		return
	}

	firstWatch := isNewWatch
	now := time.Now()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Only update view_count and monthly stats if this is a first-time watch
	if firstWatch {
		// Increment episode view_count
		h.db.Exec(`UPDATE episodes SET view_count = view_count + 1 WHERE id = ?`, req.EpisodeID)

		// Update Aggregate Creator Stats (Views & Total Watch Time)
		h.db.Exec(`
			INSERT INTO creator_analytics (id, creator_id, date, views, watch_time_seconds, earnings, created_at, updated_at)
			VALUES (gen_random_uuid(), ?, CURRENT_DATE, 1, ?, 0, NOW(), NOW())
			ON CONFLICT (creator_id, date)
			DO UPDATE SET
			  views = creator_analytics.views + 1,
			  watch_time_seconds = creator_analytics.watch_time_seconds + EXCLUDED.watch_time_seconds,
			  updated_at = NOW()`, info.CreatorID, req.WatchedSeconds)

		// Update User-Specific Monthly Stats (For Payout Calculation)
		h.db.Exec(`
			INSERT INTO monthly_viewer_stats (id, user_id, creator_id, month_date, watch_time_seconds, created_at, updated_at)
			VALUES (gen_random_uuid(), ?, ?, ?, ?, NOW(), NOW())
			ON CONFLICT (user_id, creator_id, month_date)
			DO UPDATE SET
			  watch_time_seconds = monthly_viewer_stats.watch_time_seconds + EXCLUDED.watch_time_seconds,
			  updated_at = NOW()`, userID, info.CreatorID, firstOfMonth, req.WatchedSeconds)
	}

	// Calculate real-time earnings asynchronously using goroutine
	// This runs concurrently and doesn't block the response
	go h.calculateRealtimeEarnings(userID, info.CreatorID, firstOfMonth)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WatchResponse{
		Success:    true,
		FirstWatch: firstWatch,
		Tracked:    true,
	})
}

// calculateRealtimeEarnings calculates and updates creator earnings based on
// user's subscription revenue proportionally distributed by watch time.
// This runs asynchronously in a goroutine.
func (h *AnalyticsHandler) calculateRealtimeEarnings(userID, creatorID string, monthDate time.Time) {
	// 1. Get user's distributable subscription revenue for this month
	var distributable float64
	err := h.db.Raw(`
		SELECT COALESCE(SUM(distributable), 0)
		FROM subscription_revenues
		WHERE user_id = ? AND month_date = ?
	`, userID, monthDate).Scan(&distributable).Error

	if err != nil || distributable == 0 {
		// User has no subscription for this month, no earnings to distribute
		return
	}

	// 2. Get user's total watch time across ALL creators this month
	var totalWatchTime int64
	err = h.db.Raw(`
		SELECT COALESCE(SUM(watch_time_seconds), 0)
		FROM monthly_viewer_stats
		WHERE user_id = ? AND month_date = ?
	`, userID, monthDate).Scan(&totalWatchTime).Error

	if err != nil || totalWatchTime == 0 {
		return
	}

	// 3. Get user's watch time for THIS specific creator this month
	var creatorWatchTime int64
	err = h.db.Raw(`
		SELECT COALESCE(watch_time_seconds, 0)
		FROM monthly_viewer_stats
		WHERE user_id = ? AND creator_id = ? AND month_date = ?
	`, userID, creatorID, monthDate).Scan(&creatorWatchTime).Error

	if err != nil || creatorWatchTime == 0 {
		return
	}

	// 4. Calculate this creator's proportional share of the user's subscription
	// Formula: (creator_watch_time / total_watch_time) * distributable_amount
	shareFraction := float64(creatorWatchTime) / float64(totalWatchTime)
	creatorEarnings := distributable * shareFraction

	// 5. Update creator's estimated earnings in creator_analytics for today
	if err := h.db.Exec(`
		UPDATE creator_analytics 
		SET earnings = ?, updated_at = NOW()
		WHERE creator_id = ? AND date = CURRENT_DATE
	`, creatorEarnings, creatorID).Error; err != nil {
		log.Printf("Failed to update creator earnings: %v", err)
	}

	// 6. Upsert into creator_payouts for this month (running estimate)
	h.db.Exec(`
		INSERT INTO creator_payouts (id, creator_id, month_date, total_earnings, status, created_at, updated_at)
		VALUES (gen_random_uuid(), ?, ?, ?, 'pending', NOW(), NOW())
		ON CONFLICT (creator_id, month_date)
		DO UPDATE SET
			total_earnings = EXCLUDED.total_earnings,
			updated_at = NOW()
	`, creatorID, monthDate, creatorEarnings)
}
