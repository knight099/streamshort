package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type RazorpayClient struct {
	KeyID     string
	KeySecret string
	BaseURL   string
}

func NewRazorpayClient(keyID, keySecret string) *RazorpayClient {
	baseURL := "https://api.razorpay.com/v1"
	if keyID != "" && len(keyID) > 0 && keyID[:4] == "rzp_" {
		// Test mode
		baseURL = "https://api.razorpay.com/v1"
	}
	return &RazorpayClient{
		KeyID:     keyID,
		KeySecret: keySecret,
		BaseURL:   baseURL,
	}
}

// CreateSubscription creates a subscription in Razorpay
func (c *RazorpayClient) CreateSubscription(planID string, customerNotify int, totalCount int, startAt int64) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"plan_id":         planID,
		"customer_notify": customerNotify,
		"total_count":     totalCount,
	}
	if startAt > 0 {
		payload["start_at"] = startAt
	}

	return c.makeRequest("POST", "/subscriptions", payload)
}

// CreatePaymentLink creates a payment link for a subscription
func (c *RazorpayClient) CreatePaymentLink(subscriptionID string, amount float64, currency string, description string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"amount":          amount * 100, // Razorpay expects amount in paise
		"currency":        currency,
		"description":     description,
		"subscription_id": subscriptionID,
	}

	return c.makeRequest("POST", "/payment_links", payload)
}

// makeRequest makes an authenticated request to Razorpay API
func (c *RazorpayClient) makeRequest(method, endpoint string, payload map[string]interface{}) (map[string]interface{}, error) {
	url := c.BaseURL + endpoint

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Basic Auth: key_id:key_secret base64 encoded
	auth := base64.StdEncoding.EncodeToString([]byte(c.KeyID + ":" + c.KeySecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorMsg := "unknown error"
		if errObj, ok := result["error"].(map[string]interface{}); ok {
			if desc, ok := errObj["description"].(string); ok {
				errorMsg = desc
			}
		}
		return nil, fmt.Errorf("razorpay API error (status %d): %s", resp.StatusCode, errorMsg)
	}

	return result, nil
}

// ================= RAZORPAY PAYOUTS API =================

// CreateContact creates a contact in Razorpay for payouts
func (c *RazorpayClient) CreateContact(name, referenceID string) (string, error) {
	payload := map[string]interface{}{
		"name":         name,
		"type":         "vendor",
		"reference_id": referenceID,
	}

	result, err := c.makeRequest("POST", "/contacts", payload)
	if err != nil {
		return "", err
	}

	contactID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid contact response: missing id")
	}

	return contactID, nil
}

// CreateBankFundAccount creates a bank account fund account for a contact
func (c *RazorpayClient) CreateBankFundAccount(contactID, accountName, accountNumber, ifscCode string) (string, error) {
	payload := map[string]interface{}{
		"contact_id":   contactID,
		"account_type": "bank_account",
		"bank_account": map[string]interface{}{
			"name":           accountName,
			"ifsc":           ifscCode,
			"account_number": accountNumber,
		},
	}

	result, err := c.makeRequest("POST", "/fund_accounts", payload)
	if err != nil {
		return "", err
	}

	fundAccountID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid fund account response: missing id")
	}

	return fundAccountID, nil
}

// CreateUPIFundAccount creates a UPI fund account for a contact
func (c *RazorpayClient) CreateUPIFundAccount(contactID, upiID string) (string, error) {
	payload := map[string]interface{}{
		"contact_id":   contactID,
		"account_type": "vpa",
		"vpa": map[string]interface{}{
			"address": upiID,
		},
	}

	result, err := c.makeRequest("POST", "/fund_accounts", payload)
	if err != nil {
		return "", err
	}

	fundAccountID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid fund account response: missing id")
	}

	return fundAccountID, nil
}

// CreatePayout creates a payout to a fund account
func (c *RazorpayClient) CreatePayout(fundAccountID string, amount float64, currency, mode, narration string) (string, error) {
	// Determine payout mode based on method
	payoutMode := "NEFT"
	if mode == "upi" {
		payoutMode = "UPI"
	} else if mode == "IMPS" || mode == "RTGS" || mode == "NEFT" {
		payoutMode = mode
	}

	payload := map[string]interface{}{
		"fund_account_id":      fundAccountID,
		"amount":               int64(amount * 100), // Amount in paise
		"currency":             currency,
		"mode":                 payoutMode,
		"purpose":              "payout",
		"queue_if_low_balance": true,
		"narration":            narration,
	}

	result, err := c.makeRequest("POST", "/payouts", payload)
	if err != nil {
		return "", err
	}

	payoutID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid payout response: missing id")
	}

	return payoutID, nil
}

// GetPayoutStatus gets the status of a payout
func (c *RazorpayClient) GetPayoutStatus(payoutID string) (map[string]interface{}, error) {
	return c.makeRequest("GET", "/payouts/"+payoutID, nil)
}
