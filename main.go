// @title StreamShort API
// @version 1.0
// @description API for StreamShort, a short-form video streaming platform.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"streamshort/config"
	"streamshort/handlers"
	"streamshort/middleware"
	"streamshort/services"
	"strings"

	_ "streamshort/docs"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
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

	// Initialize AWS service
	awsService, err := services.NewAWSService(cfg)
	if err != nil {
		log.Printf("Warning: AWS service initialization failed: %v", err)
		log.Println("S3 uploads and CloudFront signed URLs will use mock responses")
	} else if awsService != nil {
		log.Println("AWS service initialized successfully")
		if awsService.IsCloudFrontConfigured() {
			log.Println("CloudFront signing is configured")
		} else {
			log.Println("CloudFront signing not configured - using mock URLs for manifests")
		}
	} else {
		log.Println("AWS credentials not configured - using mock responses for S3 and CloudFront")
	}

	// Initialize Razorpay client
	razorpayClient := services.NewRazorpayClient(cfg.RazorpayKeyID, cfg.RazorpayKeySecret)
	if cfg.RazorpayKeyID == "" || cfg.RazorpayKeySecret == "" {
		log.Println("Warning: Razorpay credentials not configured - payment features will be limited")
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(svcs.DB, svcs.FirebaseAPIKey)
	authHandler.EnsureDefaultAdmin() // Seed default admin if needed
	creatorHandler := handlers.NewCreatorHandler(svcs.DB)
	contentHandler := handlers.NewContentHandlerWithServices(svcs.DB, svcs.RDB, awsService)
	paymentHandler := handlers.NewPaymentHandler(svcs.DB, razorpayClient)
	subscriptionHandler := handlers.NewSubscriptionHandler(svcs.DB)
	socialHandler := handlers.NewSocialHandler(svcs.DB)
	userHandler := handlers.NewUserHandler(svcs.DB)
	analyticsHandler := handlers.NewAnalyticsHandler(svcs.DB)
	adminHandler := handlers.NewAdminHandler(svcs.DB, awsService)
	creatorPaymentHandler := handlers.NewCreatorPaymentHandler(svcs.DB, razorpayClient)
	watchListHandler := handlers.NewWatchListHandler(svcs.DB)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware()

	// Create router
	r := mux.NewRouter()

	// Add Request Logger Middleware
	r.Use(middleware.RequestLoggerMiddleware)

	// Public routes
	r.HandleFunc("/", helloHandler).Methods("GET")
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Message: "Server is running!"})
	}).Methods("GET")

	// Swagger documentation
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Public content routes (no authentication required)
	// IMPORTANT: Specific routes must come BEFORE parameterized routes in Gorilla Mux
	r.HandleFunc("/content/series", contentHandler.ListSeries).Methods("GET")
	r.HandleFunc("/content/series/search", contentHandler.SearchSeries).Methods("GET")
	r.HandleFunc("/content/series/{id}", contentHandler.GetSeries).Methods("GET")
	r.HandleFunc("/content/series/{seriesId}/episodes", contentHandler.GetEpisodes).Methods("GET")

	// Public payment webhooks (no authentication required)
	r.HandleFunc("/payments/webhook", paymentHandler.Webhook).Methods("POST")
	r.HandleFunc("/payments/payout-webhook", creatorPaymentHandler.HandlePayoutWebhook).Methods("POST")

	// Analytics route moved to protected to enable user tracking
	// (Was public but needs auth for deduplication)

	// Auth routes (matching OpenAPI schema)
	r.HandleFunc("/auth/otp/send", authHandler.SendOTP).Methods("POST")
	r.HandleFunc("/auth/otp/verify", authHandler.VerifyOTP).Methods("POST")
	r.HandleFunc("/auth/refresh", authHandler.RefreshToken).Methods("POST")
	r.HandleFunc("/auth/admin/login", authHandler.AdminLogin).Methods("POST")
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
	protected.HandleFunc("/me/watch-history", userHandler.GetWatchHistory).Methods("GET")
	protected.HandleFunc("/me/watchlist", watchListHandler.GetWatchList).Methods("GET")
	protected.HandleFunc("/me/watchlist", watchListHandler.AddToWatchList).Methods("POST")
	protected.HandleFunc("/me/watchlist/{series_id}", watchListHandler.RemoveFromWatchList).Methods("DELETE")
	protected.HandleFunc("/me/watchlist/{series_id}/status", watchListHandler.CheckWatchListStatus).Methods("GET")
	protected.HandleFunc("/me/liked-episodes", socialHandler.GetLikedEpisodes).Methods("GET")

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

	// Creator payment/earnings routes
	protected.HandleFunc("/creators/{creator_id}/earnings", creatorPaymentHandler.GetCreatorEarnings).Methods("GET")
	protected.HandleFunc("/creators/{creator_id}/earnings/breakdown", creatorPaymentHandler.GetEarningsBreakdown).Methods("GET")
	protected.HandleFunc("/creators/{creator_id}/payouts", creatorPaymentHandler.GetPayoutHistory).Methods("GET")
	protected.HandleFunc("/creators/{creator_id}/payouts/request", creatorPaymentHandler.RequestPayout).Methods("POST")
	protected.HandleFunc("/creators/{creator_id}/payout-details", creatorPaymentHandler.GetPayoutDetails).Methods("GET")
	protected.HandleFunc("/creators/{creator_id}/payout-details", creatorPaymentHandler.UpdatePayoutDetails).Methods("PUT")

	// Series view tracking
	protected.HandleFunc("/series/{series_id}/view", creatorPaymentHandler.TrackSeriesView).Methods("POST")

	// Analytics route (protected for user tracking)
	protected.HandleFunc("/analytics/watch", analyticsHandler.RecordWatch).Methods("POST")

	// Content routes (protected - creators only)
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
	protected.HandleFunc("/episodes/{id}/like", socialHandler.GetLikeStatus).Methods("GET")
	protected.HandleFunc("/episodes/{id}/like", socialHandler.LikeEpisode).Methods("POST")
	protected.HandleFunc("/series/{id}/rating", socialHandler.RateSeries).Methods("POST")

	// Admin routes (protected - admin only)
	adminProtected := protected.PathPrefix("/admin").Subrouter()
	adminProtected.Use(authMiddleware.AdminMiddleware)
	adminProtected.HandleFunc("/dashboard/stats", adminHandler.AdminDashboardStats).Methods("GET")
	adminProtected.HandleFunc("/users", adminHandler.AdminListUsers).Methods("GET")
	adminProtected.HandleFunc("/create-admin", authHandler.CreateAdmin).Methods("POST")
	adminProtected.HandleFunc("/users/{id}", adminHandler.AdminDeleteUser).Methods("DELETE")
	adminProtected.HandleFunc("/creators/{id}", adminHandler.AdminDeleteCreator).Methods("DELETE")
	adminProtected.HandleFunc("/content/series/{id}", adminHandler.AdminDeleteSeries).Methods("DELETE")
	adminProtected.HandleFunc("/content/episodes/{id}", adminHandler.AdminDeleteEpisode).Methods("DELETE")

	// Admin Content Management
	adminProtected.HandleFunc("/creators", adminHandler.AdminListCreators).Methods("GET")
	adminProtected.HandleFunc("/content/series", adminHandler.AdminCreateSeries).Methods("POST")
	adminProtected.HandleFunc("/content/series", adminHandler.AdminListAllSeries).Methods("GET")
	adminProtected.HandleFunc("/content/series/{id}/episodes", adminHandler.AdminCreateEpisode).Methods("POST")
	adminProtected.HandleFunc("/content/upload-url", adminHandler.AdminRequestUploadURL).Methods("POST")
	adminProtected.HandleFunc("/content/uploads/{id}/notify", adminHandler.AdminNotifyUploadComplete).Methods("POST")

	adminProtected.HandleFunc("/uploads/pending", adminHandler.GetPendingUploads).Methods("GET")
	adminProtected.HandleFunc("/approve-content", adminHandler.ApproveContent).Methods("POST")

	// Admin Finance & Subscription Tracking
	adminProtected.HandleFunc("/subscriptions", adminHandler.AdminListSubscriptions).Methods("GET")
	adminProtected.HandleFunc("/payouts", adminHandler.AdminListPayouts).Methods("GET")
	adminProtected.HandleFunc("/payouts/{id}/status", adminHandler.AdminUpdatePayoutStatus).Methods("PUT")
	adminProtected.HandleFunc("/earnings", adminHandler.AdminListEarnings).Methods("GET")
	adminProtected.HandleFunc("/revenue-summary", adminHandler.AdminGetRevenueSummary).Methods("GET")

	// Admin KYC Verification
	adminProtected.HandleFunc("/kyc", adminHandler.AdminListPendingKYC).Methods("GET")
	adminProtected.HandleFunc("/kyc/{id}/verify", adminHandler.AdminVerifyKYC).Methods("POST")

	// Admin Episodes Management
	adminProtected.HandleFunc("/content/series/{id}/episodes", adminHandler.AdminGetEpisodes).Methods("GET")

	// CORS configuration
	allowedOrigins := strings.Split(cfg.CORSAllowedOrigins, ",")
	allowedOriginsMap := make(map[string]bool)
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowedOriginsMap[origin] = true
		}
	}

	c := cors.New(cors.Options{
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
		AllowOriginFunc: func(origin string) bool {
			// Check exact match first
			if allowedOriginsMap[origin] {
				return true
			}
			// Fallback: allow all localhost/127.0.0.1 for development
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
	log.Println("  GET  /api/admin/content/series/{id}/episodes - Get episodes for series (admin only)")
	log.Println("  GET  /content/series            - List series (public)")
	log.Println("  GET  /content/series/{id}       - Get series details (public)")
	log.Println("  GET  /content/series/{seriesId}/episodes - Get episodes for series (public)")
	log.Println("  GET  /content/series/search   - Search series (public)")
	log.Println("  POST /payments/webhook          - Payment webhook (public)")
	log.Println("  POST /payments/payout-webhook   - Payout webhook (public)")
	log.Println("  GET  /api/creators/{id}/earnings         - Get creator earnings (requires auth)")
	log.Println("  GET  /api/creators/{id}/earnings/breakdown - Earnings breakdown (requires auth)")
	log.Println("  GET  /api/creators/{id}/payouts          - Get payout history (requires auth)")
	log.Println("  POST /api/creators/{id}/payouts/request  - Request payout (requires auth)")
	log.Println("  PUT  /api/creators/{id}/payout-details   - Update payout details (requires auth)")
	log.Println("  POST /api/series/{id}/view               - Track series view (requires auth)")
	log.Println("  GET  /api/me/watchlist                   - Get watchlist (requires auth)")
	log.Println("  POST /api/me/watchlist                   - Add to watchlist (requires auth)")
	log.Println("  DELETE /api/me/watchlist/{series_id}     - Remove from watchlist (requires auth)")
	log.Println("  GET  /api/me/watchlist/{series_id}/status - Check watchlist status (requires auth)")

	// Bind to all interfaces (0.0.0.0) for deployment compatibility
	addr := "0.0.0.0:" + port
	log.Printf("Binding to address: %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
