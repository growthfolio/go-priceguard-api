package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/growthfolio/go-priceguard-api/internal/adapters/repository"
	"github.com/growthfolio/go-priceguard-api/internal/application/services"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/growthfolio/go-priceguard-api/internal/domain/repositories"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type AlertIntegrationTestSuite struct {
	suite.Suite
	db                     *gorm.DB
	redisClient            *redis.Client
	alertRepo              repositories.AlertRepository
	notificationRepo       repositories.NotificationRepository
	userRepo               repositories.UserRepository
	priceHistoryRepo       repositories.PriceHistoryRepository
	technicalIndicatorRepo repositories.TechnicalIndicatorRepository
	alertEngine            *services.AlertEngine
	notificationService    *services.NotificationService
	alertMonitor           *services.AlertMonitor
	testUser               *entities.User
	ctx                    context.Context
}

func (suite *AlertIntegrationTestSuite) SetupSuite() {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	suite.Require().NoError(err)

	// Auto-migrate all tables
	err = db.AutoMigrate(
		&entities.User{},
		&entities.UserSettings{},
		&entities.Alert{},
		&entities.Notification{},
		&entities.CryptoCurrency{},
		&entities.PriceHistory{},
		&entities.TechnicalIndicator{},
		&entities.Session{},
	)
	suite.Require().NoError(err)

	// Setup Redis client (in-memory would be ideal, but we'll use a test database)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use test database
	})

	// Test Redis connection (skip tests if Redis not available)
	_, err = redisClient.Ping(context.Background()).Result()
	if err != nil {
		suite.T().Skip("Redis not available, skipping integration tests")
	}

	// Setup logger
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	// Setup repositories
	alertRepo := repository.NewAlertRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	userRepo := repository.NewUserRepository(db)
	priceHistoryRepo := repository.NewPriceHistoryRepository(db)
	technicalIndicatorRepo := repository.NewTechnicalIndicatorRepository(db)

	// Setup services
	redisInterface := services.NewRedisClientWrapper(redisClient)
	notificationService := services.NewNotificationService(notificationRepo, userRepo, redisInterface, log)

	alertEngine := services.NewAlertEngine(
		alertRepo,
		priceHistoryRepo,
		technicalIndicatorRepo,
		notificationRepo,
		nil, // TechnicalIndicatorService
		log,
	)

	alertMonitor := services.NewAlertMonitor(
		alertEngine,
		notificationService,
		nil, // CryptoDataService
		alertRepo,
		log,
	)

	suite.db = db
	suite.redisClient = redisClient
	suite.alertRepo = alertRepo
	suite.notificationRepo = notificationRepo
	suite.userRepo = userRepo
	suite.priceHistoryRepo = priceHistoryRepo
	suite.technicalIndicatorRepo = technicalIndicatorRepo
	suite.alertEngine = alertEngine
	suite.notificationService = notificationService
	suite.alertMonitor = alertMonitor
	suite.ctx = context.Background()

	// Create test user
	suite.createTestUser()
}

func (suite *AlertIntegrationTestSuite) createTestUser() {
	user := &entities.User{
		GoogleID: "test-google-id",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	err := suite.userRepo.Create(suite.ctx, user)
	suite.Require().NoError(err)

	suite.testUser = user
}

func (suite *AlertIntegrationTestSuite) TearDownTest() {
	// Clean up data after each test
	suite.db.Exec("DELETE FROM alerts")
	suite.db.Exec("DELETE FROM notifications")
	suite.db.Exec("DELETE FROM price_history")
	suite.db.Exec("DELETE FROM technical_indicators")

	// Clean Redis
	suite.redisClient.FlushDB(suite.ctx)
}

func (suite *AlertIntegrationTestSuite) TearDownSuite() {
	// Close connections
	if suite.redisClient != nil {
		suite.redisClient.Close()
	}

	sqlDB, err := suite.db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

func (suite *AlertIntegrationTestSuite) TestAlertCreationAndEvaluation_Integration() {
	// Test the complete flow from alert creation to evaluation

	// 1. Create an alert
	alert := &entities.Alert{
		UserID:        suite.testUser.ID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
		NotifyVia:     []string{"app"},
	}

	err := suite.alertRepo.Create(suite.ctx, alert)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), alert.ID)

	// 2. Create price history that should trigger the alert
	priceHistory := &entities.PriceHistory{
		Symbol:     "BTCUSDT",
		Timeframe:  "1h",
		OpenPrice:  49000.0,
		HighPrice:  51000.0,
		LowPrice:   48000.0,
		ClosePrice: 51000.0, // Above target of 50000
		Volume:     1000000.0,
		Timestamp:  time.Now(),
	}

	err = suite.priceHistoryRepo.Create(suite.ctx, priceHistory)
	assert.NoError(suite.T(), err)

	// 3. Evaluate the alert using AlertEngine
	result, err := suite.alertEngine.EvaluateAlert(suite.ctx, alert)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.ShouldTrigger)
	assert.Equal(suite.T(), 51000.0, result.CurrentValue)
	assert.Equal(suite.T(), 50000.0, result.TargetValue)

	// 4. Verify that alert was marked as triggered
	updatedAlert, err := suite.alertRepo.GetByID(suite.ctx, alert.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), updatedAlert.TriggeredAt)

	// 5. Verify that notification was created
	notifications, err := suite.notificationRepo.GetByUserID(suite.ctx, suite.testUser.ID, 10, 0)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), notifications, 1)
	assert.Equal(suite.T(), "alert_triggered", notifications[0].NotificationType)
	assert.Equal(suite.T(), alert.ID, *notifications[0].AlertID)
}

