//go:build ignore

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/growthfolio/go-priceguard-api/internal/infrastructure/database"
	"gorm.io/gorm"
)

// calculateEMA calcula uma EMA simples
func calculateEMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	multiplier := 2.0 / float64(period+1)
	ema := prices[0]
	for i := 1; i < len(prices); i++ {
		ema = ((prices[i] - ema) * multiplier) + ema
	}
	return ema
}

// calculateRSI calcula um RSI simples
func calculateRSI(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 0
	}
	var gains, losses float64
	for i := 1; i <= period; i++ {
		change := closes[i] - closes[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}
	if losses == 0 {
		return 100
	}
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func main() {
	ctx := context.Background()
	// Inicializa conexão com banco
	db, err := database.NewGormDB()
	if err != nil {
		panic(err)
	}

	// Lista de símbolos/timeframes para calcular
	symbols := []string{"BTCUSDT", "ETHUSDT"} // Exemplo, pode buscar do banco
	timeframes := []string{"1h", "4h"}

	for _, symbol := range symbols {
		for _, tf := range timeframes {
			var candles []entities.PriceHistory
			err := db.Where("symbol = ? AND timeframe = ?", symbol, tf).
				Order("timestamp DESC").Limit(50).Find(&candles).Error
			if err != nil || len(candles) < 14 {
				fmt.Printf("Sem dados suficientes para %s %s\n", symbol, tf)
				continue
			}

			closes := make([]float64, len(candles))
			for i, c := range candles {
				closes[i] = c.ClosePrice
			}

			ema := calculateEMA(closes, 14)
			rsi := calculateRSI(closes, 14)

			indicator := entities.TechnicalIndicator{
				Symbol:        symbol,
				Timeframe:     tf,
				IndicatorType: "ema",
				Value:         &ema,
				Timestamp:     time.Now(),
			}
			db.Create(&indicator)

			indicatorRSI := entities.TechnicalIndicator{
				Symbol:        symbol,
				Timeframe:     tf,
				IndicatorType: "rsi",
				Value:         &rsi,
				Timestamp:     time.Now(),
			}
			db.Create(&indicatorRSI)

			fmt.Printf("%s %s - EMA: %.2f, RSI: %.2f\n", symbol, tf, ema, rsi)
		}
	}
}
