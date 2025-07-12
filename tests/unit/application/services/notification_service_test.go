package services_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/growthfolio/go-priceguard-api/internal/application/services"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/growthfolio/go-priceguard-api/tests/testutils"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type NotificationServiceTestSuite struct {
	suite.Suite
	notificationService  *services.NotificationService
	mockNotificationRepo *testutils.MockNotificationRepository
	mockUserRepo         *testutils.MockUserRepository
	mockRedisClient      *testutils.MockRedisClient
	logger               *logrus.Logger
	ctx                  context.Context
}

func (suite *NotificationServiceTestSuite) SetupTest() {
	suite.mockNotificationRepo = &testutils.MockNotificationRepository{}
	suite.mockUserRepo = &testutils.MockUserRepository{}
	suite.mockRedisClient = &testutils.MockRedisClient{}
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	suite.ctx = context.Background()

	suite.notificationService = services.NewNotificationService(
		suite.mockNotificationRepo,
		suite.mockUserRepo,
		suite.mockRedisClient,
		suite.logger,
	)
}

func (suite *NotificationServiceTestSuite) TearDownTest() {
	suite.mockNotificationRepo.AssertExpectations(suite.T())
	suite.mockUserRepo.AssertExpectations(suite.T())
	suite.mockRedisClient.AssertExpectations(suite.T())
}

func (suite *NotificationServiceTestSuite) TestNewNotificationService() {
	// Test that NewNotificationService creates a valid instance
	assert.NotNil(suite.T(), suite.notificationService)
}

func (suite *NotificationServiceTestSuite) TestCreateNotification_Success() {
	// Setup
	userID := uuid.New()
	notificationType := "test_type"
	title := "Test Title"
	message := "Test Message"
	data := map[string]interface{}{
		"key": "value",
	}

	suite.mockNotificationRepo.On("Create", suite.ctx, mock.AnythingOfType("*entities.Notification")).Return(nil)

	// Execute
	notification, err := suite.notificationService.CreateNotification(suite.ctx, userID, notificationType, title, message, data)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), notification)
	assert.Equal(suite.T(), userID, notification.UserID)
	assert.Equal(suite.T(), notificationType, notification.NotificationType)
	assert.Equal(suite.T(), title, notification.Title)
	assert.Equal(suite.T(), message, notification.Message)
	assert.False(suite.T(), notification.CreatedAt.IsZero())
}

func (suite *NotificationServiceTestSuite) TestCreateNotification_RepositoryError() {
	// Setup
	userID := uuid.New()
	notificationType := "test_type"
	title := "Test Title"
	message := "Test Message"
	data := map[string]interface{}{"key": "value"}

	expectedError := assert.AnError
	suite.mockNotificationRepo.On("Create", suite.ctx, mock.AnythingOfType("*entities.Notification")).Return(expectedError)

	// Execute
	notification, err := suite.notificationService.CreateNotification(suite.ctx, userID, notificationType, title, message, data)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), notification)
	assert.Contains(suite.T(), err.Error(), "failed to create notification")
}

func (suite *NotificationServiceTestSuite) TestQueueNotification_Success() {
	// Setup
	queuedNotification := &services.QueuedNotification{
		ID:       uuid.New(),
		UserID:   uuid.New(),
		Type:     "alert_triggered",
		Title:    "Alert Triggered",
		Message:  "Your alert has been triggered",
		Channels: []services.NotificationChannel{services.ChannelEmail},
		Priority: services.PriorityHigh,
		Data:     map[string]interface{}{"alert_id": uuid.New()},
	}

	// Mock Redis ZAdd operation
	suite.mockRedisClient.On("ZAdd", suite.ctx, "notification_queue", mock.AnythingOfType("redis.Z")).Return(&redis.IntCmd{})

	// Execute
	err := suite.notificationService.QueueNotification(suite.ctx, queuedNotification)

	// Assert
	assert.NoError(suite.T(), err)
}

