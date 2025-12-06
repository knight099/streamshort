package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"streamshort/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentHandler struct {
	db *gorm.DB
}

func NewPaymentHandler(db *gorm.DB) *PaymentHandler {
	return &PaymentHandler{db: db}
}

// Request/Response structs matching OpenAPI schema
type CreateSubscriptionRequest struct {
	TargetType string `json:"target_type"`
	PlanID     string `json:"plan_id"`
	AutoRenew  bool   `json:"auto_renew"`
}

type CreateSubscriptionResponse struct {
	SubscriptionID string    `json:"subscription_id"`
	Status         string    `json:"status"`
	PlanID         string    `json:"plan_id"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	NextBilling    time.Time `json:"next_billing"`
	CheckoutURL    string    `json:"checkout_url"`
}

type WebhookRequest struct {
	EventType string                 `json:"event_type"`
	Data      map[string]interface{} `json:"data"`
	Signature string                 `json:"signature"`
}

type WebhookResponse struct {
	Status string `json:"status"`
}

// CreateSubscription handles subscription creation
func (h *PaymentHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	var req CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Normalize inputs for all-access plans: allow missing or non-standard target
	isAllAccessPlan := req.PlanID == "all_access_30d" || req.PlanID == "all_access_365d"
	if isAllAccessPlan {
		// For all-access, we don't gate on target. Treat as global subscription.
		req.TargetType = "global"
	}

	// Validate required fields
	if req.PlanID == "" {
		http.Error(w, "Plan ID is required", http.StatusBadRequest)
		return
	}
	// For our current plans (all-access), target is not required

	// Validate target type
	if req.TargetType != "" { // allow empty when all-access
		if req.TargetType != "series" && req.TargetType != "creator" && req.TargetType != "global" {
			http.Error(w, "Target type must be 'series', 'creator', or 'global'", http.StatusBadRequest)
			return
		}
	}

	// Check if user already has an active subscription
	if isAllAccessPlan {
		// Prevent multiple active all-access plans
		var existingAllAccess models.Subscription
		if err := h.db.Where("user_id = ? AND status = ? AND plan_id IN ?",
			userID, "active", []string{"all_access_30d", "all_access_365d"}).
			First(&existingAllAccess).Error; err == nil {
			http.Error(w, "User already has an active all-access subscription", http.StatusConflict)
			return
		}
	} else {
		var existingSubscription models.Subscription
		if err := h.db.Where("user_id = ? AND target_type = ? AND status = ?",
			userID, req.TargetType, "active").First(&existingSubscription).Error; err == nil {
			http.Error(w, "User already has an active subscription to this content", http.StatusConflict)
			return
		}
	}

	// Get plan details from subscription_plans table
	var plan models.SubscriptionPlan
	if err := h.db.Where("id = ? AND is_active = ?", req.PlanID, true).First(&plan).Error; err != nil {
		http.Error(w, "Invalid or inactive plan_id", http.StatusBadRequest)
		return
	}
	planAmount := plan.Amount
	planDuration := plan.Duration

	// Create subscription record
	subscriptionID := uuid.New().String()
	now := time.Now()
	endDate := now.AddDate(0, 0, planDuration)

	subscription := models.Subscription{
		ID:         subscriptionID,
		UserID:     userID,
		TargetType: req.TargetType,
		Status:     "active",
		StartDate:  now,
		EndDate:    endDate,
		AutoRenew:  req.AutoRenew,
		PlanID:     req.PlanID,
		Amount:     planAmount,
		Currency:   "INR",
	}

	// Save to database
	if err := h.db.Create(&subscription).Error; err != nil {
		http.Error(w, "Failed to create subscription", http.StatusInternalServerError)
		return
	}

	// In real implementation, this would integrate with Razorpay
	// For now, we'll return a mock response
	response := CreateSubscriptionResponse{
		SubscriptionID: subscriptionID,
		Status:         "active",
		PlanID:         req.PlanID,
		StartDate:      now,
		EndDate:        endDate,
		NextBilling:    endDate,
		CheckoutURL:    fmt.Sprintf("https://checkout.razorpay.com/v1/checkout.js?subscription_id=%s", subscriptionID),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Webhook handles payment webhooks from payment providers
func (h *PaymentHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	var req WebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate webhook signature (in real implementation)
	if req.Signature == "" {
		http.Error(w, "Missing signature", http.StatusUnauthorized)
		return
	}

	// Process webhook based on event type
	switch req.EventType {
	case "subscription.activated":
		// Handle subscription activation
		h.handleSubscriptionActivated(req.Data)
	case "subscription.updated":
		// Handle subscription updates
		h.handleSubscriptionUpdated(req.Data)
	case "subscription.cancelled":
		// Handle subscription cancellation
		h.handleSubscriptionCancelled(req.Data)
	case "payment.succeeded":
		// Handle successful payment
		h.handlePaymentSucceeded(req.Data)
	case "payment.failed":
		// Handle failed payment
		h.handlePaymentFailed(req.Data)
	default:
		// Unknown event type
		break
	}

	response := WebhookResponse{
		Status: "processed",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleSubscriptionActivated processes subscription activation webhook
func (h *PaymentHandler) handleSubscriptionActivated(data map[string]interface{}) {
	// Extract subscription data from webhook
	if subscriptionID, ok := data["subscription_id"].(string); ok {
		// Update subscription status in database
		h.db.Model(&models.Subscription{}).
			Where("id = ?", subscriptionID).
			Update("status", "active")
	}
}

// handleSubscriptionUpdated processes subscription update webhook
func (h *PaymentHandler) handleSubscriptionUpdated(data map[string]interface{}) {
	// Handle subscription updates
}

// handleSubscriptionCancelled processes subscription cancellation webhook
func (h *PaymentHandler) handleSubscriptionCancelled(data map[string]interface{}) {
	if subscriptionID, ok := data["subscription_id"].(string); ok {
		// Update subscription status in database
		h.db.Model(&models.Subscription{}).
			Where("id = ?", subscriptionID).
			Update("status", "cancelled")
	}
}

// handlePaymentSucceeded processes successful payment webhook
func (h *PaymentHandler) handlePaymentSucceeded(data map[string]interface{}) {
	// Handle successful payment
}

// handlePaymentFailed processes failed payment webhook
func (h *PaymentHandler) handlePaymentFailed(data map[string]interface{}) {
	// Handle failed payment
}

// CheckUserSubscription checks if a user has an active subscription to specific content
func (h *PaymentHandler) CheckUserSubscription(userID, targetType, targetID string) (*models.Subscription, error) {
	var subscription models.Subscription

	err := h.db.Where("user_id = ? AND target_type = ? AND target_id = ? AND status = ?",
		userID, targetType, targetID, "active").
		First(&subscription).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No subscription found
		}
		return nil, err // Database error
	}

	// Check if subscription is still active (not expired)
	if subscription.IsExpired() {
		// Update status to expired
		h.db.Model(&subscription).Update("status", "expired")
		return nil, nil
	}

	return &subscription, nil
}

// GetUserSubscriptions returns all subscriptions for a user
func (h *PaymentHandler) GetUserSubscriptions(userID string) ([]models.Subscription, error) {
	var subscriptions []models.Subscription

	err := h.db.Where("user_id = ?", userID).
		Preload("User").
		Order("created_at DESC").
		Find(&subscriptions).Error

	return subscriptions, err
}

// CancelSubscription cancels a user's subscription
func (h *PaymentHandler) CancelSubscription(userID, subscriptionID string) error {
	var subscription models.Subscription

	// Find subscription and verify ownership
	err := h.db.Where("id = ? AND user_id = ?", subscriptionID, userID).
		First(&subscription).Error

	if err != nil {
		return err
	}

	// Update status to cancelled
	return h.db.Model(&subscription).Update("status", "cancelled").Error
}
