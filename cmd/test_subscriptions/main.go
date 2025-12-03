package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const baseURL = "http://localhost:8080"

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type SubscriptionPlan struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Duration    int     `json:"duration"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	IsActive    bool    `json:"is_active"`
}

type CreateSubscriptionRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
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

type Subscription struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Status     string    `json:"status"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	AutoRenew  bool      `json:"auto_renew"`
	PlanID     string    `json:"plan_id"`
	Amount     float64   `json:"amount"`
	Currency   string    `json:"currency"`
}

type SubscriptionsResponse struct {
	Total         int            `json:"total"`
	Page          int            `json:"page"`
	PerPage       int            `json:"per_page"`
	TotalPages    int            `json:"total_pages"`
	Subscriptions []Subscription `json:"subscriptions"`
}

type CheckStatusResponse struct {
	HasAccess    bool          `json:"has_access"`
	TargetType   string        `json:"target_type"`
	TargetID     string        `json:"target_id"`
	Subscription *Subscription `json:"subscription_details,omitempty"`
}

func main() {
	fmt.Println("🧪 Testing Subscription API Endpoints")
	fmt.Println("======================================")
	fmt.Println()

	// Check if server is running
	fmt.Println("🔍 Checking if server is running...")
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		fmt.Printf("❌ Server is not running: %v\n", err)
		fmt.Println("Please start the server with: go run main.go")
		os.Exit(1)
	}
	resp.Body.Close()
	fmt.Println("✅ Server is running")
	fmt.Println()

	// Get auth token (using a test approach)
	token := getTestToken()
	if token == "" {
		fmt.Println("⚠️  Could not get authentication token")
		fmt.Println("The subscription APIs require authentication.")
		fmt.Println()
		fmt.Println("To test manually, you need to:")
		fmt.Println("1. Set up Firebase authentication")
		fmt.Println("2. Get a valid token using Firebase OTP flow")
		fmt.Println("3. Use that token in the Authorization header")
		fmt.Println()
		fmt.Println("Testing public endpoints instead...")
		testPublicEndpoints()
		return
	}

	// Run tests
	testGetPlans(token)
	testCreateSubscription(token)
	testGetSubscriptions(token)
	testCheckStatus(token)

	fmt.Println()
	fmt.Println("✅ All subscription API tests completed!")
}

func getTestToken() string {
	// Try to create a test user and get a token
	// This is a simplified approach for testing
	// In production, you would use proper Firebase authentication

	fmt.Println("🔐 Attempting to get authentication token...")
	fmt.Println("⚠️  Note: Firebase authentication is required")
	fmt.Println()

	return ""
}

func testPublicEndpoints() {
	fmt.Println("📋 Testing public endpoints")
	fmt.Println("GET /content/series")

	resp, err := http.Get(baseURL + "/content/series")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(body))

	if resp.StatusCode == 200 {
		fmt.Println("✅ Public endpoint working")
	}
}

func testGetPlans(token string) {
	fmt.Println("📋 Test 1: Get available subscription plans")
	fmt.Println("GET /api/subscriptions/plans")

	req, _ := http.NewRequest("GET", baseURL+"/api/subscriptions/plans", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)

	if resp.StatusCode == 200 {
		var plans []SubscriptionPlan
		json.Unmarshal(body, &plans)
		fmt.Printf("✅ Found %d subscription plans\n", len(plans))
		for _, plan := range plans {
			fmt.Printf("   - %s: %.2f %s for %d days\n", plan.Name, plan.Amount, plan.Currency, plan.Duration)
		}
	} else {
		fmt.Printf("Response: %s\n", string(body))
	}
	fmt.Println()
}

func testCreateSubscription(token string) {
	fmt.Println("💳 Test 2: Create a subscription")
	fmt.Println("POST /api/payments/create-subscription")

	reqBody := CreateSubscriptionRequest{
		TargetType: "series",
		TargetID:   "test-series-id",
		PlanID:     "plan_basic_monthly",
		AutoRenew:  true,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", baseURL+"/api/payments/create-subscription", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		var sub CreateSubscriptionResponse
		json.Unmarshal(body, &sub)
		fmt.Printf("✅ Subscription created: %s (status: %s)\n", sub.SubscriptionID, sub.Status)
	} else {
		fmt.Printf("Response: %s\n", string(body))
	}
	fmt.Println()
}

func testGetSubscriptions(token string) {
	fmt.Println("📚 Test 3: Get user subscriptions")
	fmt.Println("GET /api/subscriptions")

	req, _ := http.NewRequest("GET", baseURL+"/api/subscriptions", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)

	if resp.StatusCode == 200 {
		var subs SubscriptionsResponse
		json.Unmarshal(body, &subs)
		fmt.Printf("✅ Found %d subscriptions\n", subs.Total)
		for _, sub := range subs.Subscriptions {
			fmt.Printf("   - %s: %s (status: %s)\n", sub.TargetType, sub.TargetID, sub.Status)
		}
	} else {
		fmt.Printf("Response: %s\n", string(body))
	}
	fmt.Println()
}

func testCheckStatus(token string) {
	fmt.Println("🔍 Test 4: Check subscription status")
	fmt.Println("GET /api/subscriptions/check?target_type=series&target_id=test-series-id")

	req, _ := http.NewRequest("GET", baseURL+"/api/subscriptions/check?target_type=series&target_id=test-series-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)

	if resp.StatusCode == 200 {
		var status CheckStatusResponse
		json.Unmarshal(body, &status)
		fmt.Printf("✅ Has access: %v\n", status.HasAccess)
		if status.Subscription != nil {
			fmt.Printf("   Subscription status: %s\n", status.Subscription.Status)
		}
	} else {
		fmt.Printf("Response: %s\n", string(body))
	}
	fmt.Println()
}
