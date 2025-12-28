package handlers

import (
	"encoding/json"
	"net/http"

	"streamshort/models"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type WatchListHandler struct {
	db *gorm.DB
}

func NewWatchListHandler(db *gorm.DB) *WatchListHandler {
	return &WatchListHandler{db: db}
}

// AddToWatchList adds a series to the user's watch list
// POST /api/me/watchlist
// Body: { "series_id": "uuid" }
// AddToWatchList godoc
// @Summary Add to Watchlist
// @Description Add a series to the user's watchlist.
// @Tags watchlist
// @Accept  json
// @Produce  json
// @Param   request  body     map[string]string  true  "Series ID (JSON: {series_id: string})"
// @Success 201 {object} map[string]string
// @Failure 400 {string} string "Bad Request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/me/watchlist [post]
func (h *WatchListHandler) AddToWatchList(w http.ResponseWriter, r *http.Request) {
	userIDAny := r.Context().Value("user_id")
	if userIDAny == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, _ := userIDAny.(string)

	var req struct {
		SeriesID string `json:"series_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.SeriesID == "" {
		http.Error(w, "series_id is required", http.StatusBadRequest)
		return
	}

	// Verify series exists
	var series models.Series
	if err := h.db.First(&series, "id = ?", req.SeriesID).Error; err != nil {
		http.Error(w, "Series not found", http.StatusNotFound)
		return
	}

	// Create watch list entry
	watchList := models.WatchList{
		UserID:   userID,
		SeriesID: req.SeriesID,
	}

	// Use FirstOrCreate to handle duplicates idempotently
	result := h.db.Where(models.WatchList{UserID: userID, SeriesID: req.SeriesID}).
		FirstOrCreate(&watchList)

	if result.Error != nil {
		http.Error(w, "Failed to add to watchlist", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Added to watchlist"})
}

// RemoveFromWatchList removes a series from the user's watch list
// DELETE /api/me/watchlist/{series_id}
// RemoveFromWatchList godoc
// @Summary Remove from Watchlist
// @Description Remove a series from the user's watchlist.
// @Tags watchlist
// @Produce  json
// @Param   series_id  path     string  true  "Series ID"
// @Success 200 {object} map[string]string
// @Failure 400 {string} string "Bad Request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/me/watchlist/{series_id} [delete]
func (h *WatchListHandler) RemoveFromWatchList(w http.ResponseWriter, r *http.Request) {
	userIDAny := r.Context().Value("user_id")
	if userIDAny == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, _ := userIDAny.(string)

	vars := mux.Vars(r)
	seriesID := vars["series_id"]
	if seriesID == "" {
		http.Error(w, "series_id is required", http.StatusBadRequest)
		return
	}

	if err := h.db.Where("user_id = ? AND series_id = ?", userID, seriesID).
		Delete(&models.WatchList{}).Error; err != nil {
		http.Error(w, "Failed to remove from watchlist", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Removed from watchlist"})
}

// GetWatchList returns the user's watch list
// GET /api/me/watchlist
// GetWatchList godoc
// @Summary Get Watchlist
// @Description Get the user's watchlist.
// @Tags watchlist
// @Produce  json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/me/watchlist [get]
func (h *WatchListHandler) GetWatchList(w http.ResponseWriter, r *http.Request) {
	userIDAny := r.Context().Value("user_id")
	if userIDAny == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, _ := userIDAny.(string)

	var watchList []models.WatchList
	if err := h.db.Preload("Series").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&watchList).Error; err != nil {
		http.Error(w, "Failed to fetch watchlist", http.StatusInternalServerError)
		return
	}

	// Transform response to return list of series or custom structure
	// Let's return the series details directly for easier frontend consumption
	type WatchListResponse struct {
		Series []models.Series `json:"series"`
	}

	seriesList := make([]models.Series, 0)
	for _, item := range watchList {
		if item.Series != nil {
			seriesList = append(seriesList, *item.Series)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WatchListResponse{Series: seriesList})
}

// CheckWatchListStatus checks if a specific series is in the user's watchlist
// GET /api/me/watchlist/{series_id}/status
// CheckWatchListStatus godoc
// @Summary Check Watchlist Status
// @Description Check if a series is in the user's watchlist.
// @Tags watchlist
// @Produce  json
// @Param   series_id  path     string  true  "Series ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {string} string "Bad Request"
// @Failure 401 {string} string "Unauthorized"
// @Router /api/me/watchlist/{series_id}/status [get]
func (h *WatchListHandler) CheckWatchListStatus(w http.ResponseWriter, r *http.Request) {
	userIDAny := r.Context().Value("user_id")
	if userIDAny == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, _ := userIDAny.(string)

	vars := mux.Vars(r)
	seriesID := vars["series_id"]
	if seriesID == "" {
		http.Error(w, "series_id is required", http.StatusBadRequest)
		return
	}

	var count int64
	h.db.Model(&models.WatchList{}).
		Where("user_id = ? AND series_id = ?", userID, seriesID).
		Count(&count)

	json.NewEncoder(w).Encode(map[string]bool{"in_watchlist": count > 0})
}
