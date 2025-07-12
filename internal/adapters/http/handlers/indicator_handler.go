package handlers

import (
	"net/http"
	"strconv"

	"github.com/felipe-macedo/go-priceguard-api/internal/application/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// IndicatorHandler handles technical indicator requests
type IndicatorHandler struct {
	indicatorService *services.TechnicalIndicatorService
	logger           *logrus.Logger
}

// NewIndicatorHandler creates a new indicator handler
func NewIndicatorHandler(indicatorService *services.TechnicalIndicatorService, logger *logrus.Logger) *IndicatorHandler {
	return &IndicatorHandler{
		indicatorService: indicatorService,
		logger:           logger,
	}
}

// CalculateRSI calculates RSI for a symbol and timeframe
// @Summary Calculate RSI
// @Description Calculate RSI indicator for a specific symbol and timeframe
// @Tags indicators
// @Accept json
// @Produce json
// @Param symbol path string true "Cryptocurrency symbol"
// @Param timeframe query string true "Timeframe (1m, 5m, 15m, 1h, 4h, 1d)"
// @Param period query int false "RSI period (default: 14)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/indicators/{symbol}/rsi [post]
func (h *IndicatorHandler) CalculateRSI(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.Query("timeframe")

	if symbol == "" || timeframe == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol and timeframe are required",
		})
		return
	}

	period := 14 // default
	if p := c.Query("period"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			period = parsed
		}
	}

	err := h.indicatorService.CalculateAndStoreRSI(c.Request.Context(), symbol, timeframe, period)
	if err != nil {
		h.logger.WithError(err).Error("Failed to calculate RSI")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to calculate RSI",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "RSI calculated successfully",
		"symbol":    symbol,
		"timeframe": timeframe,
		"period":    period,
	})
}

// CalculateEMA calculates EMA for a symbol and timeframe
// @Summary Calculate EMA
// @Description Calculate EMA indicator for a specific symbol and timeframe
// @Tags indicators
// @Accept json
// @Produce json
// @Param symbol path string true "Cryptocurrency symbol"
// @Param timeframe query string true "Timeframe (1m, 5m, 15m, 1h, 4h, 1d)"
// @Param period query int false "EMA period (default: 12)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/indicators/{symbol}/ema [post]
func (h *IndicatorHandler) CalculateEMA(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.Query("timeframe")

	if symbol == "" || timeframe == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol and timeframe are required",
		})
		return
	}

	period := 12 // default
	if p := c.Query("period"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			period = parsed
		}
	}

	err := h.indicatorService.CalculateAndStoreEMA(c.Request.Context(), symbol, timeframe, period)
	if err != nil {
		h.logger.WithError(err).Error("Failed to calculate EMA")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to calculate EMA",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "EMA calculated successfully",
		"symbol":    symbol,
		"timeframe": timeframe,
		"period":    period,
	})
}

// CalculateSMA calculates SMA for a symbol and timeframe
// @Summary Calculate SMA
// @Description Calculate SMA indicator for a specific symbol and timeframe
// @Tags indicators
// @Accept json
// @Produce json
// @Param symbol path string true "Cryptocurrency symbol"
// @Param timeframe query string true "Timeframe (1m, 5m, 15m, 1h, 4h, 1d)"
// @Param period query int false "SMA period (default: 20)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/indicators/{symbol}/sma [post]
func (h *IndicatorHandler) CalculateSMA(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.Query("timeframe")

	if symbol == "" || timeframe == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol and timeframe are required",
		})
		return
	}

	period := 20 // default
	if p := c.Query("period"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			period = parsed
		}
	}

	err := h.indicatorService.CalculateAndStoreSMA(c.Request.Context(), symbol, timeframe, period)
	if err != nil {
		h.logger.WithError(err).Error("Failed to calculate SMA")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to calculate SMA",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "SMA calculated successfully",
		"symbol":    symbol,
		"timeframe": timeframe,
		"period":    period,
	})
}

// CalculateSuperTrend calculates SuperTrend for a symbol and timeframe
// @Summary Calculate SuperTrend
// @Description Calculate SuperTrend indicator for a specific symbol and timeframe
// @Tags indicators
// @Accept json
// @Produce json
// @Param symbol path string true "Cryptocurrency symbol"
// @Param timeframe query string true "Timeframe (1m, 5m, 15m, 1h, 4h, 1d)"
// @Param period query int false "ATR period (default: 10)"
// @Param multiplier query number false "Multiplier (default: 3.0)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/indicators/{symbol}/supertrend [post]
func (h *IndicatorHandler) CalculateSuperTrend(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.Query("timeframe")

	if symbol == "" || timeframe == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol and timeframe are required",
		})
		return
	}

	period := 10 // default
	if p := c.Query("period"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			period = parsed
		}
	}

	multiplier := 3.0 // default
	if m := c.Query("multiplier"); m != "" {
		if parsed, err := strconv.ParseFloat(m, 64); err == nil {
			multiplier = parsed
		}
	}

	err := h.indicatorService.CalculateAndStoreSuperTrend(c.Request.Context(), symbol, timeframe, period, multiplier)
	if err != nil {
		h.logger.WithError(err).Error("Failed to calculate SuperTrend")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to calculate SuperTrend",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "SuperTrend calculated successfully",
		"symbol":     symbol,
		"timeframe":  timeframe,
		"period":     period,
		"multiplier": multiplier,
	})
}

// CalculateAllIndicators calculates all indicators for a symbol and timeframe
// @Summary Calculate All Indicators
// @Description Calculate all available indicators for a specific symbol and timeframe
// @Tags indicators
// @Accept json
// @Produce json
// @Param symbol path string true "Cryptocurrency symbol"
// @Param timeframe query string true "Timeframe (1m, 5m, 15m, 1h, 4h, 1d)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/indicators/{symbol}/all [post]
func (h *IndicatorHandler) CalculateAllIndicators(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.Query("timeframe")

	if symbol == "" || timeframe == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol and timeframe are required",
		})
		return
	}

	err := h.indicatorService.CalculateAllIndicators(c.Request.Context(), symbol, timeframe)
	if err != nil {
		h.logger.WithError(err).Error("Failed to calculate all indicators")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to calculate indicators",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "All indicators calculated successfully",
		"symbol":    symbol,
		"timeframe": timeframe,
	})
}

// GetLatestIndicators gets the latest calculated indicators for a symbol and timeframe
// @Summary Get Latest Indicators
// @Description Get the latest calculated indicators for a specific symbol and timeframe
// @Tags indicators
// @Accept json
// @Produce json
// @Param symbol path string true "Cryptocurrency symbol"
// @Param timeframe query string true "Timeframe (1m, 5m, 15m, 1h, 4h, 1d)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/indicators/{symbol}/latest [get]
func (h *IndicatorHandler) GetLatestIndicators(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.Query("timeframe")

	if symbol == "" || timeframe == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol and timeframe are required",
		})
		return
	}

	indicators, err := h.indicatorService.GetLatestIndicators(c.Request.Context(), symbol, timeframe)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get latest indicators")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get indicators",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":     symbol,
		"timeframe":  timeframe,
		"indicators": indicators,
	})
}
