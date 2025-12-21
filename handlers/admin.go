package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"streamshort/models"
	"streamshort/services"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db  *gorm.DB
	aws *services.AWSService
}

func NewAdminHandler(db *gorm.DB, aws *services.AWSService) *AdminHandler {
	return &AdminHandler{
		db:  db,
		aws: aws,
	}
}

// Request/Response structs matching OpenAPI schema
type PendingUpload struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	SizeBytes   int64     `json:"size_bytes"`
	ContentType string    `json:"content_type"`
	UploadedAt  time.Time `json:"uploaded_at"`
	CreatorID   string    `json:"creator_id"`
	SeriesID    string    `json:"series_id"`
	EpisodeID   string    `json:"episode_id"`
	Status      string    `json:"status"`
}

type PendingUploadsResponse struct {
	Total int64           `json:"total"`
	Items []PendingUpload `json:"items"`
}

type ApproveContentRequest struct {
	Action string `json:"action"` // "approve" or "reject"
	Reason string `json:"reason"` // Required if action is "reject"
	Notes  string `json:"notes"`  // Optional admin notes
}

type ApproveContentResponse struct {
	Status      string    `json:"status"`
	Action      string    `json:"action"`
	ProcessedAt time.Time `json:"processed_at"`
	AdminID     string    `json:"admin_id"`
}

