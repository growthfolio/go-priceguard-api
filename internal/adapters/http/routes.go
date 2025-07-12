package http

import (
	"github.com/gin-gonic/gin"

	"github.com/felipe-macedo/go-priceguard-api/internal/adapters/http/handlers"
	"github.com/felipe-macedo/go-priceguard-api/internal/adapters/http/middleware"
	"github.com/felipe-macedo/go-priceguard-api/internal/adapters/repository"
	appservices "github.com/felipe-macedo/go-priceguard-api/internal/application/services"
	domainservices "github.com/felipe-macedo/go-priceguard-api/internal/domain/services"
	"github.com/felipe-macedo/go-priceguard-api/internal/infrastructure/config"
	"github.com/felipe-macedo/go-priceguard-api/internal/infrastructure/database"
	"github.com/sirupsen/logrus"
)

// RouterDependencies holds all dependencies needed for setting up routes
type RouterDependencies struct {
	Config    *config.Config
	Logger    *logrus.Logger
	DBManager *database.Manager
}

// SetupRoutes configures all API routes
func SetupRoutes(router *gin.Engine, deps *RouterDependencies) {
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

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService, deps.Logger)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, deps.Logger)
	userHandler := handlers.NewUserHandler(userRepo, userSettingsRepo)
	cryptoHandler := handlers.NewCryptoHandler(cryptoRepo, priceHistoryRepo, technicalIndicatorRepo)
	alertHandler := handlers.NewAlertHandler(alertRepo)
	notificationHandler := handlers.NewNotificationHandler(notificationRepo)
	indicatorHandler := handlers.NewIndicatorHandler(technicalIndicatorService, deps.Logger)
	pullbackHandler := handlers.NewPullbackHandler(pullbackEntryService, deps.Logger)

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
		}

		// Notification routes
		notifications := protectedAPI.Group("/notifications")
		{
			notifications.GET("", notificationHandler.GetNotifications)
			notifications.POST("/mark-read", notificationHandler.MarkAsRead)
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
}
