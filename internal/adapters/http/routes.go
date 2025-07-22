package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/growthfolio/go-priceguard-api/internal/adapters/http/handlers"
	"github.com/growthfolio/go-priceguard-api/internal/adapters/http/middleware"
	"github.com/growthfolio/go-priceguard-api/internal/adapters/repository"
	"github.com/growthfolio/go-priceguard-api/internal/adapters/websocket"
	appservices "github.com/growthfolio/go-priceguard-api/internal/application/services"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	domainservices "github.com/growthfolio/go-priceguard-api/internal/domain/services"
	"github.com/growthfolio/go-priceguard-api/internal/infrastructure"
	"github.com/growthfolio/go-priceguard-api/internal/infrastructure/config"
	"github.com/growthfolio/go-priceguard-api/internal/infrastructure/database"
	"github.com/growthfolio/go-priceguard-api/internal/infrastructure/external"
	"github.com/sirupsen/logrus"
)

// RouterDependencies holds all dependencies needed for setting up routes
type RouterDependencies struct {
	Config         *config.Config
	Logger         *logrus.Logger
	ZapLogger      *zap.Logger
	DBManager      *database.Manager
	RedisClient    *redis.Client
	TracingManager *infrastructure.TracingManager
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
	metricsHandler := handlers.NewMetricsHandler(deps.DBManager.GetDB(), deps.RedisClient, deps.ZapLogger)
	authHandler := handlers.NewAuthHandler(authService, deps.Logger)
	userHandler := handlers.NewUserHandler(userRepo, userSettingsRepo)
	cryptoHandler := handlers.NewCryptoHandler(cryptoRepo, priceHistoryRepo, technicalIndicatorRepo)
	alertHandler := handlers.NewAlertHandler(alertRepo, alertMonitor, alertEngine)
	notificationHandler := handlers.NewNotificationHandler(notificationRepo, notificationService)
	indicatorHandler := handlers.NewIndicatorHandler(technicalIndicatorService, deps.Logger)
	pullbackHandler := handlers.NewPullbackHandler(pullbackEntryService, deps.Logger)

	// Health check routes (no auth required)
	setupHealthRoutes(router, healthHandler, metricsHandler)

	// Public test routes for Binance integration
	publicTest := router.Group("/test")
	{
		publicTest.GET("/binance/ping", func(c *gin.Context) {
			// Test Binance connectivity
			ticker, err := binanceClient.GetTickerPrice(c.Request.Context(), "BTCUSDT")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Failed to connect to Binance",
					"details": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status":    "success",
				"message":   "Binance connection OK",
				"btc_price": ticker.Price,
				"timestamp": time.Now().UTC(),
			})
		})

		publicTest.GET("/binance/symbols", func(c *gin.Context) {
			// Test getting exchange info
			cryptos, err := cryptoRepo.GetActive(c.Request.Context(), 10, 0)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Failed to get cryptocurrencies from database",
					"details": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status":  "success",
				"count":   len(cryptos),
				"symbols": cryptos,
			})
		})

		publicTest.GET("/crypto/live-prices", func(c *gin.Context) {
			// Get a few symbols and their live prices from Binance
			symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT"}
			prices := make(map[string]interface{})

			for _, symbol := range symbols {
				ticker, err := binanceClient.GetTickerPrice(c.Request.Context(), symbol)
				if err != nil {
					prices[symbol] = gin.H{
						"error":   "Failed to get price",
						"details": err.Error(),
					}
					continue
				}
				prices[symbol] = gin.H{
					"price":  ticker.Price,
					"symbol": ticker.Symbol,
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"status":    "success",
				"message":   "Live prices from Binance",
				"data":      prices,
				"timestamp": time.Now().UTC(),
			})
		})

		publicTest.GET("/crypto/start-collection", func(c *gin.Context) {
			// Test starting the crypto data collection service
			if cryptoDataService.IsCollecting() {
				c.JSON(http.StatusOK, gin.H{
					"status":     "info",
					"message":    "Crypto data collection is already running",
					"collecting": true,
				})
				return
			}

			err := cryptoDataService.StartDataCollection(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Failed to start data collection",
					"details": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status":     "success",
				"message":    "Crypto data collection started",
				"collecting": cryptoDataService.IsCollecting(),
			})
		})

		publicTest.GET("/crypto/collection-status", func(c *gin.Context) {
			// Check if data collection is running
			c.JSON(http.StatusOK, gin.H{
				"status":     "success",
				"collecting": cryptoDataService.IsCollecting(),
				"message":    "Data collection status",
			})
		})

		publicTest.GET("/crypto/collect-single", func(c *gin.Context) {
			// Collect data for a single cryptocurrency and store it
			symbol := c.DefaultQuery("symbol", "BTCUSDT")

			// Get ticker price from Binance
			ticker, err := binanceClient.GetTickerPrice(c.Request.Context(), symbol)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Failed to get price from Binance",
					"details": err.Error(),
				})
				return
			}

			// Parse price
			price, err := strconv.ParseFloat(ticker.Price, 64)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Failed to parse price",
					"details": err.Error(),
				})
				return
			}

			// Create price history record
			priceHistory := &entities.PriceHistory{
				Symbol:     symbol,
				Timeframe:  "1m",
				Timestamp:  time.Now(),
				OpenPrice:  price,
				HighPrice:  price,
				LowPrice:   price,
				ClosePrice: price,
				Volume:     0.0, // We don't have volume from ticker
			}

			// Store in database
			if err := priceHistoryRepo.Create(c.Request.Context(), priceHistory); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Failed to store price data",
					"details": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status":  "success",
				"message": "Price data collected and stored",
				"data": gin.H{
					"symbol":    symbol,
					"price":     price,
					"timestamp": priceHistory.Timestamp,
				},
			})
		})
	}

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
func setupGlobalMiddlewares(router *gin.Engine, deps *RouterDependencies) { // Security headers (primeiro)
	router.Use(middleware.SecurityHeadersMiddleware())

	// CORS
	router.Use(middleware.CORSMiddleware())

	// Request ID (antes do tracing para incluir nos spans)
	router.Use(middleware.RequestIDMiddleware())

	// Distributed Tracing (se habilitado)
	if deps.TracingManager != nil {
		router.Use(middleware.CustomTracingMiddleware("priceguard-api"))
	}

	// Prometheus metrics
	router.Use(middleware.PrometheusMiddleware())

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
func setupHealthRoutes(router *gin.Engine, healthHandler *handlers.HealthHandler, metricsHandler *handlers.MetricsHandler) {
	// Health check routes (sem autenticação)
	health := router.Group("/")
	{
		health.GET("/health", healthHandler.Health)
		health.GET("/health/live", healthHandler.Live)
		health.GET("/health/ready", healthHandler.Ready)
		health.GET("/metrics", healthHandler.Metrics)
	}

	// Métricas Prometheus (endpoint padrão)
	router.GET("/prometheus", metricsHandler.PrometheusMetrics())

	// Métricas customizadas
	metrics := router.Group("/api/metrics")
	{
		metrics.GET("/system", metricsHandler.SystemInfo)
		metrics.GET("/custom", metricsHandler.CustomMetrics)
		metrics.GET("/application", metricsHandler.ApplicationMetrics)
	}
}

// setupAuthenticatedRateLimit configura rate limiting específico para usuários autenticados
func setupAuthenticatedRateLimit(group *gin.RouterGroup, redisClient *redis.Client) {
	if redisClient != nil {
		authRateLimit := middleware.AuthenticatedUserRateLimitConfig()
		group.Use(middleware.RateLimitMiddleware(redisClient, authRateLimit))
	}
}
