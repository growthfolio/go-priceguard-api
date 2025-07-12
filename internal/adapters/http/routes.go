package http

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/felipe-macedo/go-priceguard-api/internal/adapters/http/handlers"
	"github.com/felipe-macedo/go-priceguard-api/internal/adapters/http/middleware"
	"github.com/felipe-macedo/go-priceguard-api/internal/adapters/repository"
	"github.com/felipe-macedo/go-priceguard-api/internal/adapters/websocket"
	appservices "github.com/felipe-macedo/go-priceguard-api/internal/application/services"
	domainservices "github.com/felipe-macedo/go-priceguard-api/internal/domain/services"
	"github.com/felipe-macedo/go-priceguard-api/internal/infrastructure/config"
	"github.com/felipe-macedo/go-priceguard-api/internal/infrastructure/database"
	"github.com/felipe-macedo/go-priceguard-api/internal/infrastructure/external"
	"github.com/sirupsen/logrus"
)

// RouterDependencies holds all dependencies needed for setting up routes
type RouterDependencies struct {
	Config      *config.Config
	Logger      *logrus.Logger
	ZapLogger   *zap.Logger
	DBManager   *database.Manager
	RedisClient *redis.Client
}

// WebSocketManager holds WebSocket-related components
type WebSocketManager struct {
	Hub     *websocket.Hub
	Handler *websocket.WebSocketHandler
	Worker  *websocket.Worker
}

