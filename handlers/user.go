package handlers

import (
	"encoding/json"
	"net/http"

	"streamshort/models"

	"gorm.io/gorm"
)

type UserHandler struct {
	db *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

type UserProfileResponse struct {
	ID    string `json:"id"`
	Phone string `json:"phone"`
	Name  string `json:"name"`
}

type UpdateProfileRequest struct {
	Name string `json:"name"`
}

// GetProfile returns the authenticated user's profile
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userIDAny := r.Context().Value("user_id")
	if userIDAny == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, _ := userIDAny.(string)

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	resp := UserProfileResponse{ID: user.ID, Phone: user.Phone, Name: user.Name}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// UpdateProfile updates the authenticated user's name
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userIDAny := r.Context().Value("user_id")
	if userIDAny == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, _ := userIDAny.(string)

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Name) == 0 || len(req.Name) > 100 {
		http.Error(w, "Invalid name", http.StatusBadRequest)
		return
	}

	if err := h.db.Model(&models.User{}).Where("id = ?", userID).Update("name", req.Name).Error; err != nil {
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	resp := UserProfileResponse{ID: user.ID, Phone: user.Phone, Name: user.Name}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
