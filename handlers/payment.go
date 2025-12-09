package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"streamshort/models"
	"streamshort/services"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentHandler struct {
	db            *gorm.DB
	razorpayClient *services.RazorpayClient
}

func NewPaymentHandler(db *gorm.DB, razorpayClient *services.RazorpayClient) *PaymentHandler {
	return &PaymentHandler{db: db, razorpayClient: razorpayClient}
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
		// Check for existing active all-access plans
		var existingAllAccess models.Subscription
		if err := h.db.Where("user_id = ? AND status = ? AND plan_id IN ?",
			userID, "active", []string{"all_access_30d", "all_access_365d"}).
			First(&existingAllAccess).Error; err == nil {

			// If plan is different (e.g. upgrading from monthly to yearly), allow it
			// The old subscription will be cancelled when the new one is activated (or immediately in this mock)
			if existingAllAccess.PlanID == req.PlanID {
				http.Error(w, "User already has an active subscription to this plan", http.StatusConflict)
				return
			}

			// If plans are different, we'll proceed to create the new one.
			// In a real Razorpay flow, you'd create a subscription and wait for webhook to swap them.
			// Since we are mocking "auto-active" here, let's cancel the old one to avoid duplicates.
			h.db.Model(&existingAllAccess).Update("status", "cancelled")
		}
	} else {
		var existingSubscription models.Subscription
		if err := h.db.Where("user_id = ? AND target_type = ? AND status = ?",
			userID, req.TargetType, "active").First(&existingSubscription).Error; err == nil {

			// Similar logic for specific content subscriptions if needed
			if existingSubscription.PlanID == req.PlanID {
				http.Error(w, "User already has an active subscription to this content", http.StatusConflict)
				return
			}
			// Cancel old one if upgrading plan
			h.db.Model(&existingSubscription).Update("status", "cancelled")
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

	// Check if Razorpay is configured
	if h.razorpayClient == nil || h.razorpayClient.KeyID == "" {
		http.Error(w, "Payment gateway not configured", http.StatusInternalServerError)
		return
	}

	// First, create subscription in Razorpay
	// Note: You need to create a Razorpay Plan first (plan_id should be Razorpay plan ID like "plan_xxxxx")
	// For now, we'll assume plan_id in DB maps to Razorpay plan_id
	// If your plan_id is not a Razorpay plan_id, you'll need to map it or create plans in Razorpay first
	
	totalCount := 1 // For one-time payment, set to 1. For recurring, set appropriate count
	if req.AutoRenew {
		// For monthly: 12, yearly: 1, etc. Adjust based on your plan
		if planDuration == 30 {
			totalCount = 12 // Monthly plan, 12 months
		} else if planDuration == 365 {
			totalCount = 1 // Yearly plan, 1 payment
		}
	}

	razorpayResp, err := h.razorpayClient.CreateSubscription(req.PlanID, 1, totalCount, 0)
	if err != nil {
		log.Printf("[payment] Failed to create Razorpay subscription: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create subscription: %v", err), http.StatusInternalServerError)
		return
	}

	razorpaySubID, ok := razorpayResp["id"].(string)
	if !ok {
		log.Printf("[payment] Invalid Razorpay response: %v", razorpayResp)
		http.Error(w, "Invalid response from payment gateway", http.StatusInternalServerError)
		return
	}

	// Extract dates from Razorpay response
	now := time.Now()
	var startDate, endDate time.Time
	var status string = "pending" // Start as pending, activate on webhook

	if startAt, ok := razorpayResp["start_at"].(float64); ok && startAt > 0 {
		startDate = time.Unix(int64(startAt), 0)
	} else {
		startDate = now
	}

	if endAt, ok := razorpayResp["end_at"].(float64); ok && endAt > 0 {
		endDate = time.Unix(int64(endAt), 0)
	} else {
		endDate = startDate.AddDate(0, 0, planDuration)
	}

	if subStatus, ok := razorpayResp["status"].(string); ok {
		status = subStatus
	}

	// Create subscription record in our DB
	subscriptionID := uuid.New().String()
	subscription := models.Subscription{
		ID:                     subscriptionID,
		UserID:                 userID,
		TargetType:             req.TargetType,
		RazorpaySubscriptionID: &razorpaySubID,
		Status:                 status,
		StartDate:              startDate,
		EndDate:                endDate,
		AutoRenew:              req.AutoRenew,
		PlanID:                 req.PlanID,
		Amount:                 planAmount,
		Currency:               "INR",
	}

	// Save to database
	if err := h.db.Create(&subscription).Error; err != nil {
		log.Printf("[payment] Failed to save subscription to DB: %v", err)
		http.Error(w, "Failed to create subscription", http.StatusInternalServerError)
		return
	}

	// Create payment link for checkout
	paymentLinkResp, err := h.razorpayClient.CreatePaymentLink(
		razorpaySubID,
		planAmount,
		"INR",
		fmt.Sprintf("Subscription for %s", req.PlanID),
	)
	if err != nil {
		log.Printf("[payment] Failed to create payment link: %v", err)
		// Continue anyway, we can use the subscription ID for checkout
	}

	var checkoutURL string
	if paymentLinkResp != nil {
		if shortURL, ok := paymentLinkResp["short_url"].(string); ok {
			checkoutURL = shortURL
		} else if url, ok := paymentLinkResp["url"].(string); ok {
			checkoutURL = url
		}
	}

	// Fallback: Use Razorpay checkout URL with subscription ID
	if checkoutURL == "" {
		checkoutURL = fmt.Sprintf("https://checkout.razorpay.com/v1/subscription/%s", razorpaySubID)
	}

	response := CreateSubscriptionResponse{
		SubscriptionID: razorpaySubID, // Return Razorpay subscription ID, not our UUID
		Status:         status,
		PlanID:         req.PlanID,
		StartDate:      startDate,
		EndDate:        endDate,
		NextBilling:    endDate,
		CheckoutURL:    checkoutURL,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Webhook handles payment webhooks from Razorpay
func (h *PaymentHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	// Razorpay sends webhook as JSON with event and payload fields
	var webhookPayload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&webhookPayload); err != nil {
		log.Printf("[webhook] Failed to decode webhook payload: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Extract event name (Razorpay format: "subscription.charged", "subscription.activated", etc.)
	event, ok := webhookPayload["event"].(string)
	if !ok {
		log.Printf("[webhook] Missing event field in webhook payload")
		http.Error(w, "Missing event field", http.StatusBadRequest)
		return
	}

	// Extract payload (contains the subscription/payment data)
	payload, ok := webhookPayload["payload"].(map[string]interface{})
	if !ok {
		// Sometimes Razorpay sends data directly, try that
		payload = webhookPayload
	}

	log.Printf("[webhook] Received event: %s", event)

	// Process webhook based on event type
	switch event {
	case "subscription.charged":
		// Payment successful - activate subscription and record revenue
		h.handleSubscriptionCharged(payload)
	case "subscription.activated":
		// Subscription activated
		h.handleSubscriptionActivated(payload)
	case "subscription.updated":
		// Subscription updated
		h.handleSubscriptionUpdated(payload)
	case "subscription.cancelled":
		// Subscription cancelled
		h.handleSubscriptionCancelled(payload)
	case "subscription.halted":
		// Payment failed, subscription halted
		h.handleSubscriptionHalted(payload)
	case "subscription.completed":
		// Subscription completed (all cycles done)
		h.handleSubscriptionCompleted(payload)
	case "payment.failed":
		// Payment failed
		h.handlePaymentFailed(payload)
	default:
		log.Printf("[webhook] Unknown event type: %s", event)
	}

	response := WebhookResponse{
		Status: "processed",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleSubscriptionCharged processes subscription.charged webhook (payment successful)
func (h *PaymentHandler) handleSubscriptionCharged(data map[string]interface{}) {
	// Razorpay sends subscription object in payload.subscription.entity
	subscriptionEntity, ok := data["subscription"].(map[string]interface{})
	if !ok {
		if entity, ok := data["entity"].(map[string]interface{}); ok {
			subscriptionEntity = entity
		} else {
			log.Printf("[webhook] subscription.charged: missing subscription entity")
			return
		}
	}

	razorpaySubID, ok := subscriptionEntity["id"].(string)
	if !ok {
		log.Printf("[webhook] subscription.charged: missing subscription id")
		return
	}

	// Find subscription by Razorpay subscription ID
	var subscription models.Subscription
	if err := h.db.Where("razorpay_subscription_id = ?", razorpaySubID).First(&subscription).Error; err != nil {
		log.Printf("[webhook] subscription.charged: subscription not found: %v", err)
		return
	}

	// Extract payment amount
	var amount float64
	if paymentEntity, ok := data["payment"].(map[string]interface{}); ok {
		if amt, ok := paymentEntity["amount"].(float64); ok {
			amount = amt / 100.0 // Razorpay sends amount in paise
		}
	}

	// Update subscription status to active
	h.db.Model(&subscription).Update("status", "active")

	// Record Revenue for Monthly Payout Calculation
	now := time.Now()
	monthDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	platformFee := amount * 0.30
	distributable := amount * 0.70

	rev := models.SubscriptionRevenue{
		UserID:         subscription.UserID,
		SubscriptionID: subscription.ID,
		MonthDate:      monthDate,
		Amount:         amount,
		Distributable:  distributable,
		PlatformFee:    platformFee,
		IsDistributed:  false,
	}
	if err := h.db.Create(&rev).Error; err != nil {
		log.Printf("[webhook] subscription.charged: failed to create revenue record: %v", err)
	}

	log.Printf("[webhook] subscription.charged: activated subscription %s, recorded revenue %.2f", razorpaySubID, amount)
}

// handleSubscriptionActivated processes subscription activation webhook
func (h *PaymentHandler) handleSubscriptionActivated(data map[string]interface{}) {
	subscriptionEntity, ok := data["subscription"].(map[string]interface{})
	if !ok {
		if entity, ok := data["entity"].(map[string]interface{}); ok {
			subscriptionEntity = entity
		} else {
			return
		}
	}

	razorpaySubID, ok := subscriptionEntity["id"].(string)
	if !ok {
		return
	}

	h.db.Model(&models.Subscription{}).
		Where("razorpay_subscription_id = ?", razorpaySubID).
		Update("status", "active")
	
	log.Printf("[webhook] subscription.activated: activated subscription %s", razorpaySubID)
}

// handleSubscriptionUpdated processes subscription update webhook
func (h *PaymentHandler) handleSubscriptionUpdated(data map[string]interface{}) {
	// Handle subscription updates (e.g., plan changes, end date changes)
	subscriptionEntity, ok := data["subscription"].(map[string]interface{})
	if !ok {
		if entity, ok := data["entity"].(map[string]interface{}); ok {
			subscriptionEntity = entity
		} else {
			return
		}
	}

	razorpaySubID, ok := subscriptionEntity["id"].(string)
	if !ok {
		return
	}

	// Update end date if provided
	if endAt, ok := subscriptionEntity["end_at"].(float64); ok && endAt > 0 {
		endDate := time.Unix(int64(endAt), 0)
		h.db.Model(&models.Subscription{}).
			Where("razorpay_subscription_id = ?", razorpaySubID).
			Update("end_date", endDate)
	}

	log.Printf("[webhook] subscription.updated: updated subscription %s", razorpaySubID)
}

// handleSubscriptionCancelled processes subscription cancellation webhook
func (h *PaymentHandler) handleSubscriptionCancelled(data map[string]interface{}) {
	subscriptionEntity, ok := data["subscription"].(map[string]interface{})
	if !ok {
		if entity, ok := data["entity"].(map[string]interface{}); ok {
			subscriptionEntity = entity
		} else {
			return
		}
	}

	razorpaySubID, ok := subscriptionEntity["id"].(string)
	if !ok {
		return
	}

	h.db.Model(&models.Subscription{}).
		Where("razorpay_subscription_id = ?", razorpaySubID).
		Update("status", "cancelled")
	
	log.Printf("[webhook] subscription.cancelled: cancelled subscription %s", razorpaySubID)
}

// handleSubscriptionHalted processes subscription halted webhook (payment failed)
func (h *PaymentHandler) handleSubscriptionHalted(data map[string]interface{}) {
	subscriptionEntity, ok := data["subscription"].(map[string]interface{})
	if !ok {
		if entity, ok := data["entity"].(map[string]interface{}); ok {
			subscriptionEntity = entity
		} else {
			return
		}
	}

	razorpaySubID, ok := subscriptionEntity["id"].(string)
	if !ok {
		return
	}

	// You might want to add a "halted" or "past_due" status
	h.db.Model(&models.Subscription{}).
		Where("razorpay_subscription_id = ?", razorpaySubID).
		Update("status", "cancelled") // Or create a "halted" status
	
	log.Printf("[webhook] subscription.halted: halted subscription %s", razorpaySubID)
}

// handleSubscriptionCompleted processes subscription completed webhook
func (h *PaymentHandler) handleSubscriptionCompleted(data map[string]interface{}) {
	subscriptionEntity, ok := data["subscription"].(map[string]interface{})
	if !ok {
		if entity, ok := data["entity"].(map[string]interface{}); ok {
			subscriptionEntity = entity
		} else {
			return
		}
	}

	razorpaySubID, ok := subscriptionEntity["id"].(string)
	if !ok {
		return
	}

	h.db.Model(&models.Subscription{}).
		Where("razorpay_subscription_id = ?", razorpaySubID).
		Update("status", "expired")
	
	log.Printf("[webhook] subscription.completed: completed subscription %s", razorpaySubID)
}

// handlePaymentFailed processes failed payment webhook
func (h *PaymentHandler) handlePaymentFailed(data map[string]interface{}) {
	log.Printf("[webhook] payment.failed: %v", data)
	// Handle failed payment - might want to notify user or update subscription status
}

// CheckUserSubscription checks if a user has an active subscription to specific content
func (h *PaymentHandler) CheckUserSubscription(userID, targetType, targetID string) (*models.Subscription, error) {
	var subscription models.Subscription

	err := h.db.Where("user_id = ? AND target_type = ? AND status = ?",
		userID, targetType, "active").
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
