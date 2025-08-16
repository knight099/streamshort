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
	creatorHandler := handlers.NewCreatorHandler(db)
	contentHandler := handlers.NewContentHandler(db)

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

	// Public content routes
	r.HandleFunc("/content/series", contentHandler.ListSeries).Methods("GET")
	r.HandleFunc("/content/series/{id}", contentHandler.GetSeries).Methods("GET")

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

	// Creator routes (protected)
	protected.HandleFunc("/creators/profile", creatorHandler.GetCreatorProfile).Methods("GET")
	protected.HandleFunc("/creators/profile", creatorHandler.UpdateCreatorProfile).Methods("PUT")
	protected.HandleFunc("/creators/onboard", creatorHandler.OnboardCreator).Methods("POST")
	protected.HandleFunc("/creators/{id}/dashboard", creatorHandler.GetCreatorDashboard).Methods("GET")

	// Content routes (protected)
	protected.HandleFunc("/content/series", contentHandler.CreateSeries).Methods("POST")
	protected.HandleFunc("/content/series/{id}", contentHandler.UpdateSeries).Methods("PUT")
	protected.HandleFunc("/content/series/{id}/episodes", contentHandler.CreateEpisode).Methods("POST")
	protected.HandleFunc("/content/upload-url", contentHandler.RequestUploadURL).Methods("POST")
	protected.HandleFunc("/content/uploads/{upload_id}/notify", contentHandler.NotifyUploadComplete).Methods("POST")
	protected.HandleFunc("/episodes/{id}/manifest", contentHandler.GetEpisodeManifest).Methods("GET")

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
	log.Println("  POST /api/creators/onboard     - Creator onboarding (requires auth)")
	log.Println("  GET  /api/creators/profile      - Get creator profile (requires auth)")
	log.Println("  PUT  /api/creators/profile      - Update creator profile (requires auth)")
	log.Println("  GET  /api/creators/{id}/dashboard - Creator dashboard (requires auth)")

	log.Fatal(http.ListenAndServe(":8080", handler))
}