func (suite *AlertIntegrationTestSuite) TestAlertMonitorLifecycle_Integration() {
	// Test AlertMonitor start/stop and basic functionality

	// 1. Create an enabled alert
	alert := &entities.Alert{
		UserID:        suite.testUser.ID,
		Symbol:        "ETHUSDT",
		AlertType:     "price",
		ConditionType: "below",
		TargetValue:   3000.0,
		Timeframe:     "1h",
		Enabled:       true,
		NotifyVia:     []string{"app"},
	}

	err := suite.alertRepo.Create(suite.ctx, alert)
	assert.NoError(suite.T(), err)

	// 2. Start AlertMonitor
	assert.False(suite.T(), suite.alertMonitor.IsRunning())
	suite.alertMonitor.Start(suite.ctx)
	assert.True(suite.T(), suite.alertMonitor.IsRunning())

	// 3. Give it a moment to run
	time.Sleep(100 * time.Millisecond)

	// 4. Stop AlertMonitor
	suite.alertMonitor.Stop()
	assert.False(suite.T(), suite.alertMonitor.IsRunning())

	// 5. Verify it can be restarted
	suite.alertMonitor.Start(suite.ctx)
	assert.True(suite.T(), suite.alertMonitor.IsRunning())
	suite.alertMonitor.Stop()
	assert.False(suite.T(), suite.alertMonitor.IsRunning())
}

func (suite *AlertIntegrationTestSuite) TestNotificationService_Integration() {
	// Test NotificationService with Redis integration

	// 1. Create a notification using the service
	notification, err := suite.notificationService.CreateNotification(
		suite.ctx,
		suite.testUser.ID,
		"system",
		"Test Notification",
		"This is a test notification",
		map[string]interface{}{"test": true},
	)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), notification.ID)

	// 2. Verify notification was saved to database
	savedNotification, err := suite.notificationRepo.GetByID(suite.ctx, notification.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Test Notification", savedNotification.Title)
	assert.Nil(suite.T(), savedNotification.ReadAt)

	// 3. Test notification queuing
	queuedNotification := &services.QueuedNotification{
		ID:          notification.ID,
		UserID:      suite.testUser.ID,
		Type:        "system",
		Title:       "Test Notification",
		Message:     "This is a test notification",
		Channels:    []services.NotificationChannel{services.ChannelInApp},
		Priority:    services.PriorityNormal,
		Data:        map[string]interface{}{"test": true},
		ScheduledAt: time.Now(),
		CreatedAt:   time.Now(),
		MaxRetries:  3,
	}

	err = suite.notificationService.QueueNotification(suite.ctx, queuedNotification)
	assert.NoError(suite.T(), err)

	// 4. Test getting statistics
	stats, err := suite.notificationService.GetNotificationStats(suite.ctx)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), stats)
	assert.Contains(suite.T(), stats, "total_queued")
}

func (suite *AlertIntegrationTestSuite) TestMultipleAlertsEvaluation_Integration() {
	// Test evaluation of multiple alerts simultaneously

	// 1. Create multiple alerts
	alerts := []*entities.Alert{
		{
			UserID:        suite.testUser.ID,
			Symbol:        "BTCUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   50000.0,
			Timeframe:     "1h",
			Enabled:       true,
			NotifyVia:     []string{"app"},
		},
		{
			UserID:        suite.testUser.ID,
			Symbol:        "ETHUSDT",
			AlertType:     "price",
			ConditionType: "below",
			TargetValue:   3000.0,
			Timeframe:     "1h",
			Enabled:       true,
			NotifyVia:     []string{"app"},
		},
		{
			UserID:        suite.testUser.ID,
			Symbol:        "ADAUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   1.0,
			Timeframe:     "1h",
			Enabled:       false, // Disabled
			NotifyVia:     []string{"app"},
		},
	}

	for _, alert := range alerts {
		err := suite.alertRepo.Create(suite.ctx, alert)
		assert.NoError(suite.T(), err)
	}

	// 2. Create price history for all symbols
	priceHistories := []*entities.PriceHistory{
		{
			Symbol:     "BTCUSDT",
			Timeframe:  "1h",
			ClosePrice: 51000.0, // Should trigger (above 50000)
			Timestamp:  time.Now(),
		},
		{
			Symbol:     "ETHUSDT",
			Timeframe:  "1h",
			ClosePrice: 2800.0, // Should trigger (below 3000)
			Timestamp:  time.Now(),
		},
		{
			Symbol:     "ADAUSDT",
			Timeframe:  "1h",
			ClosePrice: 1.5, // Would trigger but alert is disabled
			Timestamp:  time.Now(),
		},
	}

	for _, ph := range priceHistories {
		err := suite.priceHistoryRepo.Create(suite.ctx, ph)
		assert.NoError(suite.T(), err)
	}

	// 3. Evaluate all alerts
	results, err := suite.alertEngine.EvaluateAllAlerts(suite.ctx)
	assert.NoError(suite.T(), err)

	// Should only return results for enabled alerts (2)
	assert.Len(suite.T(), results, 2)

	// Both enabled alerts should trigger
	triggeredCount := 0
	for _, result := range results {
		if result.ShouldTrigger {
			triggeredCount++
		}
	}
	assert.Equal(suite.T(), 2, triggeredCount)

	// 4. Verify notifications were created for triggered alerts
	notifications, err := suite.notificationRepo.GetByUserID(suite.ctx, suite.testUser.ID, 10, 0)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), notifications, 2) // Only for enabled alerts
}

// Run the integration test suite
func TestAlertIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AlertIntegrationTestSuite))
}
