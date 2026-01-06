package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"episodd/models"
	"episodd/services"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentHandler struct {
	db             *gorm.DB
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
// CreateSubscription godoc
// @Summary Create Subscription
// @Description Create a new subscription for a plan.
// @Tags payment
// @Accept  json
// @Produce  json
// @Param   request  body     CreateSubscriptionRequest  true  "Create Request"
// @Success 201 {object} CreateSubscriptionResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 409 {string} string "Conflict"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/payments/create-subscription [post]
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

	// All subscriptions are global - set target_type to global
	req.TargetType = "global"

	// Validate required fields
	if req.PlanID == "" {
		http.Error(w, "Plan ID is required", http.StatusBadRequest)
		return
	}

	// Check if user already has an active subscription
	var existingSubscription models.Subscription
	if err := h.db.Where("user_id = ? AND status = ? AND target_type = ?",
		userID, "active", "global").First(&existingSubscription).Error; err == nil {
		// If same plan, reject
		if existingSubscription.PlanID == req.PlanID {
			http.Error(w, "User already has an active subscription to this plan", http.StatusConflict)
			return
		}
		// If different plan, cancel the old one (upgrade/downgrade)
		h.db.Model(&existingSubscription).Update("status", "cancelled")
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

	// Get Razorpay plan ID (must be set in database)
	razorpayPlanID := plan.RazorpayPlanID
	if razorpayPlanID == "" {
		log.Printf("[payment] Plan %s missing razorpay_plan_id", req.PlanID)
		http.Error(w, fmt.Sprintf("Plan %s is not configured with Razorpay. Please contact support.", req.PlanID), http.StatusBadRequest)
		return
	}

	totalCount := 1 // For one-time payment, set to 1. For recurring, set appropriate count
	if req.AutoRenew {
		// For monthly: 12, yearly: 1, etc. Adjust based on your plan
		// For monthly: 12, yearly: 1, etc. Adjust based on your plan
		switch planDuration {
		case 30:
			totalCount = 12 // Monthly plan, 12 months
		case 365:
			totalCount = 1 // Yearly plan, 1 payment
		}
	}

	razorpayResp, err := h.razorpayClient.CreateSubscription(razorpayPlanID, 1, totalCount, 0)
	if err != nil {
		log.Printf("[payment] Failed to create Razorpay subscription for plan %s (Razorpay ID: %s): %v", req.PlanID, razorpayPlanID, err)
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
		// Map Razorpay statuses to our allowed set
		switch subStatus {
		case "created", "initiated":
			status = "pending"
		case "authenticated", "active":
			status = "active"
		case "halted", "paused":
			status = "halted"
		case "cancelled":
			status = "cancelled"
		case "expired":
			status = "expired"
		default:
			// Fallback to pending for any unexpected status to avoid DB constraint violation
			status = "pending"
		}
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
		log.Printf("[payment] Failed to save subscription to DB (UserID: %s, RazorpaySubID: %s): %v", userID, razorpaySubID, err)
		http.Error(w, fmt.Sprintf("Failed to save subscription: %v", err), http.StatusInternalServerError)
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
// Webhook godoc
// @Summary Payment Webhook
// @Description Handle payment webhooks from Razorpay.
// @Tags payment
// @Accept  json
// @Produce  json
// @Param   request  body     WebhookRequest  true  "Webhook Payload"
// @Success 200 {object} WebhookResponse
// @Failure 400 {string} string "Bad Request"
// @Router /payments/webhook [post]
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
	var subscriptionEntity map[string]interface{}
	if subWrapper, ok := data["subscription"].(map[string]interface{}); ok {
		if ent, ok := subWrapper["entity"].(map[string]interface{}); ok {
			subscriptionEntity = ent
		}
	}
	// Fallback if structure differs
	if subscriptionEntity == nil {
		if ent, ok := data["entity"].(map[string]interface{}); ok {
			subscriptionEntity = ent
		}
	}

	if subscriptionEntity == nil {
		log.Printf("[webhook] subscription.charged: missing subscription entity")
		return
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
	if paymentWrapper, ok := data["payment"].(map[string]interface{}); ok {
		if paymentEntity, ok := paymentWrapper["entity"].(map[string]interface{}); ok {
			if amt, ok := paymentEntity["amount"].(float64); ok {
				amount = amt / 100.0 // Razorpay sends amount in paise
			}
		}
	}

	// Update subscription status to active
	h.db.Model(&subscription).Update("status", "active")

	// Record Revenue for Monthly Payout Calculation (legacy table)
	now := time.Now()
	monthDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	platformFee := amount * models.PlatformFeePercent
	distributable := amount * models.CreatorSharePercent

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

	// Record creator earnings for the new payment system
	// For global subscriptions, we need to distribute based on watch time (handled by payout calculation)
	// For series-specific subscriptions, record earnings directly
	if subscription.TargetType == "series" {
		// Find series and creator
		// Note: Currently subscriptions don't have a direct series_id field
		// This would need to be added or we'd need to track via metadata
		log.Printf("[webhook] subscription.charged: series-specific subscription earning recording not yet implemented")
	}
	// For global subscriptions, earnings are calculated monthly based on watch time
	// via the CalculateMonthlyPayouts function in payout_calculation.go

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