// GetPendingUploads lists all pending uploads for admin review
func (h *AdminHandler) GetPendingUploads(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you'd check if the user has admin privileges
	// For now, we'll assume this endpoint is protected by admin middleware

	// Mock pending uploads data
	pendingUploads := []PendingUpload{
		{
			ID:          uuid.New().String(),
			Filename:    "episode1_master.mp4",
			SizeBytes:   73400320,
			ContentType: "video/mp4",
			UploadedAt:  time.Now().Add(-2 * time.Hour),
			CreatorID:   "creator_123",
			SeriesID:    "series_456",
			EpisodeID:   "episode_789",
			Status:      "pending_review",
		},
		{
			ID:          uuid.New().String(),
			Filename:    "episode2_master.mp4",
			SizeBytes:   81234567,
			ContentType: "video/mp4",
			UploadedAt:  time.Now().Add(-1 * time.Hour),
			CreatorID:   "creator_124",
			SeriesID:    "series_457",
			EpisodeID:   "episode_790",
			Status:      "pending_review",
		},
	}

	response := PendingUploadsResponse{
		Total: int64(len(pendingUploads)),
		Items: pendingUploads,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ApproveContent handles content approval/rejection
func (h *AdminHandler) ApproveContent(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you'd check if the user has admin privileges
	// For now, we'll assume this endpoint is protected by admin middleware

	var req ApproveContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate action
	if req.Action != "approve" && req.Action != "reject" {
		http.Error(w, "Action must be 'approve' or 'reject'", http.StatusBadRequest)
		return
	}

	// Validate reason for rejection
	if req.Action == "reject" && req.Reason == "" {
		http.Error(w, "Reason is required when rejecting content", http.StatusBadRequest)
		return
	}

	// Mock content approval processing
	adminID := "admin_001" // In real implementation, get from context
	now := time.Now()

	response := ApproveContentResponse{
		Status:      "success",
		Action:      req.Action,
		ProcessedAt: now,
		AdminID:     adminID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// AdminDeleteUser deletes a user (and cascadingly soft-deletes their creator profile)
func (h *AdminHandler) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	// Check if user exists
	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Begin transaction
	tx := h.db.Begin()

	// Soft delete creator profile if exists
	if err := tx.Where("user_id = ?", userID).Delete(&models.CreatorProfile{}).Error; err != nil {
		tx.Rollback()
		http.Error(w, "Failed to delete creator profile", http.StatusInternalServerError)
		return
	}

	// Soft delete user
	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	tx.Commit()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User deleted successfully"})
}

// AdminDeleteCreator deletes a creator profile (downgrades user to viewer)
func (h *AdminHandler) AdminDeleteCreator(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creatorID := vars["id"]

	if err := h.db.Where("id = ?", creatorID).Delete(&models.CreatorProfile{}).Error; err != nil {
		http.Error(w, "Failed to delete creator profile", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Creator profile deleted successfully"})
}

// AdminDeleteSeries deletes a series
func (h *AdminHandler) AdminDeleteSeries(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	seriesID := vars["id"]

	// Update status to deleted (optional, but soft delete handles visibility)
	if err := h.db.Where("id = ?", seriesID).Delete(&models.Series{}).Error; err != nil {
		http.Error(w, "Failed to delete series", http.StatusInternalServerError)
		return
	}

	// Soft delete all episodes
	h.db.Where("series_id = ?", seriesID).Delete(&models.Episode{})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Series deleted successfully"})
}

// AdminDeleteEpisode deletes an episode
func (h *AdminHandler) AdminDeleteEpisode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	episodeID := vars["id"]

	var episode models.Episode
	if err := h.db.First(&episode, "id = ?", episodeID).Error; err != nil {
		http.Error(w, "Episode not found", http.StatusNotFound)
		return
	}

	// Delete S3 assets
	ctx := r.Context()
	if h.aws != nil && h.aws.IsConfigured() {
		if episode.S3MasterPath != nil && *episode.S3MasterPath != "" {
			_ = h.aws.DeleteS3Object(ctx, *episode.S3MasterPath)
		}
		transcodedPrefix := fmt.Sprintf("transcoded/%s", episodeID)
		_ = h.aws.DeleteS3ObjectsByPrefix(ctx, transcodedPrefix)

		if episode.ThumbURL != nil && *episode.ThumbURL != "" {
			_ = h.aws.DeleteS3Object(ctx, *episode.ThumbURL)
		}
	}

	if err := h.db.Delete(&episode).Error; err != nil {
		http.Error(w, "Failed to delete episode", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Episode deleted successfully"})
}

type AdminCreateSeriesRequest struct {
	CreatorID    string   `json:"creator_id"`
	Title        string   `json:"title"`
	Synopsis     string   `json:"synopsis"`
	Language     string   `json:"language"`
	CategoryTags []string `json:"category_tags"`
	PriceType    string   `json:"price_type"`
	PriceAmount  *float64 `json:"price_amount"`
	ThumbnailURL *string  `json:"thumbnail_url"`
}

// AdminListCreators lists all creators for selection
func (h *AdminHandler) AdminListCreators(w http.ResponseWriter, r *http.Request) {
	var creators []models.CreatorProfile
	if err := h.db.Find(&creators).Error; err != nil {
		http.Error(w, "Failed to fetch creators", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(creators)
}

// AdminCreateSeries creates a series on behalf of a creator
func (h *AdminHandler) AdminCreateSeries(w http.ResponseWriter, r *http.Request) {
	var req AdminCreateSeriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CreatorID == "" || req.Title == "" {
		http.Error(w, "CreatorID and Title are required", http.StatusBadRequest)
		return
	}

	// Verify creator exists
	var creator models.CreatorProfile
	if err := h.db.First(&creator, "id = ?", req.CreatorID).Error; err != nil {
		http.Error(w, "Creator not found", http.StatusNotFound)
		return
	}

	series := models.Series{
		CreatorID:    req.CreatorID,
		Title:        req.Title,
		Synopsis:     req.Synopsis,
		Language:     req.Language,
		CategoryTags: pq.StringArray(req.CategoryTags),
		PriceType:    req.PriceType,
		PriceAmount:  req.PriceAmount,
		ThumbnailURL: req.ThumbnailURL,
		Status:       "published", // Admins probably want it published or draft? Let's default to published or maybe draft? User request says "upload videos...". Let's stick to 'draft' like normal flow or 'published' since admin doing it? Let's use 'draft' to be safe.
	}
	series.Status = "draft"

	if err := h.db.Create(&series).Error; err != nil {
		http.Error(w, "Failed to create series", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(series)
}

// AdminCreateEpisode creates an episode for any series
func (h *AdminHandler) AdminCreateEpisode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	seriesID := vars["id"]

	var req CreateEpisodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if series exists
	var series models.Series
	if err := h.db.First(&series, "id = ?", seriesID).Error; err != nil {
		http.Error(w, "Series not found", http.StatusNotFound)
		return
	}

	// Check if episode number exists
	var count int64
	h.db.Model(&models.Episode{}).Where("series_id = ? AND episode_number = ?", seriesID, req.EpisodeNumber).Count(&count)
	if count > 0 {
		http.Error(w, "Episode number already exists", http.StatusConflict)
		return
	}

	episode := models.Episode{
		SeriesID:        seriesID,
		Title:           req.Title,
		EpisodeNumber:   req.EpisodeNumber,
		DurationSeconds: req.DurationSeconds,
		ThumbURL:        req.ThumbURL,
		Status:          "pending_upload",
	}

	if err := h.db.Create(&episode).Error; err != nil {
		http.Error(w, "Failed to create episode", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(episode)
}

// AdminRequestUploadURL generates upload URL for admin
func (h *AdminHandler) AdminRequestUploadURL(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("user_id").(string) // Admin's user ID

	var req UploadUrlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Basic validation matches ContentHandler
	fileType := "video"
	if req.Metadata != nil {
		if t, ok := req.Metadata["type"].(string); ok {
			fileType = t
		}
	}
	if fileType == "video" && strings.HasPrefix(req.ContentType, "image/") {
		fileType = "thumbnail"
	}

	uploadReq := models.UploadRequest{
		UserID:      userID, // Admin uploaded it
		Filename:    req.Filename,
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
		FileType:    fileType,
		Metadata:    req.Metadata,
		Status:      "pending",
	}

	if err := h.db.Create(&uploadReq).Error; err != nil {
		http.Error(w, "Failed to create upload request", http.StatusInternalServerError)
		return
	}

	uploadID := uploadReq.ID
	var presignedURL string
	expiresIn := 3600

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
		presignedURL = fmt.Sprintf("https://mock-s3.com/%s/%s", fileType, uploadID)
	}

	resp := UploadUrlResponse{
		UploadID:      uploadID,
		PresignedURL:  presignedURL,
		ExpiresIn:     expiresIn,
		UploadHeaders: map[string]string{"Content-Type": req.ContentType},
	}
	json.NewEncoder(w).Encode(resp)
}

// AdminNotifyUploadComplete handles upload completion for admin
func (h *AdminHandler) AdminNotifyUploadComplete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uploadID := vars["id"]

	var req UploadNotifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var uploadReq models.UploadRequest
	if err := h.db.First(&uploadReq, "id = ?", uploadID).Error; err != nil {
		http.Error(w, "Upload request not found", http.StatusNotFound)
		return
	}

	uploadReq.Status = "completed"
	uploadReq.UpdatedAt = time.Now()
	h.db.Save(&uploadReq)

	// Update Series/Episode without ownership checks
	if uploadReq.FileType == "thumbnail" {
		if uploadReq.Metadata != nil {
			if sIDVal, ok := uploadReq.Metadata["series_id"]; ok {
				seriesID := fmt.Sprintf("%v", sIDVal)

				// S3 URL generation logic
				url := req.S3Path
				if h.aws != nil && h.aws.IsConfigured() {
					key := h.aws.ExtractS3Key(req.S3Path)
					if key != "" {
						url = fmt.Sprintf("https://%s.s3.amazonaws.com/%s", h.aws.GetBucket(), key)
					}
				}
				h.db.Model(&models.Series{}).Where("id = ?", seriesID).Update("thumbnail_url", url)
			}
		}
	} else if req.EpisodeID != "" {
		h.db.Model(&models.Episode{}).Where("id = ?", req.EpisodeID).Updates(map[string]interface{}{
			"s3_master_path": req.S3Path,
			"status":         "queued_transcode", // Assume transcoding is needed
			"updated_at":     time.Now(),
		})
	}

	json.NewEncoder(w).Encode(UploadNotifyResponse{Status: "completed"})
}

// ========== ADMIN SUBSCRIPTION & FINANCE TRACKING ==========

// AdminListSubscriptions lists all subscriptions with optional filters
func (h *AdminHandler) AdminListSubscriptions(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status") // active, expired, cancelled, pending

	query := h.db.Model(&models.Subscription{}).Preload("User").Preload("Plan")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var subscriptions []models.Subscription
	if err := query.Order("created_at DESC").Find(&subscriptions).Error; err != nil {
		http.Error(w, "Failed to fetch subscriptions", http.StatusInternalServerError)
		return
	}

	// Get total count
	var total int64
	h.db.Model(&models.Subscription{}).Count(&total)

	// Get active count
	var activeCount int64
	h.db.Model(&models.Subscription{}).Where("status = ?", "active").Count(&activeCount)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total":        total,
		"active_count": activeCount,
		"items":        subscriptions,
	})
}

// AdminListPayouts lists all payouts with optional status filter
func (h *AdminHandler) AdminListPayouts(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status") // pending, processing, completed, failed

	query := h.db.Model(&models.Payout{}).Preload("Creator")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var payouts []models.Payout
	if err := query.Order("requested_at DESC").Find(&payouts).Error; err != nil {
		http.Error(w, "Failed to fetch payouts", http.StatusInternalServerError)
		return
	}

	// Build response with creator_name
	type PayoutItem struct {
		ID             string     `json:"id"`
		CreatorID      string     `json:"creator_id"`
		CreatorName    string     `json:"creator_name"`
		Amount         float64    `json:"amount"`
		Status         string     `json:"status"`
		TransactionRef *string    `json:"transaction_ref"`
		CreatedAt      time.Time  `json:"created_at"`
		ProcessedAt    *time.Time `json:"processed_at"`
	}

	items := make([]PayoutItem, 0, len(payouts))
	for _, p := range payouts {
		creatorName := ""
		if p.Creator != nil {
			creatorName = p.Creator.DisplayName
		}

		items = append(items, PayoutItem{
			ID:             p.ID,
			CreatorID:      p.CreatorID,
			CreatorName:    creatorName,
			Amount:         p.Amount,
			Status:         p.Status,
			TransactionRef: p.TransactionReference,
			CreatedAt:      p.CreatedAt,
			ProcessedAt:    p.CompletedAt,
		})
	}

	// Get total amounts by status
	var totalPending, totalProcessing, totalCompleted float64
	h.db.Model(&models.Payout{}).Where("status = ?", "pending").Select("COALESCE(SUM(amount), 0)").Scan(&totalPending)
	h.db.Model(&models.Payout{}).Where("status = ?", "processing").Select("COALESCE(SUM(amount), 0)").Scan(&totalProcessing)
	h.db.Model(&models.Payout{}).Where("status = ?", "completed").Select("COALESCE(SUM(amount), 0)").Scan(&totalCompleted)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items":            items,
		"total_pending":    totalPending,
		"total_processing": totalProcessing,
		"total_completed":  totalCompleted,
	})
}

// AdminGetRevenueSummary provides platform-wide revenue summary
func (h *AdminHandler) AdminGetRevenueSummary(w http.ResponseWriter, r *http.Request) {
	// Total subscription revenue
	var totalSubscriptionRevenue float64
	h.db.Model(&models.Subscription{}).Where("status = ?", "active").Select("COALESCE(SUM(amount), 0)").Scan(&totalSubscriptionRevenue)

	// Total platform fee from subscription_revenues
	var totalPlatformFee float64
	h.db.Model(&models.SubscriptionRevenue{}).Select("COALESCE(SUM(platform_fee), 0)").Scan(&totalPlatformFee)

	// Total distributable to creators
	var totalDistributable float64
	h.db.Model(&models.SubscriptionRevenue{}).Select("COALESCE(SUM(distributable), 0)").Scan(&totalDistributable)

	// Creator earnings summary
	var totalCreatorEarnings float64
	h.db.Model(&models.CreatorEarnings{}).Where("status != ?", "cancelled").Select("COALESCE(SUM(amount), 0)").Scan(&totalCreatorEarnings)

	var pendingCreatorEarnings float64
	h.db.Model(&models.CreatorEarnings{}).Where("status = ?", "pending").Select("COALESCE(SUM(amount), 0)").Scan(&pendingCreatorEarnings)

	var paidCreatorEarnings float64
	h.db.Model(&models.CreatorEarnings{}).Where("status = ?", "paid").Select("COALESCE(SUM(amount), 0)").Scan(&paidCreatorEarnings)

	// Subscription counts
	var activeSubscriptions, totalSubscriptions int64
	h.db.Model(&models.Subscription{}).Where("status = ?", "active").Count(&activeSubscriptions)
	h.db.Model(&models.Subscription{}).Count(&totalSubscriptions)

	// Total users and creators
	var totalUsers, totalCreators int64
	h.db.Model(&models.User{}).Count(&totalUsers)
	h.db.Model(&models.CreatorProfile{}).Count(&totalCreators)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"subscription_revenue": map[string]interface{}{
			"total_collected":       totalSubscriptionRevenue,
			"platform_fee_earned":   totalPlatformFee,
			"creator_distributable": totalDistributable,
		},
		"creator_earnings": map[string]interface{}{
			"total":   totalCreatorEarnings,
			"pending": pendingCreatorEarnings,
			"paid":    paidCreatorEarnings,
		},
		"subscriptions": map[string]interface{}{
			"active": activeSubscriptions,
			"total":  totalSubscriptions,
		},
		"users": map[string]interface{}{
			"total_users":    totalUsers,
			"total_creators": totalCreators,
		},
	})
}

// AdminListEarnings lists all creator earnings with optional filters
func (h *AdminHandler) AdminListEarnings(w http.ResponseWriter, r *http.Request) {
	creatorID := r.URL.Query().Get("creator_id")
	status := r.URL.Query().Get("status")     // pending, paid, cancelled
	earningsType := r.URL.Query().Get("type") // subscription, one_time, ad_revenue

	query := h.db.Model(&models.CreatorEarnings{}).Preload("Creator")

	if creatorID != "" {
		query = query.Where("creator_id = ?", creatorID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if earningsType != "" {
		query = query.Where("earnings_type = ?", earningsType)
	}

	var earnings []models.CreatorEarnings
	if err := query.Order("created_at DESC").Find(&earnings).Error; err != nil {
		http.Error(w, "Failed to fetch earnings", http.StatusInternalServerError)
		return
	}

	// Total by status
	var totalPending, totalPaid float64
	h.db.Model(&models.CreatorEarnings{}).Where("status = ?", "pending").Select("COALESCE(SUM(amount), 0)").Scan(&totalPending)
	h.db.Model(&models.CreatorEarnings{}).Where("status = ?", "paid").Select("COALESCE(SUM(amount), 0)").Scan(&totalPaid)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items":         earnings,
		"total_pending": totalPending,
		"total_paid":    totalPaid,
	})
}

// AdminUpdatePayoutStatus allows admin to update payout status
func (h *AdminHandler) AdminUpdatePayoutStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	payoutID := vars["id"]

	var req struct {
		Status         string  `json:"status"` // processing, completed, failed
		TransactionRef *string `json:"transaction_ref"`
		FailureReason  *string `json:"failure_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updates := map[string]interface{}{
		"status":     req.Status,
		"updated_at": time.Now(),
	}

	if req.Status == "completed" {
		now := time.Now()
		updates["completed_at"] = &now
	}
	if req.TransactionRef != nil {
		updates["transaction_reference"] = *req.TransactionRef
	}
	if req.FailureReason != nil {
		updates["failure_reason"] = *req.FailureReason
	}

	if err := h.db.Model(&models.Payout{}).Where("id = ?", payoutID).Updates(updates).Error; err != nil {
		http.Error(w, "Failed to update payout", http.StatusInternalServerError)
		return
	}

	// If completed, mark associated earnings as paid
	if req.Status == "completed" {
		h.db.Model(&models.CreatorEarnings{}).Where("payout_id = ?", payoutID).Updates(map[string]interface{}{
			"status":  "paid",
			"paid_at": time.Now(),
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Payout status updated"})
}

// ========== ADMIN DASHBOARD & MANAGEMENT ==========

// AdminDashboardStats returns overview statistics for the admin dashboard
func (h *AdminHandler) AdminDashboardStats(w http.ResponseWriter, r *http.Request) {
	var totalUsers int64
	h.db.Model(&models.User{}).Count(&totalUsers)

	var activeCreators int64
	h.db.Model(&models.CreatorProfile{}).Count(&activeCreators)

	// Count pending content (series or episodes with status 'pending' or 'pending_review')
	var pendingApprovals int64
	h.db.Model(&models.Series{}).Where("status IN ?", []string{"pending", "pending_review"}).Count(&pendingApprovals)
	var pendingEpisodes int64
	h.db.Model(&models.Episode{}).Where("status IN ?", []string{"pending", "pending_review", "pending_upload"}).Count(&pendingEpisodes)
	pendingApprovals += pendingEpisodes

	// Total revenue from subscriptions
	var totalRevenue float64
	h.db.Model(&models.Subscription{}).Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_users":       totalUsers,
		"active_creators":   activeCreators,
		"pending_approvals": pendingApprovals,
		"total_revenue":     totalRevenue,
	})
}

// AdminListUsers returns all users for the admin panel
func (h *AdminHandler) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse query params for filtering
	role := r.URL.Query().Get("role") // user, admin, creator

	type UserResponse struct {
		ID          string    `json:"id"`
		Phone       string    `json:"phone"`
		DisplayName string    `json:"display_name"`
		Email       *string   `json:"email"`
		Role        string    `json:"role"`
		CreatedAt   time.Time `json:"created_at"`
		IsCreator   bool      `json:"is_creator"`
	}

	var users []models.User
	query := h.db.Model(&models.User{})

	if role != "" {
		if role == "creator" {
			// Users who have a creator profile
			query = query.Joins("JOIN creator_profiles ON creator_profiles.user_id = users.id")
		} else {
			query = query.Where("role = ?", role)
		}
	}

	if err := query.Order("created_at DESC").Find(&users).Error; err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}

	// Get creator profile IDs for users
	creatorUserIDs := make(map[string]bool)
	var creatorProfiles []models.CreatorProfile
	h.db.Find(&creatorProfiles)
	for _, cp := range creatorProfiles {
		creatorUserIDs[cp.UserID] = true
	}

	// Build response
	response := make([]UserResponse, 0, len(users))
	for _, u := range users {
		displayName := u.Name
		if displayName == "" {
			displayName = u.Phone
		}

		userRole := u.Role
		if userRole == "" {
			userRole = "user"
		}

		response = append(response, UserResponse{
			ID:          u.ID,
			Phone:       u.Phone,
			DisplayName: displayName,
			Email:       nil, // Email not in current model
			Role:        userRole,
			CreatedAt:   u.CreatedAt,
			IsCreator:   creatorUserIDs[u.ID],
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AdminListAllSeries returns all series for admin content management
func (h *AdminHandler) AdminListAllSeries(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status") // draft, pending, published, rejected
	creatorID := r.URL.Query().Get("creator_id")

	type SeriesResponse struct {
		ID           string    `json:"id"`
		Title        string    `json:"title"`
		Synopsis     string    `json:"synopsis"`
		CreatorID    string    `json:"creator_id"`
		CreatorName  string    `json:"creator_name"`
		ThumbnailURL *string   `json:"thumbnail_url"`
		Status       string    `json:"status"`
		CreatedAt    time.Time `json:"created_at"`
		ViewCount    int64     `json:"view_count"`
		LikeCount    int64     `json:"like_count"`
	}

	var seriesList []models.Series
	query := h.db.Model(&models.Series{}).Preload("Creator")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if creatorID != "" {
		query = query.Where("creator_id = ?", creatorID)
	}

	if err := query.Order("created_at DESC").Find(&seriesList).Error; err != nil {
		http.Error(w, "Failed to fetch series", http.StatusInternalServerError)
		return
	}

	// Build response with computed fields
	response := make([]SeriesResponse, 0, len(seriesList))
	for _, s := range seriesList {
		creatorName := ""
		if s.Creator != nil {
			creatorName = s.Creator.DisplayName
		}

		// Get view count from series_views
		var viewCount int64
		h.db.Model(&models.SeriesView{}).Where("series_id = ?", s.ID).Select("COALESCE(SUM(view_count), 0)").Scan(&viewCount)

		// Get like count from episode_likes for all episodes in this series
		var likeCount int64
		h.db.Model(&models.EpisodeLike{}).
			Joins("JOIN episodes ON episodes.id = episode_likes.episode_id").
			Where("episodes.series_id = ?", s.ID).
			Count(&likeCount)

		response = append(response, SeriesResponse{
			ID:           s.ID,
			Title:        s.Title,
			Synopsis:     s.Synopsis,
			CreatorID:    s.CreatorID,
			CreatorName:  creatorName,
			ThumbnailURL: s.ThumbnailURL,
			Status:       s.Status,
			CreatedAt:    s.CreatedAt,
			ViewCount:    viewCount,
			LikeCount:    likeCount,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