func (suite *NotificationServiceTestSuite) TestQueueAlertNotification_Success() {
	// Setup
	userID := uuid.New()
	alert := &entities.Alert{
		ID:            uuid.New(),
		UserID:        userID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		NotifyVia:     []string{"app", "email"},
		Enabled:       true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	currentValue := 51000.0
	channels := []services.NotificationChannel{services.ChannelInApp, services.ChannelEmail}

	// Mock user repository
	user := &entities.User{
		ID:       userID,
		GoogleID: "google123",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	suite.mockUserRepo.On("GetByID", suite.ctx, userID).Return(user, nil)

	// Mock notification creation
	suite.mockNotificationRepo.On("Create", suite.ctx, mock.AnythingOfType("*entities.Notification")).Return(nil)

	// Mock Redis operations for queuing
	suite.mockRedisClient.On("ZAdd", suite.ctx, "notification_queue", mock.AnythingOfType("redis.Z")).Return(&redis.IntCmd{})

	// Execute
	err := suite.notificationService.QueueAlertNotification(suite.ctx, alert, currentValue, channels)

	// Assert
	assert.NoError(suite.T(), err)
}

func (suite *NotificationServiceTestSuite) TestQueueAlertNotification_UserNotFound() {
	// Setup
	userID := uuid.New()
	alert := &entities.Alert{
		ID:            uuid.New(),
		UserID:        userID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		NotifyVia:     []string{"app"},
		Enabled:       true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	currentValue := 51000.0
	channels := []services.NotificationChannel{services.ChannelInApp}

	// Mock user repository error
	expectedError := assert.AnError
	suite.mockUserRepo.On("GetByID", suite.ctx, userID).Return((*entities.User)(nil), expectedError)

	// Execute
	err := suite.notificationService.QueueAlertNotification(suite.ctx, alert, currentValue, channels)

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to get user")
}

func (suite *NotificationServiceTestSuite) TestStartStopProcessing() {
	// Test starting processing
	suite.notificationService.StartProcessing(suite.ctx)

	// Add a small delay to allow goroutine to start
	time.Sleep(10 * time.Millisecond)

	// Test stopping processing
	suite.notificationService.StopProcessing()

	// No assertions needed, just verify no panics occur
	assert.True(suite.T(), true) // Placeholder assertion
}

func TestNotificationServiceTestSuite(t *testing.T) {
	suite.Run(t, new(NotificationServiceTestSuite))
}

// Additional unit tests for specific methods
func TestNotificationService_GetNotificationStats(t *testing.T) {
	mockNotificationRepo := &testutils.MockNotificationRepository{}
	mockUserRepo := &testutils.MockUserRepository{}
	mockRedisClient := &testutils.MockRedisClient{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ctx := context.Background()

	notificationService := services.NewNotificationService(
		mockNotificationRepo,
		mockUserRepo,
		mockRedisClient,
		logger,
	)

	t.Run("get_notification_stats_success", func(t *testing.T) {
		// Mock Redis operations for queue stats
		mockRedisClient.On("ZCard", ctx, "notification_queue").Return(&redis.IntCmd{})
		mockRedisClient.On("ZCard", ctx, "notification_dlq").Return(&redis.IntCmd{})

		stats, err := notificationService.GetNotificationStats(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Contains(t, stats, "queue_size")
		assert.Contains(t, stats, "dlq_size")
		assert.Contains(t, stats, "is_processing")

		mockRedisClient.AssertExpectations(t)
	})
}

func TestNotificationService_QueuedNotificationJSON(t *testing.T) {
	// Test QueuedNotification JSON marshalling/unmarshalling
	queuedNotification := &services.QueuedNotification{
		ID:       uuid.New(),
		UserID:   uuid.New(),
		Type:     "test_type",
		Title:    "Test Title",
		Message:  "Test Message",
		Channels: []services.NotificationChannel{services.ChannelEmail, services.ChannelPush},
		Priority: services.PriorityHigh,
		Data:     map[string]interface{}{"key": "value"},
	}

	// Marshal to JSON
	data, err := json.Marshal(queuedNotification)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshalled services.QueuedNotification
	err = json.Unmarshal(data, &unmarshalled)
	assert.NoError(t, err)
	assert.Equal(t, queuedNotification.ID, unmarshalled.ID)
	assert.Equal(t, queuedNotification.Type, unmarshalled.Type)
	assert.Equal(t, queuedNotification.Priority, unmarshalled.Priority)
}

func TestNotificationService_NotificationChannelTypes(t *testing.T) {
	// Test different notification channel types
	channels := []services.NotificationChannel{
		services.ChannelInApp,
		services.ChannelEmail,
		services.ChannelPush,
		services.ChannelSMS,
	}

	expectedStrings := []string{"app", "email", "push", "sms"}

	for i, channel := range channels {
		assert.Equal(t, expectedStrings[i], string(channel))
	}
}

func TestNotificationService_NotificationPriorityTypes(t *testing.T) {
	// Test different notification priority types
	priorities := []services.NotificationPriority{
		services.PriorityLow,
		services.PriorityNormal,
		services.PriorityHigh,
		services.PriorityUrgent,
	}

	expectedStrings := []string{"low", "normal", "high", "urgent"}

	for i, priority := range priorities {
		assert.Equal(t, expectedStrings[i], string(priority))
	}
}
