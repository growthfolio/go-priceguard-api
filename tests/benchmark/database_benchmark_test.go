package benchmark

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// BenchmarkDatabaseOperations testa performance de operações de banco
func BenchmarkDatabaseOperations(b *testing.B) {
	// Configurar banco de teste para benchmark
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(db)

	// Criar usuário de teste
	user := &entities.User{
		GoogleID: fmt.Sprintf("bench_user_%d", time.Now().UnixNano()),
		Email:    fmt.Sprintf("bench_%d@test.com", time.Now().UnixNano()),
		Name:     "Benchmark User",
	}
	err := db.Create(user).Error
	require.NoError(b, err)

	b.Run("CreateAlert", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			alert := &entities.Alert{
				UserID:        user.ID,
				Symbol:        "BTCUSDT",
				AlertType:     "price",
				ConditionType: "above",
				TargetValue:   50000.0,
				Timeframe:     "1h",
				Enabled:       true,
			}
			err := db.Create(alert).Error
			require.NoError(b, err)
		}
	})

	b.Run("GetUserAlerts", func(b *testing.B) {
		// Criar alguns alertas primeiro
		for i := 0; i < 100; i++ {
			alert := &entities.Alert{
				UserID:        user.ID,
				Symbol:        fmt.Sprintf("SYMBOL%d", i),
				AlertType:     "price",
				ConditionType: "above",
				TargetValue:   float64(50000 + i),
				Timeframe:     "1h",
				Enabled:       true,
			}
			_ = db.Create(alert).Error
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var alerts []entities.Alert
			err := db.Where("user_id = ?", user.ID).Find(&alerts).Error
			require.NoError(b, err)
		}
	})

	b.Run("UpdateAlert", func(b *testing.B) {
		// Criar alerta para atualizar
		alert := &entities.Alert{
			UserID:        user.ID,
			Symbol:        "BTCUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   50000.0,
			Timeframe:     "1h",
			Enabled:       true,
		}
		err := db.Create(alert).Error
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			alert.TargetValue = float64(50000 + i)
			err := db.Save(alert).Error
			require.NoError(b, err)
		}
	})

	b.Run("BatchCreateAlerts", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			alerts := make([]*entities.Alert, 10)
			for j := 0; j < 10; j++ {
				alerts[j] = &entities.Alert{
					UserID:        user.ID,
					Symbol:        fmt.Sprintf("BATCH%d_%d", i, j),
					AlertType:     "price",
					ConditionType: "above",
					TargetValue:   float64(50000 + j),
					Timeframe:     "1h",
					Enabled:       true,
				}
			}

			// Batch insert
			err := db.CreateInBatches(alerts, 10).Error
			require.NoError(b, err)
		}
	})
}

// BenchmarkConcurrentDatabaseAccess testa performance sob concorrência
func BenchmarkConcurrentDatabaseAccess(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(db)

	// Criar usuário de teste
	user := &entities.User{
		GoogleID: fmt.Sprintf("concurrent_user_%d", time.Now().UnixNano()),
		Email:    fmt.Sprintf("concurrent_%d@test.com", time.Now().UnixNano()),
		Name:     "Concurrent User",
	}
	err := db.Create(user).Error
	require.NoError(b, err)

	b.Run("ConcurrentReads", func(b *testing.B) {
		// Criar alguns alertas primeiro
		for i := 0; i < 50; i++ {
			alert := &entities.Alert{
				UserID:        user.ID,
				Symbol:        fmt.Sprintf("CONCURRENT%d", i),
				AlertType:     "price",
				ConditionType: "above",
				TargetValue:   float64(50000 + i),
				Timeframe:     "1h",
				Enabled:       true,
			}
			_ = db.Create(alert).Error
		}

		b.ResetTimer()
		var wg sync.WaitGroup
		concurrency := 10

		for i := 0; i < b.N; i++ {
			wg.Add(concurrency)
			for j := 0; j < concurrency; j++ {
				go func() {
					defer wg.Done()
					var alerts []entities.Alert
					err := db.Where("user_id = ?", user.ID).Find(&alerts).Error
					require.NoError(b, err)
				}()
			}
			wg.Wait()
		}
	})

	b.Run("ConcurrentWrites", func(b *testing.B) {
		b.ResetTimer()
		var wg sync.WaitGroup
		concurrency := 5

		for i := 0; i < b.N; i++ {
			wg.Add(concurrency)
			for j := 0; j < concurrency; j++ {
				go func(iteration, worker int) {
					defer wg.Done()
					alert := &entities.Alert{
						UserID:        user.ID,
						Symbol:        fmt.Sprintf("WRITE_%d_%d", iteration, worker),
						AlertType:     "price",
						ConditionType: "above",
						TargetValue:   float64(50000 + iteration + worker),
						Timeframe:     "1h",
						Enabled:       true,
					}
					err := db.Create(alert).Error
					require.NoError(b, err)
				}(i, j)
			}
			wg.Wait()
		}
	})
}

