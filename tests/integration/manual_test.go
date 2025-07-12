package main

import (
	"context"
	"fmt"
	"log"

	"github.com/growthfolio/go-priceguard-api/internal/adapters/repository"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	fmt.Println("Starting integration test...")

	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate tables
	err = db.AutoMigrate(
		&entities.User{},
		&entities.Alert{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Setup repositories
	userRepo := repository.NewUserRepository(db)
	alertRepo := repository.NewAlertRepository(db)

	ctx := context.Background()

	// Create test user
	user := &entities.User{
		GoogleID: "test-google-id",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	err = userRepo.Create(ctx, user)
	if err != nil {
		log.Fatal("Failed to create user:", err)
	}
	fmt.Printf("Created user: %s (ID: %s)\n", user.Name, user.ID)

	// Create test alert
	alert := &entities.Alert{
		UserID:        user.ID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
		NotifyVia:     []string{"app"},
	}

	err = alertRepo.Create(ctx, alert)
	if err != nil {
		log.Fatal("Failed to create alert:", err)
	}
	fmt.Printf("Created alert: %s %s %f (ID: %s)\n", alert.Symbol, alert.ConditionType, alert.TargetValue, alert.ID)

	// Retrieve the alert
	retrievedAlert, err := alertRepo.GetByID(ctx, alert.ID)
	if err != nil {
		log.Fatal("Failed to retrieve alert:", err)
	}
	fmt.Printf("Retrieved alert: %s %s %f\n", retrievedAlert.Symbol, retrievedAlert.ConditionType, retrievedAlert.TargetValue)

	// Test user alerts list
	userAlerts, err := alertRepo.GetByUserID(ctx, user.ID, 10, 0)
	if err != nil {
		log.Fatal("Failed to get user alerts:", err)
	}
	fmt.Printf("User has %d alerts\n", len(userAlerts))

	fmt.Println("Integration test completed successfully!")
}
