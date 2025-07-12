package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/growthfolio/go-priceguard-api/internal/adapters/http/handlers"
	"github.com/growthfolio/go-priceguard-api/internal/adapters/repository"
	"github.com/growthfolio/go-priceguard-api/internal/application/services"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type APIIntegrationTestSuite struct {
	suite.Suite
	db       *gorm.DB
	router   *gin.Engine
	testUser *entities.User
	ctx      context.Context
}

func (suite *APIIntegrationTestSuite) SetupSuite() {
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

	// Setup repositories
	alertRepo := repository.NewAlertRepository(db)
	userRepo := repository.NewUserRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	priceHistoryRepo := repository.NewPriceHistoryRepository(db)
	technicalIndicatorRepo := repository.NewTechnicalIndicatorRepository(db)

	// Setup logger
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	// Setup Redis client (mock for testing)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Test database
	})

	// Create services
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

	// Setup handlers
	alertHandler := handlers.NewAlertHandler(alertRepo, alertMonitor, alertEngine)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup routes (simplified for testing)
	api := router.Group("/api/v1")
	{
		alerts := api.Group("/alerts")
		{
			alerts.POST("", alertHandler.CreateAlert)
			alerts.GET("", alertHandler.GetAlerts)
			alerts.PUT("/:id", alertHandler.UpdateAlert)
			alerts.DELETE("/:id", alertHandler.DeleteAlert)
		}
	}

	suite.db = db
	suite.router = router
	suite.ctx = context.Background()

	// Create test user
	suite.createTestUser()
}

func (suite *APIIntegrationTestSuite) createTestUser() {
	userRepo := repository.NewUserRepository(suite.db)
	user := &entities.User{
		GoogleID: "test-google-id",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	err := userRepo.Create(suite.ctx, user)
	suite.Require().NoError(err)

	suite.testUser = user
}

func (suite *APIIntegrationTestSuite) TearDownSuite() {
	sqlDB, err := suite.db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

func (suite *APIIntegrationTestSuite) TestCreateAlert_API() {
	// Test creating an alert via API
	alertData := map[string]interface{}{
		"symbol":         "BTCUSDT",
		"alert_type":     "price",
		"condition_type": "above",
		"target_value":   50000.0,
		"timeframe":      "1h",
		"enabled":        true,
		"notify_via":     []string{"app"},
	}

	body, _ := json.Marshal(alertData)
	req, _ := http.NewRequest("POST", "/api/v1/alerts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// In a real implementation, we would add authentication headers here
	req.Header.Set("X-User-ID", suite.testUser.ID.String())

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response, "id")
	assert.Equal(suite.T(), "BTCUSDT", response["symbol"])
}

func (suite *APIIntegrationTestSuite) TestGetUserAlerts_API() {
	// First create an alert
	alertRepo := repository.NewAlertRepository(suite.db)
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

	err := alertRepo.Create(suite.ctx, alert)
	assert.NoError(suite.T(), err)

	// Test getting user alerts via API
	req, _ := http.NewRequest("GET", "/api/v1/alerts", nil)
	req.Header.Set("X-User-ID", suite.testUser.ID.String())

	// Add user_id to context for the test
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", suite.testUser.ID)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response, "alerts")

	alerts, ok := response["alerts"].([]interface{})
	assert.True(suite.T(), ok)
	assert.GreaterOrEqual(suite.T(), len(alerts), 1)
}

func (suite *APIIntegrationTestSuite) TestGetSpecificAlert_API() {
	// First create an alert
	alertRepo := repository.NewAlertRepository(suite.db)
	alert := &entities.Alert{
		UserID:        suite.testUser.ID,
		Symbol:        "ADAUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   1.0,
		Timeframe:     "1h",
		Enabled:       true,
		NotifyVia:     []string{"app"},
	}

	err := alertRepo.Create(suite.ctx, alert)
	assert.NoError(suite.T(), err)

	// Test getting specific alert via API
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/alerts/%s", alert.ID), nil)
	req.Header.Set("X-User-ID", suite.testUser.ID.String())

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response entities.Alert
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), alert.ID, response.ID)
	assert.Equal(suite.T(), "ADAUSDT", response.Symbol)
}

func (suite *APIIntegrationTestSuite) TestUpdateAlert_API() {
	// First create an alert
	alertRepo := repository.NewAlertRepository(suite.db)
	alert := &entities.Alert{
		UserID:        suite.testUser.ID,
		Symbol:        "BNBUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   300.0,
		Timeframe:     "1h",
		Enabled:       true,
		NotifyVia:     []string{"app"},
	}

	err := alertRepo.Create(suite.ctx, alert)
	assert.NoError(suite.T(), err)

	// Test updating the alert via API
	updateData := map[string]interface{}{
		"target_value": 350.0,
		"enabled":      false,
	}

	body, _ := json.Marshal(updateData)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/alerts/%s", alert.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", suite.testUser.ID.String())

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response entities.Alert
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 350.0, response.TargetValue)
	assert.False(suite.T(), response.Enabled)
}

func (suite *APIIntegrationTestSuite) TestDeleteAlert_API() {
	// First create an alert
	alertRepo := repository.NewAlertRepository(suite.db)
	alert := &entities.Alert{
		UserID:        suite.testUser.ID,
		Symbol:        "DOGEUSDT",
		AlertType:     "price",
		ConditionType: "below",
		TargetValue:   0.1,
		Timeframe:     "1h",
		Enabled:       true,
		NotifyVia:     []string{"app"},
	}

	err := alertRepo.Create(suite.ctx, alert)
	assert.NoError(suite.T(), err)

	// Test deleting the alert via API
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/alerts/%s", alert.ID), nil)
	req.Header.Set("X-User-ID", suite.testUser.ID.String())

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNoContent, w.Code)

	// Verify the alert was deleted
	_, err = alertRepo.GetByID(suite.ctx, alert.ID)
	assert.Error(suite.T(), err) // Should return error for not found
}

// Run the API integration test suite
func TestAPIIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(APIIntegrationTestSuite))
}
