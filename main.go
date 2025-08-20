package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"streamshort/config"
	"streamshort/handlers"
	"streamshort/middleware"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
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
	// Load environment variables from files for local development
	// .env.local (if present) overrides .env
	if err := godotenv.Load(".env.local"); err == nil {
		log.Println("Loaded environment from .env.local")
	}
	_ = godotenv.Load() // ignore if .env is missing

	// Initialize database
	db := config.InitDB()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db)
	creatorHandler := handlers.NewCreatorHandler(db)
	contentHandler := handlers.NewContentHandler(db)
	paymentHandler := handlers.NewPaymentHandler()
	socialHandler := handlers.NewSocialHandler(db)
	adminHandler := handlers.NewAdminHandler()

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

	// Public content routes (no authentication required)
	r.HandleFunc("/content/series", contentHandler.ListSeries).Methods("GET")
	r.HandleFunc("/content/series/{id}", contentHandler.GetSeries).Methods("GET")

	// Public payment webhook (no authentication required)
	r.HandleFunc("/payments/webhook", paymentHandler.Webhook).Methods("POST")

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

	// Content routes (protected - creators only)
	protected.HandleFunc("/content/series", contentHandler.CreateSeries).Methods("POST")
	protected.HandleFunc("/content/series/{id}", contentHandler.UpdateSeries).Methods("PUT")
	protected.HandleFunc("/content/series/{id}/episodes", contentHandler.CreateEpisode).Methods("POST")
	protected.HandleFunc("/content/upload-url", contentHandler.RequestUploadURL).Methods("POST")
	protected.HandleFunc("/content/uploads/{upload_id}/notify", contentHandler.NotifyUploadComplete).Methods("POST")
	protected.HandleFunc("/episodes/{id}/manifest", contentHandler.GetEpisodeManifest).Methods("GET")

	// Payment routes (protected)
	protected.HandleFunc("/payments/create-subscription", paymentHandler.CreateSubscription).Methods("POST")

	// Social/Engagement routes (protected)
	protected.HandleFunc("/episodes/{id}/like", socialHandler.LikeEpisode).Methods("POST")
	protected.HandleFunc("/episodes/{id}/rating", socialHandler.RateEpisode).Methods("POST")
	protected.HandleFunc("/episodes/{id}/comments", socialHandler.CommentEpisode).Methods("POST")

	// Admin routes (protected - admin only)
	protected.HandleFunc("/admin/uploads/pending", adminHandler.GetPendingUploads).Methods("GET")
	protected.HandleFunc("/admin/approve-content", adminHandler.ApproveContent).Methods("POST")

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	// Apply CORS middleware
	handler := c.Handler(r)

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...", port)
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
	log.Println("  POST /api/content/series        - Create series (creators only)")
	log.Println("  PUT  /api/content/series/{id}   - Update series (creators only)")
	log.Println("  POST /api/content/series/{id}/episodes - Create episode (creators only)")
	log.Println("  POST /api/content/upload-url    - Request upload URL (creators only)")
	log.Println("  POST /api/content/uploads/{id}/notify - Notify upload complete (creators only)")
	log.Println("  GET  /api/episodes/{id}/manifest - Get episode manifest (requires auth)")
	log.Println("  POST /api/payments/create-subscription - Create subscription (requires auth)")
	log.Println("  POST /api/episodes/{id}/like    - Like/unlike episode (requires auth)")
	log.Println("  POST /api/episodes/{id}/rating  - Rate episode (requires auth)")
	log.Println("  POST /api/episodes/{id}/comments - Comment on episode (requires auth)")
	log.Println("  GET  /api/admin/uploads/pending - List pending uploads (admin only)")
	log.Println("  POST /api/admin/approve-content - Approve/reject content (admin only)")
	log.Println("  GET  /content/series            - List series (public)")
	log.Println("  GET  /content/series/{id}       - Get series details (public)")
	log.Println("  POST /payments/webhook          - Payment webhook (public)")

	log.Fatal(http.ListenAndServe(":"+port, handler))
}
