package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type AdminHandler struct {
	// In a real implementation, you'd have admin service clients here
}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
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