// BenchmarkConnectionPooling testa performance do pool de conexões
func BenchmarkConnectionPooling(b *testing.B) {
	configs := []struct {
		name        string
		maxOpen     int
		maxIdle     int
		maxLifetime time.Duration
	}{
		{"Small_Pool", 10, 5, time.Hour},
		{"Medium_Pool", 25, 10, time.Hour},
		{"Large_Pool", 50, 20, time.Hour},
	}

	for _, config := range configs {
		b.Run(config.name, func(b *testing.B) {
			db := setupBenchmarkDBWithConfig(b, config.maxOpen, config.maxIdle, config.maxLifetime)
			defer cleanupBenchmarkDB(db)

			b.ResetTimer()
			var wg sync.WaitGroup
			concurrency := 20

			for i := 0; i < b.N; i++ {
				wg.Add(concurrency)
				for j := 0; j < concurrency; j++ {
					go func(iteration, worker int) {
						defer wg.Done()
						user := &entities.User{
							GoogleID: fmt.Sprintf("pool_test_%d_%d_%d", config.maxOpen, iteration, worker),
							Email:    fmt.Sprintf("pool_%d_%d_%d@test.com", config.maxOpen, iteration, worker),
							Name:     "Pool Test User",
						}
						err := db.Create(user).Error
						require.NoError(b, err)
					}(i, j)
				}
				wg.Wait()
			}
		})
	}
}

// Funções auxiliares para setup e cleanup
func setupBenchmarkDB(b *testing.B) *gorm.DB {
	return setupBenchmarkDBWithConfig(b, 25, 10, time.Hour)
}

func setupBenchmarkDBWithConfig(b *testing.B, maxOpen, maxIdle int, maxLifetime time.Duration) *gorm.DB {
	dsn := "host=localhost user=postgres password=postgres dbname=priceguard_benchmark port=5432 sslmode=disable"

	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Silenciar logs para benchmark
	}

	db, err := gorm.Open(postgres.Open(dsn), config)
	require.NoError(b, err)

	sqlDB, err := db.DB()
	require.NoError(b, err)

	// Configurar pool de conexões
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(maxLifetime)

	// Auto-migrate para benchmark
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
	require.NoError(b, err)

	return db
}

func cleanupBenchmarkDB(db *gorm.DB) {
	// Limpar dados de teste
	db.Exec("DELETE FROM alerts WHERE symbol LIKE 'BENCH%' OR symbol LIKE 'SYMBOL%' OR symbol LIKE 'BATCH%' OR symbol LIKE 'CONCURRENT%' OR symbol LIKE 'WRITE_%'")
	db.Exec("DELETE FROM users WHERE google_id LIKE 'bench_%' OR google_id LIKE 'concurrent_%' OR google_id LIKE 'pool_test_%'")
}
