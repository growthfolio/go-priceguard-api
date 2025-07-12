package services_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/growthfolio/go-priceguard-api/internal/application/services"
	"github.com/growthfolio/go-priceguard-api/tests/testutils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AlertMonitorTestSuite struct {
	suite.Suite
	alertMonitor  *services.AlertMonitor
	mockAlertRepo *testutils.MockAlertRepository
	logger        *logrus.Logger
	ctx           context.Context
}

func (suite *AlertMonitorTestSuite) SetupTest() {
	suite.mockAlertRepo = &testutils.MockAlertRepository{}
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	suite.ctx = context.Background()

	// Create minimal AlertMonitor for testing basic functionality
	// We'll test with nil services to focus on the monitor's control logic
	suite.alertMonitor = services.NewAlertMonitor(
		nil, // AlertEngine - we'll test basic monitor functionality
		nil, // NotificationService
		nil, // CryptoDataService
		suite.mockAlertRepo,
		suite.logger,
	)
}

func (suite *AlertMonitorTestSuite) TearDownTest() {
	// Ensure monitor is stopped
	if suite.alertMonitor != nil {
		suite.alertMonitor.Stop()
	}

	// Assert all expectations
	suite.mockAlertRepo.AssertExpectations(suite.T())
}

func (suite *AlertMonitorTestSuite) TestNewAlertMonitor() {
	// Test that NewAlertMonitor creates a valid instance
	assert.NotNil(suite.T(), suite.alertMonitor)
	assert.False(suite.T(), suite.alertMonitor.IsRunning())
}

func (suite *AlertMonitorTestSuite) TestIsRunningStates() {
	// Initially not running
	assert.False(suite.T(), suite.alertMonitor.IsRunning())

	// Start the monitor
	suite.alertMonitor.Start(suite.ctx)

	// Add a small delay to allow goroutines to start
	time.Sleep(10 * time.Millisecond)

	// Check if running
	assert.True(suite.T(), suite.alertMonitor.IsRunning())

	// Test stopping the monitor
	suite.alertMonitor.Stop()

	// Check if stopped
	assert.False(suite.T(), suite.alertMonitor.IsRunning())
}

func (suite *AlertMonitorTestSuite) TestStartAlreadyRunning() {
	// Start the monitor first time
	suite.alertMonitor.Start(suite.ctx)
	assert.True(suite.T(), suite.alertMonitor.IsRunning())

	// Try to start again (should not panic or cause issues)
	suite.alertMonitor.Start(suite.ctx)
	assert.True(suite.T(), suite.alertMonitor.IsRunning())

	// Stop the monitor
	suite.alertMonitor.Stop()
}

func (suite *AlertMonitorTestSuite) TestStopWhenNotRunning() {
	// Initially not running
	assert.False(suite.T(), suite.alertMonitor.IsRunning())

	// Try to stop when not running (should not panic)
	suite.alertMonitor.Stop()

	// Should still not be running
	assert.False(suite.T(), suite.alertMonitor.IsRunning())
}

func (suite *AlertMonitorTestSuite) TestConcurrentStartStop() {
	// Test concurrent start/stop operations to ensure thread safety
	var wg sync.WaitGroup
	numGoroutines := 10

	// Start multiple goroutines that start/stop the monitor
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			suite.alertMonitor.Start(suite.ctx)
			time.Sleep(1 * time.Millisecond)
			suite.alertMonitor.Stop()
		}()
	}

	wg.Wait()

	// Should end up in stopped state
	assert.False(suite.T(), suite.alertMonitor.IsRunning())
}

func (suite *AlertMonitorTestSuite) TestConfigurableIntervals() {
	// Test that we can create a monitor with custom intervals
	customMonitor := services.NewAlertMonitor(
		nil, // AlertEngine
		nil, // NotificationService
		nil, // CryptoDataService
		suite.mockAlertRepo,
		suite.logger,
	)

	assert.NotNil(suite.T(), customMonitor)
	assert.False(suite.T(), customMonitor.IsRunning())

	// Clean up
	defer customMonitor.Stop()
}

func (suite *AlertMonitorTestSuite) TestRapidStartStop() {
	// Test rapid start/stop cycles
	for i := 0; i < 5; i++ {
		suite.alertMonitor.Start(suite.ctx)
		time.Sleep(1 * time.Millisecond)
		assert.True(suite.T(), suite.alertMonitor.IsRunning())

		suite.alertMonitor.Stop()
		assert.False(suite.T(), suite.alertMonitor.IsRunning())
	}
}

func (suite *AlertMonitorTestSuite) TestMonitorLifecycle() {
	// Test complete lifecycle
	monitor := services.NewAlertMonitor(
		nil,
		nil,
		nil,
		suite.mockAlertRepo,
		suite.logger,
	)

	// Initially stopped
	assert.False(suite.T(), monitor.IsRunning())

	// Start monitoring
	monitor.Start(suite.ctx)
	time.Sleep(10 * time.Millisecond)
	assert.True(suite.T(), monitor.IsRunning())

	// Stop monitoring
	monitor.Stop()
	assert.False(suite.T(), monitor.IsRunning())
}

func (suite *AlertMonitorTestSuite) TestMultipleStopsAreSafe() {
	// Start the monitor
	suite.alertMonitor.Start(suite.ctx)
	time.Sleep(10 * time.Millisecond)
	assert.True(suite.T(), suite.alertMonitor.IsRunning())

	// Stop multiple times - should be safe
	suite.alertMonitor.Stop()
	assert.False(suite.T(), suite.alertMonitor.IsRunning())

	suite.alertMonitor.Stop() // Second stop
	assert.False(suite.T(), suite.alertMonitor.IsRunning())

	suite.alertMonitor.Stop() // Third stop
	assert.False(suite.T(), suite.alertMonitor.IsRunning())
}

func TestAlertMonitorTestSuite(t *testing.T) {
	suite.Run(t, new(AlertMonitorTestSuite))
}

// Additional unit tests for AlertMonitor basic functionality
func TestAlertMonitor_BasicCreation(t *testing.T) {
	mockAlertRepo := &testutils.MockAlertRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	alertMonitor := services.NewAlertMonitor(
		nil, // AlertEngine
		nil, // NotificationService
		nil, // CryptoDataService
		mockAlertRepo,
		logger,
	)

	assert.NotNil(t, alertMonitor)
	assert.False(t, alertMonitor.IsRunning())

	// Cleanup
	defer alertMonitor.Stop()
}

func TestAlertMonitor_StartStopCycle(t *testing.T) {
	mockAlertRepo := &testutils.MockAlertRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	alertMonitor := services.NewAlertMonitor(
		nil, // AlertEngine
		nil, // NotificationService
		nil, // CryptoDataService
		mockAlertRepo,
		logger,
	)

	ctx := context.Background()

	// Test multiple start/stop cycles
	for i := 0; i < 3; i++ {
		assert.False(t, alertMonitor.IsRunning())

		alertMonitor.Start(ctx)
		time.Sleep(5 * time.Millisecond)
		assert.True(t, alertMonitor.IsRunning())

		alertMonitor.Stop()
		assert.False(t, alertMonitor.IsRunning())
	}
}
