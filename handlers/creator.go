package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"streamshort/models"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type CreatorHandler struct {
	db *gorm.DB
}

func NewCreatorHandler(db *gorm.DB) *CreatorHandler {
	return &CreatorHandler{db: db}
}

// Request/Response structs matching OpenAPI schema
type CreatorOnboardRequest struct {
	DisplayName     string `json:"display_name"`
	Bio             string `json:"bio"`
	KYCDocumentPath string `json:"kyc_document_s3_path"`
}

type CreatorDashboardResponse struct {
	Views            int64   `json:"views"`
	WatchTimeSeconds int64   `json:"watch_time_seconds"`
	Earnings         float64 `json:"earnings"`
}

// Creator onboarding endpoint
func (h *CreatorHandler) OnboardCreator(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	var req CreatorOnboardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.DisplayName == "" {
		http.Error(w, "Display name is required", http.StatusBadRequest)
		return
	}

	if req.KYCDocumentPath == "" {
		http.Error(w, "KYC document path is required", http.StatusBadRequest)
		return
	}

	// Check if user already has a creator profile
	var existingProfile models.CreatorProfile
	if err := h.db.Where("user_id = ?", userID).First(&existingProfile).Error; err == nil {
		http.Error(w, "Creator profile already exists for this user", http.StatusConflict)
		return
	} else if err != gorm.ErrRecordNotFound {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Create creator profile
	creatorProfile := models.CreatorProfile{
		UserID:          userID,
		DisplayName:     req.DisplayName,
		Bio:             req.Bio,
		KYCDocumentPath: req.KYCDocumentPath,
		KYCStatus:       "pending",
	}

	if err := h.db.Create(&creatorProfile).Error; err != nil {
		http.Error(w, "Failed to create creator profile", http.StatusInternalServerError)
		return
	}

	// Return the created profile
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(creatorProfile)
}

// Creator dashboard endpoint
func (h *CreatorHandler) GetCreatorDashboard(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Get creator ID from URL path
	vars := mux.Vars(r)
	creatorID := vars["id"]

	// Verify that the user is accessing their own dashboard
	var creatorProfile models.CreatorProfile
	if err := h.db.Where("id = ? AND user_id = ?", creatorID, userID).First(&creatorProfile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Creator profile not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Get analytics for the last 30 days
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	var analytics []models.CreatorAnalytics
	if err := h.db.Where("creator_id = ? AND date >= ?", creatorID, thirtyDaysAgo).
		Find(&analytics).Error; err != nil {
		http.Error(w, "Failed to fetch analytics", http.StatusInternalServerError)
		return
	}

	// Aggregate analytics
	var totalViews int64
	var totalWatchTime int64
	var totalEarnings float64

	for _, analytic := range analytics {
		totalViews += analytic.Views
		totalWatchTime += analytic.WatchTimeSeconds
		totalEarnings += analytic.Earnings
	}

	// Create mock analytics if none exist (for development)
	if len(analytics) == 0 {
		// In production, you would calculate real analytics
		totalViews = 1245
		totalWatchTime = 456780
		totalEarnings = 1299.50
	}

	response := CreatorDashboardResponse{
		Views:            totalViews,
		WatchTimeSeconds: totalWatchTime,
		Earnings:         totalEarnings,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get creator profile endpoint
func (h *CreatorHandler) GetCreatorProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Get creator profile for the authenticated user
	var creatorProfile models.CreatorProfile
	if err := h.db.Where("user_id = ?", userID).First(&creatorProfile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Creator profile not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(creatorProfile)
}

// Update creator profile endpoint
func (h *CreatorHandler) UpdateCreatorProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	var req CreatorOnboardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing creator profile
	var creatorProfile models.CreatorProfile
	if err := h.db.Where("user_id = ?", userID).First(&creatorProfile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Creator profile not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Update fields
	if req.DisplayName != "" {
		creatorProfile.DisplayName = req.DisplayName
	}
	if req.Bio != "" {
		creatorProfile.Bio = req.Bio
	}
	if req.KYCDocumentPath != "" {
		creatorProfile.KYCDocumentPath = req.KYCDocumentPath
		// Reset KYC status to pending when document is updated
		creatorProfile.KYCStatus = "pending"
	}

	// Save changes
	if err := h.db.Save(&creatorProfile).Error; err != nil {
		http.Error(w, "Failed to update creator profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(creatorProfile)
}

// Helper function to create mock analytics for testing
func (h *CreatorHandler) CreateMockAnalytics(creatorID string) error {
	// Create analytics for the last 7 days
	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)

		// Generate some realistic mock data
		views := int64(100 + (i * 50) + int(time.Now().Unix()%100))
		watchTime := int64(views * 300)   // 5 minutes average watch time
		earnings := float64(views) * 0.01 // $0.01 per view

		analytic := models.CreatorAnalytics{
			CreatorID:        creatorID,
			Date:             date,
			Views:            views,
			WatchTimeSeconds: watchTime,
			Earnings:         earnings,
		}

		if err := h.db.Create(&analytic).Error; err != nil {
			return err
		}
	}

	return nil
}
