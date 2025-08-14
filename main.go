package main

import (
	"encoding/json"
	"log"
	"net/http"
	"streamshort/config"
	"streamshort/handlers"
	"streamshort/middleware"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Response struct {
	Message string `json:"message"`
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Message: "Hello World!",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Initialize database
	db := config.InitDB()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware()

	// Create router
	r := mux.NewRouter()

	// Public routes
	r.HandleFunc("/", helloHandler).Methods("GET")
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Message: "Server is running!"})
	}).Methods("GET")

	// Auth routes (matching OpenAPI schema)
	r.HandleFunc("/auth/otp/send", authHandler.SendOTP).Methods("POST")
	r.HandleFunc("/auth/otp/verify", authHandler.VerifyOTP).Methods("POST")
	r.HandleFunc("/auth/refresh", authHandler.RefreshToken).Methods("POST")

	// Protected routes (example)
	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(authMiddleware.AuthMiddleware)
	protected.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id")
		phone := r.Context().Value("phone")

		response := map[string]interface{}{
			"user_id": userID,
			"phone":   phone,
			"message": "Protected endpoint accessed successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	// Apply CORS middleware
	handler := c.Handler(r)

	log.Println("Server starting on port 8080...")
	log.Println("Available endpoints:")
	log.Println("  GET  /                    - Hello World")
	log.Println("  GET  /health              - Health check")
	log.Println("  POST /auth/otp/send       - Send OTP")
	log.Println("  POST /auth/otp/verify     - Verify OTP")
	log.Println("  POST /auth/refresh        - Refresh token")
	log.Println("  GET  /api/profile         - Protected profile (requires auth)")

	log.Fatal(http.ListenAndServe(":8080", handler))
}
