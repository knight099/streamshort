package handlers

import (
	"encoding/json"
	"fmt"
	mathrand "math/rand"
	"net/http"
	"strconv"
	"time"

	"streamshort/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{db: db}
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

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// JWT Claims
type Claims struct {
	UserID string `json:"user_id"`
	Phone  string `json:"phone"`
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

	// Generate OTP (6 digits)
	otp := generateOTP()

	// Generate transaction ID
	txnID := "otp_txn_" + uuid.New().String()[:8]

	// Create OTP transaction
	otpTx := models.OTPTransaction{
		TxnID:     txnID,
		Phone:     req.Phone,
		OTP:       otp,
		ExpiresAt: time.Now().Add(OTPExpiration),
	}

	if err := h.db.Create(&otpTx).Error; err != nil {
		http.Error(w, "Failed to create OTP transaction", http.StatusInternalServerError)
		return
	}

	// In a real application, you would send the OTP via SMS here
	// For now, we'll just log it
	fmt.Printf("OTP for %s: %s\n", req.Phone, otp)

	response := PhoneOtpSendResponse{
		TxnID:     txnID,
		ExpiresIn: int(OTPExpiration.Seconds()),
		Message:   fmt.Sprintf("OTP sent to %s", req.Phone),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Verify OTP endpoint
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

// Helper functions
func (h *AuthHandler) generateAccessToken(user models.User) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Phone:  user.Phone,
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
