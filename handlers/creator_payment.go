package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"streamshort/models"
	"streamshort/services"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type CreatorPaymentHandler struct {
	db             *gorm.DB
	razorpayClient *services.RazorpayClient
}

func NewCreatorPaymentHandler(db *gorm.DB, razorpayClient *services.RazorpayClient) *CreatorPaymentHandler {
	return &CreatorPaymentHandler{db: db, razorpayClient: razorpayClient}
}

// GetCreatorEarnings returns earnings summary for a creator
// GET /api/creators/{creator_id}/earnings
// GetCreatorEarnings godoc
// @Summary Get Creator Earnings
// @Description Get earnings summary for a creator.
// @Tags creator-payment
// @Produce  json
// @Param   creator_id  path     string  true  "Creator ID"
// @Success 200 {object} models.EarningsSummary
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/creators/{creator_id}/earnings [get]
func (h *CreatorPaymentHandler) GetCreatorEarnings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creatorID := vars["creator_id"]

	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Verify the creator belongs to this user
	var creator models.CreatorProfile
	if err := h.db.Where("id = ? AND user_id = ?", creatorID, userID).First(&creator).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Creator not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Calculate earnings summary
	summary, err := h.calculateEarningsSummary(creatorID)
	if err != nil {
		log.Printf("Error calculating earnings summary: %v", err)
		http.Error(w, "Failed to calculate earnings", http.StatusInternalServerError)
		return
	}

	// Add creator details to the summary
	summary.CreatorDetails = &models.CreatorDetails{
		ID:            creator.ID,
		DisplayName:   creator.DisplayName,
		Bio:           creator.Bio,
		KYCStatus:     creator.KYCStatus,
		FollowerCount: creator.FollowerCount,
		Rating:        creator.Rating,
	}

	// Add payout details if available
	var payoutDetails models.PayoutDetails
	if err := h.db.Where("creator_id = ?", creatorID).First(&payoutDetails).Error; err == nil {
		summary.PayoutDetails = &models.PayoutDetailsInfo{
			AccountHolderName: payoutDetails.AccountHolder,
			AccountNumber:     payoutDetails.AccountNumber,
			IFSCCode:          payoutDetails.IFSCCode,
			BankName:          payoutDetails.BankName,
			AccountType:       payoutDetails.AccountType,
			UPIID:             payoutDetails.UPIID,
			Verified:          payoutDetails.Verified,
			UpdatedAt:         payoutDetails.UpdatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// calculateEarningsSummary calculates the complete earnings summary for a creator
func (h *CreatorPaymentHandler) calculateEarningsSummary(creatorID string) (*models.EarningsSummary, error) {
	summary := &models.EarningsSummary{
		CreatorID:        creatorID,
		MinimumThreshold: models.MinimumPayoutThreshold,
		Currency:         "INR",
	}

	// Get non-subscription earnings from creator_earnings (one_time, ad_revenue)
	// Subscription earnings come from creator_payouts to avoid double-counting
	var totalEarnings float64
	if err := h.db.Model(&models.CreatorEarnings{}).
		Where("creator_id = ? AND status != 'cancelled' AND earnings_type != 'subscription'", creatorID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalEarnings).Error; err != nil {
		return nil, err
	}
	summary.TotalEarnings = totalEarnings

	// Get pending non-subscription earnings
	var pendingEarnings float64
	if err := h.db.Model(&models.CreatorEarnings{}).
		Where("creator_id = ? AND status = 'pending' AND earnings_type != 'subscription'", creatorID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&pendingEarnings).Error; err != nil {
		return nil, err
	}
	summary.PendingEarnings = pendingEarnings
	summary.AvailableForPayout = pendingEarnings

	// Get paid earnings (all types)
	var paidEarnings float64
	if err := h.db.Model(&models.CreatorEarnings{}).
		Where("creator_id = ? AND status = 'paid'", creatorID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&paidEarnings).Error; err != nil {
		return nil, err
	}
	summary.PaidEarnings = paidEarnings

	// Check if can request payout
	summary.CanRequestPayout = pendingEarnings >= models.MinimumPayoutThreshold

	// Get subscription revenue from creator_payouts (single source of truth)
	var subTotal float64
	if err := h.db.Raw(`
		SELECT COALESCE(SUM(total_earnings), 0)
		FROM creator_payouts
		WHERE creator_id = $1 AND status = 'pending'
	`, creatorID).Scan(&subTotal).Error; err != nil {
		return nil, err
	}

	// Add subscription earnings from creator_payouts to totals
	summary.TotalEarnings = summary.TotalEarnings + subTotal
	summary.PendingEarnings = summary.PendingEarnings + subTotal
	summary.AvailableForPayout = summary.AvailableForPayout + subTotal
	summary.CanRequestPayout = summary.AvailableForPayout >= models.MinimumPayoutThreshold

	// Count subscribers from monthly_viewer_stats for this creator
	var subCount int64
	h.db.Raw(`
		SELECT COUNT(DISTINCT user_id)
		FROM monthly_viewer_stats
		WHERE creator_id = $1 AND month_date = DATE_TRUNC('month', CURRENT_DATE)
	`, creatorID).Scan(&subCount)

	summary.Breakdown.SubscriptionRevenue = models.SubscriptionRevenueBreakdown{
		Total:               subTotal,
		ActiveSubscriptions: subCount,
		MonthlyRecurring:    subTotal, // Current month's earnings
	}

	// Get one-time purchase breakdown
	var otpTotal float64
	var otpCount int64
	if err := h.db.Model(&models.CreatorEarnings{}).
		Where("creator_id = ? AND earnings_type = 'one_time' AND status != 'cancelled'", creatorID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&otpTotal).Error; err != nil {
		return nil, err
	}
	if err := h.db.Model(&models.CreatorEarnings{}).
		Where("creator_id = ? AND earnings_type = 'one_time' AND status != 'cancelled'", creatorID).
		Count(&otpCount).Error; err != nil {
		return nil, err
	}
	summary.Breakdown.OneTimePurchases = models.OneTimePurchaseBreakdown{
		Total: otpTotal,
		Count: otpCount,
	}

	// Get ad revenue breakdown
	var adTotal float64
	if err := h.db.Model(&models.CreatorEarnings{}).
		Where("creator_id = ? AND earnings_type = 'ad_revenue' AND status != 'cancelled'", creatorID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&adTotal).Error; err != nil {
		return nil, err
	}
	summary.Breakdown.AdRevenue = models.AdRevenueBreakdown{
		Total: adTotal,
	}

	// Get last payout date
	var lastPayout models.Payout
	if err := h.db.Where("creator_id = ? AND status = 'completed'", creatorID).
		Order("completed_at DESC").
		First(&lastPayout).Error; err == nil && lastPayout.CompletedAt != nil {
		summary.LastPayoutDate = lastPayout.CompletedAt
	}

	// Calculate next payout date (1st of next month)
	now := time.Now()
	nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	summary.NextPayoutDate = &nextMonth

	return summary, nil
}

// RequestPayout handles payout request from a creator
// POST /api/creators/{creator_id}/payouts/request
// RequestPayout godoc
// @Summary Request Payout
// @Description Request a payout for available earnings.
// @Tags creator-payment
// @Accept  json
// @Produce  json
// @Param   creator_id  path     string              true  "Creator ID"
// @Param   request     body     models.PayoutRequest true  "Payout Request"
// @Success 201 {object} models.PayoutResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/creators/{creator_id}/payouts/request [post]
func (h *CreatorPaymentHandler) RequestPayout(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creatorID := vars["creator_id"]

	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Verify the creator belongs to this user
	var creator models.CreatorProfile
	if err := h.db.Where("id = ? AND user_id = ?", creatorID, userID).First(&creator).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Creator not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var req models.PayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default to bank_transfer if not specified
	if req.PayoutMethod == "" {
		req.PayoutMethod = "bank_transfer"
	}

	// Validate payout method
	if req.PayoutMethod != "bank_transfer" && req.PayoutMethod != "upi" {
		http.Error(w, "Invalid payout method. Must be 'bank_transfer' or 'upi'", http.StatusBadRequest)
		return
	}

	// Get pending earnings
	var pendingEarnings float64
	if err := h.db.Model(&models.CreatorEarnings{}).
		Where("creator_id = ? AND status = 'pending'", creatorID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&pendingEarnings).Error; err != nil {
		http.Error(w, "Failed to calculate pending earnings", http.StatusInternalServerError)
		return
	}

	// Check minimum threshold
	if pendingEarnings < models.MinimumPayoutThreshold {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INSUFFICIENT_BALANCE",
				"message": "Earnings below minimum payout threshold of ₹500",
				"details": map[string]interface{}{
					"current_balance":   pendingEarnings,
					"minimum_threshold": models.MinimumPayoutThreshold,
				},
			},
		})
		return
	}

	// Use requested amount or all pending earnings
	payoutAmount := req.Amount
	if payoutAmount <= 0 || payoutAmount > pendingEarnings {
		payoutAmount = pendingEarnings
	}

	// Get payout details
	var payoutDetails models.PayoutDetails
	if err := h.db.Where("creator_id = ?", creatorID).First(&payoutDetails).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Payout details not configured. Please update your bank account details first.", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to get payout details", http.StatusInternalServerError)
		return
	}

	// Create payout record
	payout := models.Payout{
		CreatorID:    creatorID,
		Amount:       payoutAmount,
		Status:       "pending",
		PayoutMethod: req.PayoutMethod,
		RequestedAt:  time.Now(),
	}

	tx := h.db.Begin()

	if err := tx.Create(&payout).Error; err != nil {
		tx.Rollback()
		http.Error(w, "Failed to create payout request", http.StatusInternalServerError)
		return
	}

	// Mark earnings as being processed (link to payout)
	if err := tx.Model(&models.CreatorEarnings{}).
		Where("creator_id = ? AND status = 'pending'", creatorID).
		Updates(map[string]interface{}{
			"payout_id":  payout.ID,
			"updated_at": time.Now(),
		}).Error; err != nil {
		tx.Rollback()
		http.Error(w, "Failed to link earnings to payout", http.StatusInternalServerError)
		return
	}

	// Initiate Razorpay payout if configured
	if h.razorpayClient != nil && h.razorpayClient.KeyID != "" {
		razorpayPayoutID, err := h.initiateRazorpayPayout(creatorID, payoutAmount, req.PayoutMethod, &payoutDetails)
		if err != nil {
			log.Printf("Warning: Razorpay payout initiation failed: %v", err)
			// Continue with payout record, will process manually
		} else {
			payout.RazorpayPayoutID = &razorpayPayoutID
			payout.Status = "processing"
			tx.Save(&payout)
		}
	}

	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Failed to commit payout request", http.StatusInternalServerError)
		return
	}

	// Calculate estimated completion (3-5 business days)
	estimatedCompletion := time.Now().AddDate(0, 0, 5)

	response := models.PayoutResponse{
		PayoutID:            payout.ID,
		CreatorID:           creatorID,
		Amount:              payoutAmount,
		Status:              payout.Status,
		RequestedAt:         payout.RequestedAt,
		EstimatedCompletion: estimatedCompletion,
		RazorpayPayoutID:    payout.RazorpayPayoutID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// initiateRazorpayPayout initiates a payout via Razorpay Payouts API
func (h *CreatorPaymentHandler) initiateRazorpayPayout(creatorID string, amount float64, method string, details *models.PayoutDetails) (string, error) {
	// Create or get fund account from Razorpay
	fundAccountID := ""
	if details.RazorpayFundAccountID != nil {
		fundAccountID = *details.RazorpayFundAccountID
	}

	if fundAccountID == "" {
		// Create contact and fund account in Razorpay
		contactID, err := h.razorpayClient.CreateContact(details.AccountHolder, creatorID)
		if err != nil {
			return "", err
		}

		if method == "bank_transfer" {
			fundAccountID, err = h.razorpayClient.CreateBankFundAccount(
				contactID,
				details.AccountHolder,
				details.AccountNumber,
				details.IFSCCode,
			)
		} else if method == "upi" && details.UPIID != nil {
			fundAccountID, err = h.razorpayClient.CreateUPIFundAccount(
				contactID,
				*details.UPIID,
			)
		}
		if err != nil {
			return "", err
		}

		// Save the fund account ID for future use
		h.db.Model(details).Updates(map[string]interface{}{
			"razorpay_contact_id":      contactID,
			"razorpay_fund_account_id": fundAccountID,
		})
	}

	// Create payout
	payoutID, err := h.razorpayClient.CreatePayout(
		fundAccountID,
		amount,
		"INR",
		method,
		"StreamShort Creator Payout",
	)
	if err != nil {
		return "", err
	}

	return payoutID, nil
}

// GetPayoutHistory returns payout history for a creator
// GET /api/creators/{creator_id}/payouts
// GetPayoutHistory godoc
// @Summary Get Payout History
// @Description Get history of payout requests.
// @Tags creator-payment
// @Produce  json
// @Param   creator_id  path     string  true  "Creator ID"
// @Param   page        query    int     false "Page"
// @Param   per_page    query    int     false "Per Page"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/creators/{creator_id}/payouts [get]
func (h *CreatorPaymentHandler) GetPayoutHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creatorID := vars["creator_id"]

	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Verify the creator belongs to this user
	var creator models.CreatorProfile
	if err := h.db.Where("id = ? AND user_id = ?", creatorID, userID).First(&creator).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Creator not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Get pagination parameters
	page := 1
	perPage := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if parsed, err := strconv.Atoi(pp); err == nil && parsed > 0 && parsed <= 100 {
			perPage = parsed
		}
	}

	// Get total count
	var total int64
	h.db.Model(&models.Payout{}).Where("creator_id = ?", creatorID).Count(&total)

	// Get payouts with pagination
	var payouts []models.Payout
	offset := (page - 1) * perPage
	if err := h.db.Where("creator_id = ?", creatorID).
		Order("requested_at DESC").
		Offset(offset).
		Limit(perPage).
		Find(&payouts).Error; err != nil {
		http.Error(w, "Failed to get payouts", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	items := make([]models.PayoutHistoryItem, len(payouts))
	for i, p := range payouts {
		items[i] = models.PayoutHistoryItem{
			PayoutID:             p.ID,
			Amount:               p.Amount,
			Status:               p.Status,
			RequestedAt:          p.RequestedAt,
			CompletedAt:          p.CompletedAt,
			RazorpayPayoutID:     p.RazorpayPayoutID,
			TransactionReference: p.TransactionReference,
		}
	}

	response := map[string]interface{}{
		"payouts":  items,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPayoutDetails returns payout/bank account details for a creator
// GET /api/creators/{creator_id}/payout-details
// GetPayoutDetails godoc
// @Summary Get Payout Details
// @Description Get stored bank account details for payouts.
// @Tags creator-payment
// @Produce  json
// @Param   creator_id  path     string  true  "Creator ID"
// @Success 200 {object} models.PayoutDetailsResponse
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/creators/{creator_id}/payout-details [get]
func (h *CreatorPaymentHandler) GetPayoutDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creatorID := vars["creator_id"]

	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Verify the creator belongs to this user
	var creator models.CreatorProfile
	if err := h.db.Where("id = ? AND user_id = ?", creatorID, userID).First(&creator).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Creator not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Get payout details
	var payoutDetails models.PayoutDetails
	if err := h.db.Where("creator_id = ?", creatorID).First(&payoutDetails).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return empty payout details with creator info
			response := models.PayoutDetailsResponse{
				CreatorID:     creatorID,
				PayoutDetails: models.PayoutDetailsInfo{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	response := models.PayoutDetailsResponse{
		CreatorID: creatorID,
		PayoutDetails: models.PayoutDetailsInfo{
			AccountHolderName: payoutDetails.AccountHolder,
			AccountNumber:     payoutDetails.AccountNumber,
			IFSCCode:          payoutDetails.IFSCCode,
			BankName:          payoutDetails.BankName,
			AccountType:       payoutDetails.AccountType,
			UPIID:             payoutDetails.UPIID,
			Verified:          payoutDetails.Verified,
			UpdatedAt:         payoutDetails.UpdatedAt,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdatePayoutDetails updates payout/bank account details for a creator
// PUT /api/creators/{creator_id}/payout-details
// UpdatePayoutDetails godoc
// @Summary Update Payout Details
// @Description Update bank account details for payouts.
// @Tags creator-payment
// @Accept  json
// @Produce  json
// @Param   creator_id  path     string                     true  "Creator ID"
// @Param   request     body     models.PayoutDetailsRequest true  "Payout Details"
// @Success 200 {object} models.PayoutDetailsResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/creators/{creator_id}/payout-details [put]
func (h *CreatorPaymentHandler) UpdatePayoutDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creatorID := vars["creator_id"]

	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Verify the creator belongs to this user
	var creator models.CreatorProfile
	if err := h.db.Where("id = ? AND user_id = ?", creatorID, userID).First(&creator).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Creator not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var req models.PayoutDetailsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.AccountHolderName == "" || req.AccountNumber == "" || req.IFSCCode == "" || req.BankName == "" {
		http.Error(w, "Account holder name, account number, IFSC code, and bank name are required", http.StatusBadRequest)
		return
	}

	// Validate account type
	if req.AccountType == "" {
		req.AccountType = "savings"
	}
	if req.AccountType != "savings" && req.AccountType != "current" {
		http.Error(w, "Account type must be 'savings' or 'current'", http.StatusBadRequest)
		return
	}

	// Check if payout details already exist
	var existing models.PayoutDetails
	err := h.db.Where("creator_id = ?", creatorID).First(&existing).Error

	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		// Create new payout details
		details := models.PayoutDetails{
			CreatorID:     creatorID,
			AccountHolder: req.AccountHolderName,
			AccountNumber: req.AccountNumber,
			IFSCCode:      req.IFSCCode,
			BankName:      req.BankName,
			AccountType:   req.AccountType,
			UPIID:         req.UPIID,
			Verified:      false, // Will be verified via Razorpay
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if err := h.db.Create(&details).Error; err != nil {
			http.Error(w, "Failed to create payout details", http.StatusInternalServerError)
			return
		}

		// Clear Razorpay fund account since details changed
		existing = details
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	} else {
		// Update existing payout details
		updates := map[string]interface{}{
			"account_holder": req.AccountHolderName,
			"account_number": req.AccountNumber,
			"ifsc_code":      req.IFSCCode,
			"bank_name":      req.BankName,
			"account_type":   req.AccountType,
			"upi_id":         req.UPIID,
			"verified":       false, // Reset verification when details change
			"updated_at":     now,
			// Clear Razorpay IDs since account changed
			"razorpay_contact_id":      nil,
			"razorpay_fund_account_id": nil,
		}

		if err := h.db.Model(&existing).Updates(updates).Error; err != nil {
			http.Error(w, "Failed to update payout details", http.StatusInternalServerError)
			return
		}

		// Reload to get updated values
		h.db.Where("creator_id = ?", creatorID).First(&existing)
	}

	response := models.PayoutDetailsResponse{
		CreatorID: creatorID,
		PayoutDetails: models.PayoutDetailsInfo{
			AccountHolderName: existing.AccountHolder,
			AccountNumber:     existing.AccountNumber,
			IFSCCode:          existing.IFSCCode,
			BankName:          existing.BankName,
			AccountType:       existing.AccountType,
			UPIID:             existing.UPIID,
			Verified:          existing.Verified,
			UpdatedAt:         existing.UpdatedAt,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetEarningsBreakdown returns detailed earnings breakdown by series/episode
// GET /api/creators/{creator_id}/earnings/breakdown
// GetEarningsBreakdown godoc
// @Summary Get Earnings Breakdown
// @Description Get detailed earnings breakdown by series.
// @Tags creator-payment
// @Produce  json
// @Param   creator_id  path     string  true  "Creator ID"
// @Param   start_date  query    string  false "Start Date (limit)"
// @Param   end_date    query    string  false "End Date (limit)"
// @Param   series_id   query    string  false "Series ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/creators/{creator_id}/earnings/breakdown [get]
func (h *CreatorPaymentHandler) GetEarningsBreakdown(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creatorID := vars["creator_id"]

	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusInternalServerError)
		return
	}

	// Verify the creator belongs to this user
	var creator models.CreatorProfile
	if err := h.db.Where("id = ? AND user_id = ?", creatorID, userID).First(&creator).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Creator not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Parse date filters
	var startDate, endDate time.Time
	now := time.Now()

	if sd := r.URL.Query().Get("start_date"); sd != "" {
		if parsed, err := time.Parse(time.RFC3339, sd); err == nil {
			startDate = parsed
		}
	}
	if startDate.IsZero() {
		// Default to 30 days ago
		startDate = now.AddDate(0, -1, 0)
	}

	if ed := r.URL.Query().Get("end_date"); ed != "" {
		if parsed, err := time.Parse(time.RFC3339, ed); err == nil {
			endDate = parsed
		}
	}
	if endDate.IsZero() {
		endDate = now
	}

	// Filter by series if provided
	seriesIDFilter := r.URL.Query().Get("series_id")

	// Get total earnings for period
	var totalEarnings float64
	query := h.db.Model(&models.CreatorEarnings{}).
		Where("creator_id = ? AND status != 'cancelled' AND created_at >= ? AND created_at <= ?", creatorID, startDate, endDate)
	if seriesIDFilter != "" {
		query = query.Where("series_id = ?", seriesIDFilter)
	}
	query.Select("COALESCE(SUM(amount), 0)").Scan(&totalEarnings)

	// Get series breakdown
	type SeriesStats struct {
		SeriesID   string  `gorm:"column:series_id"`
		Title      string  `gorm:"column:title"`
		SubRevenue float64 `gorm:"column:sub_revenue"`
		OTPRevenue float64 `gorm:"column:otp_revenue"`
		TotalViews int64   `gorm:"column:total_views"`
	}

	var seriesStats []SeriesStats
	seriesQuery := h.db.Raw(`
		SELECT 
			s.id as series_id,
			s.title,
			COALESCE(SUM(CASE WHEN ce.earnings_type = 'subscription' THEN ce.amount ELSE 0 END), 0) as sub_revenue,
			COALESCE(SUM(CASE WHEN ce.earnings_type = 'one_time' THEN ce.amount ELSE 0 END), 0) as otp_revenue,
			s.view_count as total_views
		FROM series s
		LEFT JOIN creator_earnings ce ON ce.series_id = s.id 
			AND ce.status != 'cancelled' 
			AND ce.created_at >= ? 
			AND ce.created_at <= ?
		WHERE s.creator_id = ?
		GROUP BY s.id, s.title, s.view_count
		ORDER BY (COALESCE(SUM(ce.amount), 0)) DESC
	`, startDate, endDate, creatorID)

	if seriesIDFilter != "" {
		seriesQuery = h.db.Raw(`
			SELECT 
				s.id as series_id,
				s.title,
				COALESCE(SUM(CASE WHEN ce.earnings_type = 'subscription' THEN ce.amount ELSE 0 END), 0) as sub_revenue,
				COALESCE(SUM(CASE WHEN ce.earnings_type = 'one_time' THEN ce.amount ELSE 0 END), 0) as otp_revenue,
				s.view_count as total_views
			FROM series s
			LEFT JOIN creator_earnings ce ON ce.series_id = s.id 
				AND ce.status != 'cancelled' 
				AND ce.created_at >= ? 
				AND ce.created_at <= ?
			WHERE s.creator_id = ? AND s.id = ?
			GROUP BY s.id, s.title, s.view_count
		`, startDate, endDate, creatorID, seriesIDFilter)
	}
	seriesQuery.Scan(&seriesStats)

	seriesBreakdown := make([]models.SeriesEarningsBreakdown, len(seriesStats))
	for i, ss := range seriesStats {
		// Since subscriptions are global (not series-specific), we count unique subscribers
		// who generated subscription earnings for this specific series
		var activeSubCount int64
		h.db.Model(&models.CreatorEarnings{}).
			Where("series_id = ? AND earnings_type = 'subscription' AND status = 'pending'", ss.SeriesID).
			Distinct("subscription_id").
			Count(&activeSubCount)

		seriesBreakdown[i] = models.SeriesEarningsBreakdown{
			SeriesID:            ss.SeriesID,
			SeriesTitle:         ss.Title,
			Earnings:            ss.SubRevenue + ss.OTPRevenue,
			SubscriptionRevenue: ss.SubRevenue,
			OneTimePurchases:    ss.OTPRevenue,
			ActiveSubscriptions: activeSubCount,
			TotalViews:          ss.TotalViews,
		}
	}

	// Get episode breakdown (simplified - just earnings per episode)
	type EpisodeStats struct {
		EpisodeID string  `gorm:"column:episode_id"`
		Title     string  `gorm:"column:title"`
		SeriesID  string  `gorm:"column:series_id"`
		Earnings  float64 `gorm:"column:earnings"`
	}

	var episodeStats []EpisodeStats
	episodeQuery := h.db.Raw(`
		SELECT 
			e.id as episode_id,
			e.title,
			e.series_id,
			COALESCE(SUM(ce.amount), 0) as earnings
		FROM episodes e
		JOIN series s ON e.series_id = s.id
		LEFT JOIN creator_earnings ce ON ce.episode_id = e.id 
			AND ce.status != 'cancelled'
			AND ce.created_at >= ?
			AND ce.created_at <= ?
		WHERE s.creator_id = ?
		GROUP BY e.id, e.title, e.series_id
		HAVING COALESCE(SUM(ce.amount), 0) > 0
		ORDER BY earnings DESC
		LIMIT 20
	`, startDate, endDate, creatorID)

	if seriesIDFilter != "" {
		episodeQuery = h.db.Raw(`
			SELECT 
				e.id as episode_id,
				e.title,
				e.series_id,
				COALESCE(SUM(ce.amount), 0) as earnings
			FROM episodes e
			JOIN series s ON e.series_id = s.id
			LEFT JOIN creator_earnings ce ON ce.episode_id = e.id 
				AND ce.status != 'cancelled'
				AND ce.created_at >= ?
				AND ce.created_at <= ?
			WHERE s.creator_id = ? AND s.id = ?
			GROUP BY e.id, e.title, e.series_id
			ORDER BY earnings DESC
		`, startDate, endDate, creatorID, seriesIDFilter)
	}
	episodeQuery.Scan(&episodeStats)

	episodeBreakdown := make([]models.EpisodeEarningsBreakdown, len(episodeStats))
	for i, es := range episodeStats {
		// Get view/watch stats from analytics if available
		var analytics models.CreatorAnalytics
		h.db.Where("creator_id = ? AND date >= ? AND date <= ?", creatorID, startDate, endDate).
			First(&analytics)

		episodeBreakdown[i] = models.EpisodeEarningsBreakdown{
			EpisodeID:        es.EpisodeID,
			EpisodeTitle:     es.Title,
			SeriesID:         es.SeriesID,
			Earnings:         es.Earnings,
			Views:            analytics.Views,
			WatchTimeSeconds: analytics.WatchTimeSeconds,
		}
	}

	response := map[string]interface{}{
		"period": map[string]interface{}{
			"start_date": startDate.Format(time.RFC3339),
			"end_date":   endDate.Format(time.RFC3339),
		},
		"total_earnings":    totalEarnings,
		"series_breakdown":  seriesBreakdown,
		"episode_breakdown": episodeBreakdown,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TrackSeriesView tracks a view for a series
// POST /api/series/{series_id}/view
func (h *CreatorPaymentHandler) TrackSeriesView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	seriesID := vars["series_id"]

	// Get optional user ID from context (might be anonymous view)
	var userID *string
	if uid, ok := r.Context().Value("user_id").(string); ok && uid != "" {
		userID = &uid
	}

	// Verify series exists
	var series models.Series
	if err := h.db.Where("id = ?", seriesID).First(&series).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Series not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Get request metadata
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}
	userAgent := r.Header.Get("User-Agent")

	alreadyViewed := false

	// For authenticated users: only increment view_count on first view (INSERT succeeds)
	if userID != nil {
		// Try to insert a new view record; if conflict (already exists), do nothing
		result := h.db.Exec(`
			INSERT INTO series_views (id, series_id, user_id, view_count, ip_address, user_agent, created_at, updated_at)
			VALUES (gen_random_uuid(), ?, ?, 1, ?, ?, NOW(), NOW())
			ON CONFLICT (series_id, user_id) WHERE user_id IS NOT NULL
			DO NOTHING
		`, seriesID, *userID, ipAddress, userAgent)

		if result.Error != nil {
			log.Printf("Warning: Failed to insert series view record: %v", result.Error)
		}

		// Only increment series.view_count if this was a first-time view (row was inserted)
		if result.RowsAffected > 0 {
			if err := h.db.Model(&series).Update("view_count", gorm.Expr("view_count + 1")).Error; err != nil {
				log.Printf("Warning: Failed to increment view count: %v", err)
			}
		} else {
			alreadyViewed = true
		}
	} else {
		// For anonymous users: create individual records BUT don't increment view_count
		// (anonymous views are not counted to prevent abuse)
		view := models.SeriesView{
			SeriesID:  seriesID,
			UserID:    nil,
			ViewCount: 1,
			IPAddress: &ipAddress,
			UserAgent: &userAgent,
		}
		if err := h.db.Create(&view).Error; err != nil {
			log.Printf("Warning: Failed to create anonymous series view record: %v", err)
		}
		// Note: For anonymous users, we don't increment series.view_count
		// This prevents gaming the view count metric
	}

	// Reload to get current count
	h.db.Where("id = ?", seriesID).First(&series)

	response := map[string]interface{}{
		"success":        true,
		"series_id":      seriesID,
		"view_count":     series.ViewCount,
		"already_viewed": alreadyViewed,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandlePayoutWebhook handles Razorpay payout webhooks
// POST /payments/payout-webhook
func (h *CreatorPaymentHandler) HandlePayoutWebhook(w http.ResponseWriter, r *http.Request) {
	var webhookPayload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&webhookPayload); err != nil {
		log.Printf("[payout-webhook] Failed to decode webhook payload: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	event, ok := webhookPayload["event"].(string)
	if !ok {
		log.Printf("[payout-webhook] Missing event field")
		http.Error(w, "Missing event field", http.StatusBadRequest)
		return
	}

	log.Printf("[payout-webhook] Received event: %s", event)

	payload, _ := webhookPayload["payload"].(map[string]interface{})
	if payload == nil {
		payload = webhookPayload
	}

	switch event {
	case "payout.processed":
		h.handlePayoutProcessed(payload)
	case "payout.failed":
		h.handlePayoutFailed(payload)
	case "payout.reversed":
		h.handlePayoutReversed(payload)
	default:
		log.Printf("[payout-webhook] Unknown event: %s", event)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "processed"})
}

func (h *CreatorPaymentHandler) handlePayoutProcessed(data map[string]interface{}) {
	payoutEntity := extractPayoutEntity(data)
	if payoutEntity == nil {
		return
	}

	razorpayPayoutID, ok := payoutEntity["id"].(string)
	if !ok {
		return
	}

	// Find payout by Razorpay ID
	var payout models.Payout
	if err := h.db.Where("razorpay_payout_id = ?", razorpayPayoutID).First(&payout).Error; err != nil {
		log.Printf("[payout-webhook] Payout not found for ID %s: %v", razorpayPayoutID, err)
		return
	}

	now := time.Now()

	// Get UTR (transaction reference) if available
	var utr *string
	if u, ok := payoutEntity["utr"].(string); ok {
		utr = &u
	}

	// Update payout status
	updates := map[string]interface{}{
		"status":                "completed",
		"completed_at":          now,
		"transaction_reference": utr,
		"updated_at":            now,
	}

	h.db.Model(&payout).Updates(updates)

	// Mark all linked earnings as paid
	h.db.Model(&models.CreatorEarnings{}).
		Where("payout_id = ?", payout.ID).
		Updates(map[string]interface{}{
			"status":     "paid",
			"paid_at":    now,
			"updated_at": now,
		})

	log.Printf("[payout-webhook] Payout %s completed successfully", razorpayPayoutID)
}

func (h *CreatorPaymentHandler) handlePayoutFailed(data map[string]interface{}) {
	payoutEntity := extractPayoutEntity(data)
	if payoutEntity == nil {
		return
	}

	razorpayPayoutID, ok := payoutEntity["id"].(string)
	if !ok {
		return
	}

	var payout models.Payout
	if err := h.db.Where("razorpay_payout_id = ?", razorpayPayoutID).First(&payout).Error; err != nil {
		return
	}

	// Get failure reason
	var failureReason *string
	if reason, ok := payoutEntity["failure_reason"].(string); ok {
		failureReason = &reason
	}

	// Update payout status
	h.db.Model(&payout).Updates(map[string]interface{}{
		"status":         "failed",
		"failure_reason": failureReason,
		"updated_at":     time.Now(),
	})

	// Unlink earnings from failed payout (so they can be paid out again)
	h.db.Model(&models.CreatorEarnings{}).
		Where("payout_id = ?", payout.ID).
		Updates(map[string]interface{}{
			"payout_id":  nil,
			"updated_at": time.Now(),
		})

	log.Printf("[payout-webhook] Payout %s failed: %v", razorpayPayoutID, failureReason)
}

func (h *CreatorPaymentHandler) handlePayoutReversed(data map[string]interface{}) {
	// Similar to failed - mark as failed and unlink earnings
	h.handlePayoutFailed(data)
}

func extractPayoutEntity(data map[string]interface{}) map[string]interface{} {
	if payoutWrapper, ok := data["payout"].(map[string]interface{}); ok {
		if entity, ok := payoutWrapper["entity"].(map[string]interface{}); ok {
			return entity
		}
	}
	if entity, ok := data["entity"].(map[string]interface{}); ok {
		return entity
	}
	return nil
}

// RecordEarnings is a helper function to record earnings for a creator
// Called when a subscription is created or a one-time purchase is made
func (h *CreatorPaymentHandler) RecordEarnings(creatorID string, amount float64, earningsType string, seriesID, episodeID, subscriptionID *string) error {
	// Calculate creator's share (70%)
	creatorAmount := amount * models.CreatorSharePercent

	earnings := models.CreatorEarnings{
		CreatorID:      creatorID,
		Amount:         creatorAmount,
		EarningsType:   earningsType,
		SeriesID:       seriesID,
		EpisodeID:      episodeID,
		SubscriptionID: subscriptionID,
		Status:         "pending",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return h.db.Create(&earnings).Error
}
