package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	mathrand "math/rand"
	"net/http"
	"strconv"
	"time"

	"streamshort/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db             *gorm.DB
	firebaseAPIKey string
}

func NewAuthHandler(db *gorm.DB, firebaseAPIKey string) *AuthHandler {
	return &AuthHandler{db: db, firebaseAPIKey: firebaseAPIKey}
}

// Request/Response structs matching OpenAPI schema
type PhoneOtpRequest struct {
	Phone string `json:"phone"`
}

type PhoneOtpSendResponse struct {
	TxnID     string `json:"txn_id"`
	ExpiresIn int    `json:"expires_in"`
	Message   string `json:"message"`
}

type PhoneOtpVerifyRequest struct {
	Phone string `json:"phone"`
	OTP   string `json:"otp"`
}

type AdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateAdminRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// Firebase OTP: send code to phone using Identity Toolkit REST API
// Client must provide a reCAPTCHA token obtained from Firebase SDK on client side
// FirebaseSendOTP godoc
// @Summary Send OTP via Firebase
// @Description Sends an OTP to the given phone number using Firebase Identity Toolkit.
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   request  body     FirebaseSendOTPRequest  true  "OTP Request"
// @Success 200 {object} FirebaseSendOTPResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /auth/firebase/otp/send [post]
func (h *AuthHandler) FirebaseSendOTP(w http.ResponseWriter, r *http.Request) {
	if h.firebaseAPIKey == "" {
		log.Printf("[auth] FirebaseSendOTP: missing Firebase API key")
		http.Error(w, "Firebase API key not configured", http.StatusInternalServerError)
		return
	}

	var req FirebaseSendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[auth] FirebaseSendOTP: invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Phone == "" || req.RecaptchaToken == "" {
		log.Printf("[auth] FirebaseSendOTP: missing phone or recaptcha_token (phone=%s)", req.Phone)
		http.Error(w, "phone and recaptcha_token are required", http.StatusBadRequest)
		return
	}

	payload := map[string]interface{}{
		"phoneNumber":    req.Phone,
		"recaptchaToken": req.RecaptchaToken,
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:sendVerificationCode?key=%s", h.firebaseAPIKey)
	httpReq, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	log.Printf("[auth] FirebaseSendOTP: sending verification code (phone=%s)", req.Phone)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Printf("[auth] FirebaseSendOTP: request error: %v", err)
		http.Error(w, "failed to contact Firebase", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		log.Printf("[auth] FirebaseSendOTP: firebase responded %d: %v", resp.StatusCode, errBody)
		w.WriteHeader(resp.StatusCode)
		_ = json.NewEncoder(w).Encode(errBody)
		return
	}

	var firebaseResp struct {
		SessionInfo string `json:"sessionInfo"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&firebaseResp); err != nil {
		log.Printf("[auth] FirebaseSendOTP: decode response error: %v", err)
		http.Error(w, "failed to parse Firebase response", http.StatusBadGateway)
		return
	}

	log.Printf("[auth] FirebaseSendOTP: success (phone=%s)", req.Phone)
	out := FirebaseSendOTPResponse{SessionInfo: firebaseResp.SessionInfo, Message: "OTP sent via Firebase"}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// Firebase OTP: verify the code and sign in
// FirebaseVerifyOTP godoc
// @Summary Verify Firebase OTP
// @Description Verifies the OTP sent via Firebase and returns access tokens.
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   request  body     FirebaseVerifyOTPRequest  true  "Verify Request"
// @Success 200 {object} TokenResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /auth/firebase/otp/verify [post]
func (h *AuthHandler) FirebaseVerifyOTP(w http.ResponseWriter, r *http.Request) {
	if h.firebaseAPIKey == "" {
		log.Printf("[auth] FirebaseVerifyOTP: missing Firebase API key")
		http.Error(w, "Firebase API key not configured", http.StatusInternalServerError)
		return
	}

	var req FirebaseVerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[auth] FirebaseVerifyOTP: invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.SessionInfo == "" || req.Code == "" {
		log.Printf("[auth] FirebaseVerifyOTP: missing session_info or code")
		http.Error(w, "session_info and code are required", http.StatusBadRequest)
		return
	}

	payload := map[string]interface{}{
		"sessionInfo": req.SessionInfo,
		"code":        req.Code,
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithPhoneNumber?key=%s", h.firebaseAPIKey)
	httpReq, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	log.Printf("[auth] FirebaseVerifyOTP: verifying code with Firebase")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Printf("[auth] FirebaseVerifyOTP: request error: %v", err)
		http.Error(w, "failed to contact Firebase", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		log.Printf("[auth] FirebaseVerifyOTP: firebase responded %d: %v", resp.StatusCode, errBody)
		w.WriteHeader(resp.StatusCode)
		_ = json.NewEncoder(w).Encode(errBody)
		return
	}

	var firebaseResp struct {
		PhoneNumber  string `json:"phoneNumber"`
		IDToken      string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    string `json:"expiresIn"`
		LocalID      string `json:"localId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&firebaseResp); err != nil {
		log.Printf("[auth] FirebaseVerifyOTP: decode response error: %v", err)
		http.Error(w, "failed to parse Firebase response", http.StatusBadGateway)
		return
	}

	phone := firebaseResp.PhoneNumber
	if phone == "" {
		log.Printf("[auth] FirebaseVerifyOTP: missing phoneNumber in Firebase response")
		http.Error(w, "Firebase did not return phoneNumber", http.StatusBadGateway)
		return
	}

	// Get or create user in our DB
	var user models.User
	if err := h.db.Where("phone = ?", phone).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[auth] FirebaseVerifyOTP: creating user (phone=%s)", phone)
			user = models.User{Phone: phone}
			if err := h.db.Create(&user).Error; err != nil {
				log.Printf("[auth] FirebaseVerifyOTP: create user error: %v", err)
				http.Error(w, "Failed to create user", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("[auth] FirebaseVerifyOTP: db error: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	// Issue our own app tokens
	accessToken, err := h.generateAccessToken(user)
	if err != nil {
		log.Printf("[auth] FirebaseVerifyOTP: access token generation error: %v", err)
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}
	refreshToken, err := h.generateRefreshToken(user.ID)
	if err != nil {
		log.Printf("[auth] FirebaseVerifyOTP: refresh token generation error: %v", err)
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	log.Printf("[auth] FirebaseVerifyOTP: success (phone=%s, user_id=%s)", phone, user.ID)
	response := TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(TokenExpiration.Seconds()),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// Exchange a Firebase ID token (from native SDK signInWithPhoneNumber) for app tokens
// FirebaseExchangeIDToken godoc
// @Summary Exchange Firebase ID Token
// @Description Exchange a Firebase ID token for application access/refresh tokens.
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   request  body     FirebaseExchangeRequest  true  "Exchange Request"
// @Success 200 {object} TokenResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal Server Error"
// @Router /auth/firebase/exchange [post]
func (h *AuthHandler) FirebaseExchangeIDToken(w http.ResponseWriter, r *http.Request) {
	if h.firebaseAPIKey == "" {
		log.Printf("[auth] FirebaseExchangeIDToken: missing Firebase API key")
		http.Error(w, "Firebase API key not configured", http.StatusInternalServerError)
		return
	}

	var req FirebaseExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[auth] FirebaseExchangeIDToken: invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.IDToken == "" {
		log.Printf("[auth] FirebaseExchangeIDToken: missing id_token")
		http.Error(w, "id_token is required", http.StatusBadRequest)
		return
	}

	// Verify ID token and lookup user via Identity Toolkit
	payload := map[string]interface{}{
		"idToken": req.IDToken,
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:lookup?key=%s", h.firebaseAPIKey)
	httpReq, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	log.Printf("[auth] FirebaseExchangeIDToken: looking up id token with Firebase")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Printf("[auth] FirebaseExchangeIDToken: request error: %v", err)
		http.Error(w, "failed to contact Firebase", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		log.Printf("[auth] FirebaseExchangeIDToken: firebase responded %d: %v", resp.StatusCode, errBody)
		w.WriteHeader(resp.StatusCode)
		_ = json.NewEncoder(w).Encode(errBody)
		return
	}

	var lookup struct {
		Users []struct {
			LocalID     string `json:"localId"`
			PhoneNumber string `json:"phoneNumber"`
		} `json:"users"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&lookup); err != nil {
		log.Printf("[auth] FirebaseExchangeIDToken: decode response error: %v", err)
		http.Error(w, "failed to parse Firebase response", http.StatusBadGateway)
		return
	}
	if len(lookup.Users) == 0 || lookup.Users[0].PhoneNumber == "" {
		log.Printf("[auth] FirebaseExchangeIDToken: missing phone number in Firebase user")
		http.Error(w, "Firebase user missing phone number", http.StatusUnauthorized)
		return
	}
	phone := lookup.Users[0].PhoneNumber

	// Get or create our user
	var user models.User
	if err := h.db.Where("phone = ?", phone).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[auth] FirebaseExchangeIDToken: creating user (phone=%s)", phone)
			user = models.User{Phone: phone}
			if err := h.db.Create(&user).Error; err != nil {
				log.Printf("[auth] FirebaseExchangeIDToken: create user error: %v", err)
				http.Error(w, "Failed to create user", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("[auth] FirebaseExchangeIDToken: db error: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	// Issue our app tokens
	accessToken, err := h.generateAccessToken(user)
	if err != nil {
		log.Printf("[auth] FirebaseExchangeIDToken: access token generation error: %v", err)
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}
	refreshToken, err := h.generateRefreshToken(user.ID)
	if err != nil {
		log.Printf("[auth] FirebaseExchangeIDToken: refresh token generation error: %v", err)
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	log.Printf("[auth] FirebaseExchangeIDToken: success (phone=%s, user_id=%s)", phone, user.ID)
	response := TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(TokenExpiration.Seconds()),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Firebase Phone OTP (Identity Toolkit) request/response types
type FirebaseSendOTPRequest struct {
	Phone          string `json:"phone"`
	RecaptchaToken string `json:"recaptcha_token"`
}

type FirebaseSendOTPResponse struct {
	SessionInfo string `json:"session_info"`
	Message     string `json:"message"`
}

type FirebaseVerifyOTPRequest struct {
	SessionInfo string `json:"session_info"`
	Code        string `json:"code"`
}

type FirebaseExchangeRequest struct {
	IDToken string `json:"id_token"`
}

// JWT Claims
type Claims struct {
	UserID string `json:"user_id"`
	Phone  string `json:"phone,omitempty"`
	Role   string `json:"role"`
	Name   string `json:"name,omitempty"`
	jwt.RegisteredClaims
}

const (
	JWTSecret              = "your-secret-key-change-in-production"
	OTPExpiration          = 5 * time.Minute
	TokenExpiration        = 1 * time.Hour
	RefreshTokenExpiration = 7 * 24 * time.Hour
)

// GetJWTSecret returns the JWT secret for use in middleware
func GetJWTSecret() string {
	return JWTSecret
}

// Send OTP endpoint
// SendOTP godoc
// @Summary Send OTP (Deprecated)
// @Description Deprecated: Use /auth/firebase/otp/send instead.
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   request  body     PhoneOtpRequest  true  "OTP Request"
// @Success 200 {object} PhoneOtpSendResponse
// @Failure 400 {string} string "Bad Request"
// @Router /auth/otp/send [post]
func (h *AuthHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req PhoneOtpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Phone == "" {
		http.Error(w, "Phone number is required", http.StatusBadRequest)
		return
	}

	// Since we are moving to Firebase for OTP delivery, we keep this endpoint
	// for backward compatibility but simply acknowledge the request.
	response := PhoneOtpSendResponse{
		TxnID:     "deprecated",
		ExpiresIn: int(OTPExpiration.Seconds()),
		Message:   fmt.Sprintf("Deprecated: Use /auth/firebase/otp/send for %s", req.Phone),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Verify OTP endpoint
// VerifyOTP godoc
// @Summary Verify OTP (Deprecated)
// @Description Deprecated: Use /auth/firebase/otp/verify instead.
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   request  body     PhoneOtpVerifyRequest  true  "Verify Request"
// @Success 200 {object} TokenResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal Server Error"
// @Router /auth/otp/verify [post]
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req PhoneOtpVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Phone == "" || req.OTP == "" {
		http.Error(w, "Phone and OTP are required", http.StatusBadRequest)
		return
	}

	// Find OTP transaction
	// phone := "+91" + req.Phone
	var otpTx models.OTPTransaction
	if err := h.db.Where("phone = ? AND otp = ? AND used = ?",
		req.Phone, req.OTP, false).First(&otpTx).Error; err != nil {
		http.Error(w, "Invalid OTP", http.StatusUnauthorized)
		return
	}

	// Mark OTP as used
	h.db.Model(&otpTx).Update("used", true)

	// Get or create user
	var user models.User
	if err := h.db.Where("phone = ?", req.Phone).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new user
			user = models.User{Phone: req.Phone}
			if err := h.db.Create(&user).Error; err != nil {
				http.Error(w, "Failed to create user", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	// Generate tokens
	accessToken, err := h.generateAccessToken(user)
	if err != nil {
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := h.generateRefreshToken(user.ID)
	if err != nil {
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	response := TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(TokenExpiration.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Refresh token endpoint
// RefreshToken godoc
// @Summary Refresh Access Token
// @Description Use a refresh token to get a new access token.
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   request  body     RefreshRequest  true  "Refresh Request"
// @Success 200 {object} TokenResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal Server Error"
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.RefreshToken == "" {
		http.Error(w, "Refresh token is required", http.StatusBadRequest)
		return
	}

	// Find refresh token
	var refreshToken models.RefreshToken
	if err := h.db.Where("token = ? AND revoked = ? AND expires_at > ?",
		req.RefreshToken, false, time.Now()).First(&refreshToken).Error; err != nil {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	// Get user
	var user models.User
	if err := h.db.First(&user, refreshToken.UserID).Error; err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Generate new tokens
	accessToken, err := h.generateAccessToken(user)
	if err != nil {
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	newRefreshToken, err := h.generateRefreshToken(user.ID)
	if err != nil {
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	// Revoke old refresh token
	h.db.Model(&refreshToken).Update("revoked", true)

	response := TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(TokenExpiration.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AdminLogin handles username/password login for admins
// AdminLogin godoc
// @Summary Admin Login
// @Description Login for admin users using username and password.
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   request  body     AdminLoginRequest  true  "Login Request"
// @Success 200 {object} TokenResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal Server Error"
// @Router /auth/admin/login [post]
func (h *AuthHandler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	var req AdminLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	var user models.User
	if err := h.db.Where("username = ? AND role = 'admin'", req.Username).First(&user).Error; err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate tokens
	accessToken, err := h.generateAccessToken(user)
	if err != nil {
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := h.generateRefreshToken(user.ID)
	if err != nil {
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	response := TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(TokenExpiration.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// EnsureDefaultAdmin checks if any admin exists, if not creates one
func (h *AuthHandler) EnsureDefaultAdmin() {
	var count int64
	h.db.Model(&models.User{}).Where("role = ?", "admin").Count(&count)
	if count == 0 {
		log.Println("[auth] No admin found. Creating default admin...")
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		adminUsername := "admin"
		admin := models.User{
			Phone:        "admin_default_" + uuid.New().String(),
			Username:     &adminUsername,
			PasswordHash: string(hashedPassword),
			Role:         "admin",
			Name:         "Default Admin",
		}
		if err := h.db.Create(&admin).Error; err != nil {
			log.Printf("[auth] Failed to create default admin: %v", err)
		} else {
			log.Println("[auth] Default admin created: username=admin, password=admin123")
		}
	}
}

// CreateAdmin creates a new admin user (Protected by Middleware in main)
// CreateAdmin godoc
// @Summary Create Admin
// @Description Create a new admin user (Protected).
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   request  body     CreateAdminRequest  true  "Create Admin Request"
// @Success 201 {object} map[string]string
// @Failure 400 {string} string "Bad Request"
// @Failure 409 {string} string "Conflict"
// @Failure 500 {string} string "Internal Server Error"
// @Router /auth/create-admin [post]
func (h *AuthHandler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	var req CreateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Check if username already exists
	var existingUser models.User
	if err := h.db.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		http.Error(w, "Username already taken", http.StatusConflict)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	// Generate unique placeholder phone for admin accounts (avoids unique constraint issue)
	adminPhone := "admin_" + uuid.New().String()

	username := req.Username
	user := models.User{
		Phone:        adminPhone,
		Username:     &username,
		PasswordHash: string(hashedPassword),
		Role:         "admin",
		Name:         req.Username, // Use username as display name
	}

	if err := h.db.Create(&user).Error; err != nil {
		log.Printf("[auth] CreateAdmin: db error: %v", err)
		http.Error(w, "Failed to create admin", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Admin created successfully",
		"user_id": user.ID,
	})
}

// Helper functions
func (h *AuthHandler) generateAccessToken(user models.User) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Phone:  user.Phone,
		Role:   user.Role,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWTSecret))
}

func (h *AuthHandler) generateRefreshToken(userID string) (string, error) {
	token := "rfrsh_" + uuid.New().String()

	refreshToken := models.RefreshToken{
		Token:     token,
		UserID:    userID,
		ExpiresAt: time.Now().Add(RefreshTokenExpiration),
	}

	if err := h.db.Create(&refreshToken).Error; err != nil {
		return "", err
	}

	return token, nil
}

func generateOTP() string {
	// Generate 6-digit OTP
	otp := ""
	for i := 0; i < 6; i++ {
		otp += strconv.Itoa(mathrand.Intn(10))
	}
	return otp
}

type RecaptchaSiteKeyResponse struct {
	SiteKey string `json:"site_key"`
}

// GetRecaptchaSiteKey godoc
// @Summary Get Recaptcha Site Key
// @Description Returns the reCAPTCHA site key for the client.
// @Tags auth
// @Produce  json
// @Success 200 {object} RecaptchaSiteKeyResponse
// @Router /auth/recaptcha/site-key [get]
func (h *AuthHandler) GetRecaptchaSiteKey(w http.ResponseWriter, r *http.Request) {
	// This handler will be wired with a closure to include the site key
	w.Header().Set("Content-Type", "application/json")
	// Placeholder will be replaced in main wiring via a bound handler
	_ = json.NewEncoder(w).Encode(RecaptchaSiteKeyResponse{SiteKey: ""})
}
