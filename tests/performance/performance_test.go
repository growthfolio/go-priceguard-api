package performance_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/growthfolio/go-priceguard-api/internal/adapters/repository"
	"github.com/growthfolio/go-priceguard-api/internal/application/services"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB creates an in-memory database for performance testing
func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(err)
	}

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
	if err != nil {
		panic(err)
	}

	return db
}

// createTestUser creates a test user for performance tests
func createTestUser(db *gorm.DB, ctx context.Context) *entities.User {
	userRepo := repository.NewUserRepository(db)
	user := &entities.User{
		GoogleID: "perf-test-google-id",
		Email:    "perftest@example.com",
		Name:     "Performance Test User",
	}

	err := userRepo.Create(ctx, user)
	if err != nil {
		panic(err)
	}

	return user
}

func BenchmarkAlertCreation(b *testing.B) {
	db := setupTestDB()
	ctx := context.Background()
	user := createTestUser(db, ctx)
	alertRepo := repository.NewAlertRepository(db)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		alert := &entities.Alert{
			UserID:        user.ID,
			Symbol:        fmt.Sprintf("TEST%dUSDT", i),
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   float64(50000 + i),
			Timeframe:     "1h",
			Enabled:       true,
			NotifyVia:     []string{"app"},
		}

		err := alertRepo.Create(ctx, alert)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAlertRetrieval(b *testing.B) {
	db := setupTestDB()
	ctx := context.Background()
	user := createTestUser(db, ctx)
	alertRepo := repository.NewAlertRepository(db)

	// Pre-populate database with alerts
	numAlerts := 1000
	for i := 0; i < numAlerts; i++ {
		alert := &entities.Alert{
			UserID:        user.ID,
			Symbol:        fmt.Sprintf("TEST%dUSDT", i),
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   float64(50000 + i),
			Timeframe:     "1h",
			Enabled:       true,
			NotifyVia:     []string{"app"},
		}

		err := alertRepo.Create(ctx, alert)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := alertRepo.GetByUserID(ctx, user.ID, 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrentAlertCreation(b *testing.B) {
	db := setupTestDB()
	ctx := context.Background()
	user := createTestUser(db, ctx)
	alertRepo := repository.NewAlertRepository(db)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			alert := &entities.Alert{
				UserID:        user.ID,
				Symbol:        fmt.Sprintf("CONC%dUSDT", i),
				AlertType:     "price",
				ConditionType: "above",
				TargetValue:   float64(50000 + i),
				Timeframe:     "1h",
				Enabled:       true,
				NotifyVia:     []string{"app"},
			}

			err := alertRepo.Create(ctx, alert)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func TestAlertCreationPerformance(t *testing.T) {
	db := setupTestDB()
	ctx := context.Background()
	user := createTestUser(db, ctx)
	alertRepo := repository.NewAlertRepository(db)

	// Test creating 1000 alerts and measure time
	numAlerts := 1000
	start := time.Now()

	for i := 0; i < numAlerts; i++ {
		alert := &entities.Alert{
			UserID:        user.ID,
			Symbol:        fmt.Sprintf("PERF%dUSDT", i),
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   float64(50000 + i),
			Timeframe:     "1h",
			Enabled:       true,
			NotifyVia:     []string{"app"},
		}

		err := alertRepo.Create(ctx, alert)
		assert.NoError(t, err)
	}

	duration := time.Since(start)
	alertsPerSecond := float64(numAlerts) / duration.Seconds()

	t.Logf("Created %d alerts in %v", numAlerts, duration)
	t.Logf("Performance: %.2f alerts/second", alertsPerSecond)

	// Assert reasonable performance (should be able to create at least 100 alerts/second)
	assert.Greater(t, alertsPerSecond, 100.0, "Alert creation performance should be at least 100 alerts/second")
}

func TestConcurrentAlertAccess(t *testing.T) {
	db := setupTestDB()
	ctx := context.Background()
	user := createTestUser(db, ctx)
	alertRepo := repository.NewAlertRepository(db)

	// Pre-populate with some alerts
	numAlerts := 100
	alertIDs := make([]string, numAlerts)

	for i := 0; i < numAlerts; i++ {
		alert := &entities.Alert{
			UserID:        user.ID,
			Symbol:        fmt.Sprintf("CONCTEST%dUSDT", i),
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   float64(50000 + i),
			Timeframe:     "1h",
			Enabled:       true,
			NotifyVia:     []string{"app"},
		}

		err := alertRepo.Create(ctx, alert)
		assert.NoError(t, err)
		alertIDs[i] = alert.ID.String()
	}

	// Test concurrent access
	numGoroutines := 50
	numOperationsPerGoroutine := 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperationsPerGoroutine)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperationsPerGoroutine; j++ {
				// Mix of read and write operations
				if j%2 == 0 {
					// Read operation
					_, err := alertRepo.GetByUserID(ctx, user.ID, 10, 0)
					if err != nil {
						errors <- fmt.Errorf("goroutine %d read error: %v", goroutineID, err)
						return
					}
				} else {
					// Write operation
					alert := &entities.Alert{
						UserID:        user.ID,
						Symbol:        fmt.Sprintf("CONC%d_%dUSDT", goroutineID, j),
						AlertType:     "price",
						ConditionType: "above",
						TargetValue:   float64(50000 + goroutineID + j),
						Timeframe:     "1h",
						Enabled:       true,
						NotifyVia:     []string{"app"},
					}

					err := alertRepo.Create(ctx, alert)
					if err != nil {
						errors <- fmt.Errorf("goroutine %d write error: %v", goroutineID, err)
						return
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(start)
	totalOperations := numGoroutines * numOperationsPerGoroutine
	operationsPerSecond := float64(totalOperations) / duration.Seconds()

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
		errorCount++
	}

	t.Logf("Completed %d concurrent operations in %v", totalOperations, duration)
	t.Logf("Performance: %.2f operations/second", operationsPerSecond)
	t.Logf("Error count: %d", errorCount)

	assert.Equal(t, 0, errorCount, "No errors should occur during concurrent operations")
	assert.Greater(t, operationsPerSecond, 500.0, "Concurrent operations should achieve at least 500 ops/second")
}

func TestAlertEnginePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	db := setupTestDB()
	ctx := context.Background()
	user := createTestUser(db, ctx)

	// Setup repositories and services
	alertRepo := repository.NewAlertRepository(db)
	priceHistoryRepo := repository.NewPriceHistoryRepository(db)
	technicalIndicatorRepo := repository.NewTechnicalIndicatorRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)

	// Create AlertEngine
	alertEngine := services.NewAlertEngine(
		alertRepo,
		priceHistoryRepo,
		technicalIndicatorRepo,
		notificationRepo,
		nil, // TechnicalIndicatorService
		nil, // Logger
	)

	// Create multiple alerts
	numAlerts := 100
	for i := 0; i < numAlerts; i++ {
		alert := &entities.Alert{
			UserID:        user.ID,
			Symbol:        "BTCUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   float64(45000 + i*100),
			Timeframe:     "1h",
			Enabled:       true,
			NotifyVia:     []string{"app"},
		}

		err := alertRepo.Create(ctx, alert)
		assert.NoError(t, err)
	}

	// Create price history
	priceHistory := &entities.PriceHistory{
		Symbol:     "BTCUSDT",
		Timeframe:  "1h",
		OpenPrice:  49000.0,
		HighPrice:  51000.0,
		LowPrice:   48000.0,
		ClosePrice: 50000.0,
		Volume:     1000000.0,
		Timestamp:  time.Now(),
	}

	err := priceHistoryRepo.Create(ctx, priceHistory)
	assert.NoError(t, err)

	// Test alert evaluation performance
	start := time.Now()
	results, err := alertEngine.EvaluateAllAlerts(ctx)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.NotNil(t, results)

	alertsPerSecond := float64(numAlerts) / duration.Seconds()

	t.Logf("Evaluated %d alerts in %v", numAlerts, duration)
	t.Logf("Performance: %.2f alerts/second", alertsPerSecond)

	// Assert reasonable performance for alert evaluation
	assert.Greater(t, alertsPerSecond, 50.0, "Alert evaluation should process at least 50 alerts/second")
}