// SetupRoutes configures all API routes and WebSocket endpoints
func SetupRoutes(router *gin.Engine, deps *RouterDependencies) *WebSocketManager {
	// Initialize repositories
	userRepo := repository.NewUserRepository(deps.DBManager.GetDB())
	userSettingsRepo := repository.NewUserSettingsRepository(deps.DBManager.GetDB())
	cryptoRepo := repository.NewCryptoCurrencyRepository(deps.DBManager.GetDB())
	alertRepo := repository.NewAlertRepository(deps.DBManager.GetDB())
	notificationRepo := repository.NewNotificationRepository(deps.DBManager.GetDB())
	priceHistoryRepo := repository.NewPriceHistoryRepository(deps.DBManager.GetDB())
	technicalIndicatorRepo := repository.NewTechnicalIndicatorRepository(deps.DBManager.GetDB())
	sessionRepo := repository.NewSessionRepository(deps.DBManager.GetDB())

	// Initialize domain services
	jwtService := domainservices.NewJWTService(deps.Config.JWT.Secret, deps.Config.JWT.Expiration, deps.Config.JWT.RefreshExpiration)
	googleOAuthService := domainservices.NewGoogleOAuthService(deps.Config.Google.ClientID, deps.Config.Google.ClientSecret, deps.Config.Google.RedirectURL)

	// Initialize application services
	authService := appservices.NewAuthService(
		userRepo,
		sessionRepo,
		userSettingsRepo,
		jwtService,
		googleOAuthService,
		deps.DBManager.GetRedis(),
		deps.Logger,
	)

	// Initialize technical indicator service
	technicalIndicatorService := appservices.NewTechnicalIndicatorService(
		priceHistoryRepo,
		technicalIndicatorRepo,
		deps.Logger,
	)

	// Initialize pullback entry service
	pullbackEntryService := appservices.NewPullbackEntryService(
		priceHistoryRepo,
		technicalIndicatorRepo,
		deps.Logger,
	)

	// Initialize Binance client (for crypto data service)
	binanceClient := external.NewBinanceClient(&deps.Config.Binance, deps.Logger)

	// Initialize crypto data service
	cryptoDataService := appservices.NewCryptoDataService(
		binanceClient,
		cryptoRepo,
		priceHistoryRepo,
		technicalIndicatorRepo,
		deps.Logger,
	)

	// Initialize Alert Engine
	alertEngine := appservices.NewAlertEngine(
		alertRepo,
		priceHistoryRepo,
		technicalIndicatorRepo,
		notificationRepo,
		technicalIndicatorService,
		deps.Logger,
	)

	// Initialize Notification Service
	notificationService := appservices.NewNotificationService(
		notificationRepo,
		userRepo,
		deps.DBManager.GetRedis().GetClient(),
		deps.Logger,
	)

	// Initialize WebSocket components
	wsHub := websocket.NewHub(authService, deps.Logger)

	// Initialize Alert WebSocket Service
	alertWebSocketService := appservices.NewAlertWebSocketService(
		wsHub,
		notificationService,
		alertEngine,
		deps.Logger,
	)

	// Set WebSocket service in alert engine for broadcasting
	alertEngine.SetWebSocketService(alertWebSocketService)

	// Initialize Alert Monitor
	alertMonitor := appservices.NewAlertMonitor(
		alertEngine,
		notificationService,
		cryptoDataService,
		alertRepo,
		deps.Logger,
	)

	// Start services
	ctx := context.Background()
	notificationService.StartProcessing(ctx)
	alertMonitor.Start(ctx)

	wsHandler := websocket.NewWebSocketHandler(wsHub, cryptoDataService, technicalIndicatorService, pullbackEntryService, deps.Logger)
	wsWorker := websocket.NewWorker(
		wsHub,
		wsHandler,
		cryptoDataService,
		technicalIndicatorService,
		pullbackEntryService,
		alertEngine,
		notificationService,
		alertRepo,
		priceHistoryRepo,
		deps.Logger,
	)

	// Start WebSocket hub and worker
	go wsHub.Start()
	go wsWorker.Start(ctx)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService, deps.Logger)

	// Setup global middlewares
	setupGlobalMiddlewares(router, deps)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(deps.DBManager.GetDB(), deps.RedisClient)
	authHandler := handlers.NewAuthHandler(authService, deps.Logger)
	userHandler := handlers.NewUserHandler(userRepo, userSettingsRepo)
	cryptoHandler := handlers.NewCryptoHandler(cryptoRepo, priceHistoryRepo, technicalIndicatorRepo)
	alertHandler := handlers.NewAlertHandler(alertRepo, alertMonitor, alertEngine)
	notificationHandler := handlers.NewNotificationHandler(notificationRepo, notificationService)
	indicatorHandler := handlers.NewIndicatorHandler(technicalIndicatorService, deps.Logger)
	pullbackHandler := handlers.NewPullbackHandler(pullbackEntryService, deps.Logger)

	// Health check routes (no auth required)
	setupHealthRoutes(router, healthHandler)

	// Public routes
	publicAPI := router.Group("/api")
	{
		// Authentication routes
		auth := publicAPI.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authMiddleware.RequireAuth(), authHandler.Logout)
			auth.GET("/verify", authMiddleware.RequireAuth(), authHandler.VerifyToken)
		}
	}

	// Protected routes
	protectedAPI := router.Group("/api")
	protectedAPI.Use(authMiddleware.RequireAuth())
	setupAuthenticatedRateLimit(protectedAPI, deps.RedisClient)
	{
		// User routes
		user := protectedAPI.Group("/user")
		{
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
			user.GET("/settings", userHandler.GetSettings)
			user.PUT("/settings", userHandler.UpdateSettings)
		}

		// Cryptocurrency routes
		crypto := protectedAPI.Group("/crypto")
		{
			crypto.GET("/data", cryptoHandler.GetCryptoData)
			crypto.GET("/detail/:symbol", cryptoHandler.GetCryptoDetail)
			crypto.GET("/history/:symbol", cryptoHandler.GetPriceHistory)
			crypto.GET("/indicators/:symbol", cryptoHandler.GetTechnicalIndicators)
		}

		// Alert routes
		alerts := protectedAPI.Group("/alerts")
		{
			alerts.GET("", alertHandler.GetAlerts)
			alerts.POST("", alertHandler.CreateAlert)
			alerts.PUT("/:id", alertHandler.UpdateAlert)
			alerts.DELETE("/:id", alertHandler.DeleteAlert)
			alerts.GET("/types", alertHandler.GetAlertTypes)
			alerts.GET("/stats", alertHandler.GetAlertStats)
			alerts.POST("/trigger-evaluation", alertHandler.TriggerEvaluation)
			alerts.POST("/:id/evaluate", alertHandler.EvaluateAlert)
		}

		// Notification routes
		notifications := protectedAPI.Group("/notifications")
		{
			notifications.GET("", notificationHandler.GetNotifications)
			notifications.POST("/mark-read", notificationHandler.MarkAsRead)
			notifications.POST("/mark-all-read", notificationHandler.MarkAllAsRead)
			notifications.DELETE("/:id", notificationHandler.DeleteNotification)
			notifications.POST("/test", notificationHandler.CreateTestNotification)
			notifications.GET("/stats", notificationHandler.GetNotificationStats)
		}

		// Technical Indicator routes
		indicators := protectedAPI.Group("/indicators")
		{
			indicators.POST("/:symbol/rsi", indicatorHandler.CalculateRSI)
			indicators.POST("/:symbol/ema", indicatorHandler.CalculateEMA)
			indicators.POST("/:symbol/sma", indicatorHandler.CalculateSMA)
			indicators.POST("/:symbol/supertrend", indicatorHandler.CalculateSuperTrend)
			indicators.POST("/:symbol/all", indicatorHandler.CalculateAllIndicators)
			indicators.GET("/:symbol/latest", indicatorHandler.GetLatestIndicators)
		}

		// Pullback Entry routes
		pullback := protectedAPI.Group("/pullback")
		{
			pullback.GET("/:symbol/analyze", pullbackHandler.AnalyzePullbackEntry)
			pullback.GET("/:symbol/multi", pullbackHandler.GetPullbackEntriesMultiTimeframe)
		}
	}

	// WebSocket routes (with JWT authentication via query parameter)
	router.GET("/ws", wsHandler.HandleConnection)

	return &WebSocketManager{
		Hub:     wsHub,
		Handler: wsHandler,
		Worker:  wsWorker,
	}
}

