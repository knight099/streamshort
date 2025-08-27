package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"streamshort/models"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type SubscriptionHandler struct {
	db *gorm.DB
}

func NewSubscriptionHandler(db *gorm.DB) *SubscriptionHandler {
	return &SubscriptionHandler{db: db}
}

// GetUserSubscriptions returns all subscriptions for the authenticated user
func (h *SubscriptionHandler) GetUserSubscriptions(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Get query parameters for pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage <= 0 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	// Get subscriptions with pagination
	var subscriptions []models.Subscription
	var total int64

	// Count total subscriptions
	if err := h.db.Model(&models.Subscription{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		http.Error(w, "Failed to count subscriptions", http.StatusInternalServerError)
		return
	}

	// Get paginated subscriptions
	if err := h.db.Where("user_id = ?", userID).
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(perPage).
		Find(&subscriptions).Error; err != nil {
		http.Error(w, "Failed to fetch subscriptions", http.StatusInternalServerError)
		return
	}

	// Build response
	response := map[string]interface{}{
		"total":         total,
		"page":          page,
		"per_page":      perPage,
		"total_pages":   (total + int64(perPage) - 1) / int64(perPage),
		"subscriptions": subscriptions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUserSubscription returns a specific subscription for the authenticated user
func (h *SubscriptionHandler) GetUserSubscription(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Get subscription ID from URL
	vars := mux.Vars(r)
	subscriptionID := vars["id"]

	// Find subscription and verify ownership
	var subscription models.Subscription
	if err := h.db.Where("id = ? AND user_id = ?", subscriptionID, userID).
		Preload("User").
		First(&subscription).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Subscription not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscription)
}

// CancelUserSubscription cancels a user's subscription
func (h *SubscriptionHandler) CancelUserSubscription(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Get subscription ID from URL
	vars := mux.Vars(r)
	subscriptionID := vars["id"]

	// Find subscription and verify ownership
	var subscription models.Subscription
	if err := h.db.Where("id = ? AND user_id = ?", subscriptionID, userID).
		First(&subscription).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Subscription not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check if subscription can be cancelled
	if subscription.Status != "active" {
		http.Error(w, "Subscription is not active and cannot be cancelled", http.StatusBadRequest)
		return
	}

	// Update status to cancelled
	if err := h.db.Model(&subscription).Update("status", "cancelled").Error; err != nil {
		http.Error(w, "Failed to cancel subscription", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":        "Subscription cancelled successfully",
		"subscription_id": subscriptionID,
		"status":         "cancelled",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RenewUserSubscription renews a user's subscription
func (h *SubscriptionHandler) RenewUserSubscription(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Get subscription ID from URL
	vars := mux.Vars(r)
	subscriptionID := vars["id"]

	// Find subscription and verify ownership
	var subscription models.Subscription
	if err := h.db.Where("id = ? AND user_id = ?", subscriptionID, userID).
		First(&subscription).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Subscription not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check if subscription can be renewed
	if subscription.Status != "active" && subscription.Status != "expired" {
		http.Error(w, "Subscription cannot be renewed", http.StatusBadRequest)
		return
	}

	// Calculate new end date (extend from current end date or now)
	var newEndDate time.Time
	if subscription.Status == "active" {
		newEndDate = subscription.EndDate.AddDate(0, 0, 30) // Add 30 days
	} else {
		newEndDate = time.Now().AddDate(0, 0, 30) // Add 30 days from now
	}

	// Update subscription
	updates := map[string]interface{}{
		"status":    "active",
		"end_date":  newEndDate,
		"updated_at": time.Now(),
	}

	if err := h.db.Model(&subscription).Updates(updates).Error; err != nil {
		http.Error(w, "Failed to renew subscription", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":        "Subscription renewed successfully",
		"subscription_id": subscriptionID,
		"status":         "active",
		"new_end_date":   newEndDate,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetSubscriptionPlans returns available subscription plans
func (h *SubscriptionHandler) GetSubscriptionPlans(w http.ResponseWriter, r *http.Request) {
	var plans []models.SubscriptionPlan

	if err := h.db.Where("is_active = ?", true).Find(&plans).Error; err != nil {
		http.Error(w, "Failed to fetch subscription plans", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plans)
}

// CheckSubscriptionStatus checks if a user has access to specific content
func (h *SubscriptionHandler) CheckSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Get query parameters
	targetType := r.URL.Query().Get("target_type")
	targetID := r.URL.Query().Get("target_id")

	if targetType == "" || targetID == "" {
		http.Error(w, "Target type and target ID are required", http.StatusBadRequest)
		return
	}

	// Check subscription status
	var subscription models.Subscription
	err := h.db.Where("user_id = ? AND target_type = ? AND target_id = ? AND status = ?", 
		userID, targetType, targetID, "active").
		First(&subscription).Error

	hasAccess := false
	var subscriptionDetails *models.Subscription

	if err == nil && !subscription.IsExpired() {
		hasAccess = true
		subscriptionDetails = &subscription
	}

	response := map[string]interface{}{
		"has_access":           hasAccess,
		"target_type":          targetType,
		"target_id":            targetID,
		"subscription_details": subscriptionDetails,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
