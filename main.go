package main

import (
	"encoding/json"
	"log"
	"net/http"
	"streamshort/config"
	"streamshort/handlers"
	"streamshort/middleware"
	"strings"

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

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize services (DB, Redis, Elasticsearch)
	svcs := config.InitServices()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(svcs.DB, svcs.FirebaseAPIKey)
	creatorHandler := handlers.NewCreatorHandler(svcs.DB)
	contentHandler := handlers.NewContentHandlerWithServices(svcs.DB, svcs.RDB)
	paymentHandler := handlers.NewPaymentHandler(svcs.DB)
	subscriptionHandler := handlers.NewSubscriptionHandler(svcs.DB)
	socialHandler := handlers.NewSocialHandler(svcs.DB)
	userHandler := handlers.NewUserHandler(svcs.DB)
	analyticsHandler := handlers.NewAnalyticsHandler(svcs.DB, config.LoadConfig().CreatorRPMUSDPer1000Min)
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
	r.HandleFunc("/content/series/{seriesId}/episodes", contentHandler.GetEpisodes).Methods("GET")
	r.HandleFunc("/content/series/search", contentHandler.SearchSeries).Methods("GET")

	// Public payment webhook (no authentication required)
	r.HandleFunc("/payments/webhook", paymentHandler.Webhook).Methods("POST")

	// Auth routes (matching OpenAPI schema)
	r.HandleFunc("/auth/otp/send", authHandler.SendOTP).Methods("POST")
	r.HandleFunc("/auth/otp/verify", authHandler.VerifyOTP).Methods("POST")
	r.HandleFunc("/auth/refresh", authHandler.RefreshToken).Methods("POST")
	// Firebase phone auth routes
	r.HandleFunc("/auth/firebase/otp/send", authHandler.FirebaseSendOTP).Methods("POST")
	r.HandleFunc("/auth/firebase/otp/verify", authHandler.FirebaseVerifyOTP).Methods("POST")
	r.HandleFunc("/auth/firebase/exchange", authHandler.FirebaseExchangeIDToken).Methods("POST")
	// Recaptcha site key (for clients that need to render widget)
	r.HandleFunc("/auth/recaptcha/site-key", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"site_key": svcs.RecaptchaSiteKey})
	}).Methods("GET")

	// Protected routes (example)
	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(authMiddleware.AuthMiddleware)
	protected.HandleFunc("/profile", userHandler.GetProfile).Methods("GET")
	protected.HandleFunc("/profile", userHandler.UpdateProfile).Methods("PUT")

	// Creator routes (protected)
	protected.HandleFunc("/creators/profile", creatorHandler.GetCreatorProfile).Methods("GET")
	protected.HandleFunc("/creators/profile", creatorHandler.UpdateCreatorProfile).Methods("PUT")
	protected.HandleFunc("/creators/onboard", creatorHandler.OnboardCreator).Methods("POST")
	protected.HandleFunc("/creators/{id}/dashboard", creatorHandler.GetCreatorDashboard).Methods("GET")
	protected.HandleFunc("/creators/content", contentHandler.GetCreatorContent).Methods("GET")
	// Follow routes
	protected.HandleFunc("/creators/{id}/follow", creatorHandler.FollowCreator).Methods("POST")
	protected.HandleFunc("/creators/{id}/follow", creatorHandler.UnfollowCreator).Methods("DELETE")
	protected.HandleFunc("/creators/{id}/following", creatorHandler.IsFollowing).Methods("GET")
	protected.HandleFunc("/me/following", creatorHandler.ListFollowing).Methods("GET")

	// Content routes (protected - creators only)
	// Analytics
	protected.HandleFunc("/analytics/watch", analyticsHandler.RecordWatch).Methods("POST")
	protected.HandleFunc("/content/series", contentHandler.CreateSeries).Methods("POST")
	protected.HandleFunc("/content/series/{id}", contentHandler.UpdateSeries).Methods("PUT")
	protected.HandleFunc("/content/series/{id}/episodes", contentHandler.CreateEpisode).Methods("POST")
	protected.HandleFunc("/content/upload-url", contentHandler.RequestUploadURL).Methods("POST")
	protected.HandleFunc("/content/uploads/{upload_id}/notify", contentHandler.NotifyUploadComplete).Methods("POST")
	protected.HandleFunc("/episodes/{id}/manifest", contentHandler.GetEpisodeManifest).Methods("GET")
	protected.HandleFunc("/content/episodes/{id}/status", contentHandler.UpdateEpisodeStatus).Methods("PUT")
	protected.HandleFunc("/content/episodes/{id}", contentHandler.UpdateEpisode).Methods("PUT")
	protected.HandleFunc("/content/episodes/{id}", contentHandler.DeleteEpisode).Methods("DELETE")
	protected.HandleFunc("/content/series/{id}/status", contentHandler.UpdateSeriesStatus).Methods("PUT")

	// Payment routes (protected)
	protected.HandleFunc("/payments/create-subscription", paymentHandler.CreateSubscription).Methods("POST")

	// Subscription routes (protected)
	// IMPORTANT: Specific routes must come BEFORE parameterized routes in Gorilla Mux
	protected.HandleFunc("/subscriptions/plans", subscriptionHandler.GetSubscriptionPlans).Methods("GET")
	protected.HandleFunc("/subscriptions/check", subscriptionHandler.CheckSubscriptionStatus).Methods("GET")
	protected.HandleFunc("/subscriptions", subscriptionHandler.GetUserSubscriptions).Methods("GET")
	protected.HandleFunc("/subscriptions/{id}", subscriptionHandler.GetUserSubscription).Methods("GET")
	protected.HandleFunc("/subscriptions/{id}/cancel", subscriptionHandler.CancelUserSubscription).Methods("POST")
	protected.HandleFunc("/subscriptions/{id}/renew", subscriptionHandler.RenewUserSubscription).Methods("POST")

	// Social/Engagement routes (protected)
	protected.HandleFunc("/episodes/{id}/like", socialHandler.LikeEpisode).Methods("POST")
	protected.HandleFunc("/episodes/{id}/rating", socialHandler.RateEpisode).Methods("POST")
	protected.HandleFunc("/episodes/{id}/comments", socialHandler.CommentEpisode).Methods("POST")

	// Admin routes (protected - admin only)
	protected.HandleFunc("/admin/uploads/pending", adminHandler.GetPendingUploads).Methods("GET")
	protected.HandleFunc("/admin/approve-content", adminHandler.ApproveContent).Methods("POST")

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
		AllowOriginFunc: func(origin string) bool {
			return strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "https://localhost:") ||
				strings.HasPrefix(origin, "http://127.0.0.1:") ||
				strings.HasPrefix(origin, "https://127.0.0.1:")
		},
	})

	// Apply CORS middleware
	handler := c.Handler(r)

	// Get port from environment variable or use default
	port := cfg.Port
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
	log.Println("  POST /auth/firebase/otp/send   - Send Firebase OTP")
	log.Println("  POST /auth/firebase/otp/verify - Verify Firebase OTP")
	log.Println("  POST /auth/firebase/exchange   - Exchange Firebase ID token for app tokens")
	log.Println("  GET  /auth/recaptcha/site-key  - Get reCAPTCHA site key")
	log.Println("  GET  /api/profile         - Protected profile (requires auth)")
	log.Println("  POST /api/creators/onboard     - Creator onboarding (requires auth)")
	log.Println("  GET  /api/creators/profile      - Get creator profile (requires auth)")
	log.Println("  PUT  /api/creators/profile      - Update creator profile (requires auth)")
	log.Println("  GET  /api/creators/{id}/dashboard - Creator dashboard (requires auth)")
	log.Println("  GET  /api/creators/content - Get creator content (requires auth)")
	log.Println("  POST /api/content/series        - Create series (creators only)")
	log.Println("  PUT  /api/content/series/{id}   - Update series (creators only)")
	log.Println("  POST /api/content/series/{id}/episodes - Create episode (creators only)")
	log.Println("  POST /api/content/upload-url    - Request upload URL (creators only)")
	log.Println("  POST /api/content/uploads/{id}/notify - Notify upload complete (creators only)")
	log.Println("  GET  /api/episodes/{id}/manifest - Get episode manifest (requires auth)")
	log.Println("  PUT  /api/content/episodes/{id}/status - Update episode status (creators only)")
	log.Println("  PUT  /api/content/episodes/{id}   - Update episode (creators only)")
	log.Println("  DELETE /api/content/episodes/{id} - Delete episode (creators only)")
	log.Println("  PUT  /api/content/series/{id}/status - Update series status (creators only)")
	log.Println("  POST /api/payments/create-subscription - Create subscription (requires auth)")
	log.Println("  GET  /api/subscriptions              - Get user subscriptions (requires auth)")
	log.Println("  GET  /api/subscriptions/{id}         - Get specific subscription (requires auth)")
	log.Println("  POST /api/subscriptions/{id}/cancel  - Cancel subscription (requires auth)")
	log.Println("  POST /api/subscriptions/{id}/renew   - Renew subscription (requires auth)")
	log.Println("  GET  /api/subscriptions/check        - Check subscription status (requires auth)")
	log.Println("  GET  /api/subscriptions/plans        - Get available plans (requires auth)")
	log.Println("  POST /api/episodes/{id}/like    - Like/unlike episode (requires auth)")
	log.Println("  POST /api/episodes/{id}/rating  - Rate episode (requires auth)")
	log.Println("  POST /api/episodes/{id}/comments - Comment on episode (requires auth)")
	log.Println("  GET  /api/admin/uploads/pending - List pending uploads (admin only)")
	log.Println("  POST /api/admin/approve-content - Approve/reject content (admin only)")
	log.Println("  GET  /content/series            - List series (public)")
	log.Println("  GET  /content/series/{id}       - Get series details (public)")
	log.Println("  GET  /content/series/{seriesId}/episodes - Get episodes for series (public)")
	log.Println("  GET  /content/series/search   - Search series (public)")
	log.Println("  POST /payments/webhook          - Payment webhook (public)")

	// Bind to all interfaces (0.0.0.0) for deployment compatibility
	addr := "0.0.0.0:" + port
	log.Printf("Binding to address: %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