// setupGlobalMiddlewares configura middlewares globais de segurança e observabilidade
func setupGlobalMiddlewares(router *gin.Engine, deps *RouterDependencies) {
	// Security headers (primeiro)
	router.Use(middleware.SecurityHeadersMiddleware())

	// CORS
	router.Use(middleware.CORSMiddleware())

	// Request ID
	router.Use(middleware.RequestIDMiddleware())

	// Logging (com zap logger se disponível)
	if deps.ZapLogger != nil {
		router.Use(middleware.LoggingMiddleware(deps.ZapLogger))
	}

	// Error handling
	if deps.ZapLogger != nil {
		router.Use(middleware.ErrorHandlingMiddleware(deps.ZapLogger))
	}

	// Compression
	router.Use(middleware.CompressionMiddleware())

	// Rate limiting (se Redis estiver disponível)
	if deps.RedisClient != nil {
		// Rate limit mais permissivo para rotas públicas
		publicRateLimit := middleware.DefaultRateLimitConfig()
		router.Use(middleware.RateLimitMiddleware(deps.RedisClient, publicRateLimit))
	}

	// Input sanitization
	router.Use(middleware.SanitizeInputMiddleware())

	// CSRF protection para rotas que modificam estado
	router.Use(middleware.CSRFProtectionMiddleware())
}

// setupHealthRoutes configura rotas de health check e métricas
func setupHealthRoutes(router *gin.Engine, healthHandler *handlers.HealthHandler) {
	// Health check routes (sem autenticação)
	health := router.Group("/")
	{
		health.GET("/health", healthHandler.Health)
		health.GET("/health/live", healthHandler.Live)
		health.GET("/health/ready", healthHandler.Ready)
		health.GET("/metrics", healthHandler.Metrics)
	}
}

// setupAuthenticatedRateLimit configura rate limiting específico para usuários autenticados
func setupAuthenticatedRateLimit(group *gin.RouterGroup, redisClient *redis.Client) {
	if redisClient != nil {
		authRateLimit := middleware.AuthenticatedUserRateLimitConfig()
		group.Use(middleware.RateLimitMiddleware(redisClient, authRateLimit))
	}
}
