package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/growthfolio/go-priceguard-api/internal/domain/repositories"
)

type CryptoHandler struct {
	cryptoRepo    repositories.CryptoCurrencyRepository
	priceHistRepo repositories.PriceHistoryRepository
	techRepo      repositories.TechnicalIndicatorRepository
}

// NewCryptoHandler creates a new crypto handler
func NewCryptoHandler(
	cryptoRepo repositories.CryptoCurrencyRepository,
	priceHistRepo repositories.PriceHistoryRepository,
	techRepo repositories.TechnicalIndicatorRepository,
) *CryptoHandler {
	return &CryptoHandler{
		cryptoRepo:    cryptoRepo,
		priceHistRepo: priceHistRepo,
		techRepo:      techRepo,
	}
}

// GetCryptoData godoc
// @Summary Get cryptocurrency data
// @Description Get list of cryptocurrencies with optional filtering
// @Tags Crypto
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param active query bool false "Filter by active status"
// @Param limit query int false "Limit number of results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} entities.CryptoCurrency
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/crypto/data [get]
func (h *CryptoHandler) GetCryptoData(c *gin.Context) {
	// Parse query parameters
	activeStr := c.DefaultQuery("active", "true")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	active, _ := strconv.ParseBool(activeStr)
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	// Validate limits
	if limit > 100 {
		limit = 100
	}
	if limit <= 0 {
		limit = 50
	}

	var cryptos []entities.CryptoCurrency
	var err error

	if active {
		cryptos, err = h.cryptoRepo.GetActive(c.Request.Context(), limit, offset)
	} else {
		cryptos, err = h.cryptoRepo.GetAll(c.Request.Context(), limit, offset)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cryptocurrency data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   cryptos,
		"limit":  limit,
		"offset": offset,
		"count":  len(cryptos),
	})
}

// GetCryptoDetail godoc
// @Summary Get cryptocurrency details
// @Description Get detailed information about a specific cryptocurrency
// @Tags Crypto
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param symbol path string true "Cryptocurrency symbol"
// @Success 200 {object} entities.CryptoCurrency
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Cryptocurrency not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/crypto/detail/{symbol} [get]
func (h *CryptoHandler) GetCryptoDetail(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	crypto, err := h.cryptoRepo.GetBySymbol(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cryptocurrency not found"})
		return
	}

	c.JSON(http.StatusOK, crypto)
}

// GetPriceHistory godoc
// @Summary Get price history
// @Description Get historical price data for a cryptocurrency
// @Tags Crypto
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param symbol path string true "Cryptocurrency symbol"
// @Param timeframe query string false "Timeframe (1m, 5m, 15m, 1h, 4h, 1d)" default("1h")
// @Param limit query int false "Limit number of results" default(100)
// @Success 200 {array} entities.PriceHistory
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/crypto/history/{symbol} [get]
func (h *CryptoHandler) GetPriceHistory(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	timeframe := c.DefaultQuery("timeframe", "1h")
	limitStr := c.DefaultQuery("limit", "100")

	limit, _ := strconv.Atoi(limitStr)
	if limit > 1000 {
		limit = 1000
	}
	if limit <= 0 {
		limit = 100
	}

	// Validate timeframe
	validTimeframes := map[string]bool{
		"1m": true, "5m": true, "15m": true, "30m": true,
		"1h": true, "4h": true, "1d": true, "1w": true,
	}
	if !validTimeframes[timeframe] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timeframe"})
		return
	}

	history, err := h.priceHistRepo.GetBySymbol(c.Request.Context(), symbol, timeframe, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch price history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":    symbol,
		"timeframe": timeframe,
		"data":      history,
		"count":     len(history),
	})
}

// GetTechnicalIndicators godoc
// @Summary Get technical indicators
// @Description Get technical indicators for a cryptocurrency
// @Tags Crypto
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param symbol path string true "Cryptocurrency symbol"
// @Param timeframe query string false "Timeframe (1m, 5m, 15m, 1h, 4h, 1d)" default("1h")
// @Param indicator_type query string false "Indicator type (RSI, EMA, SMA, SuperTrend)" default("RSI")
// @Param limit query int false "Limit number of results" default(50)
// @Success 200 {array} entities.TechnicalIndicator
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/crypto/indicators/{symbol} [get]
func (h *CryptoHandler) GetTechnicalIndicators(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	timeframe := c.DefaultQuery("timeframe", "1h")
	indicatorType := c.DefaultQuery("indicator_type", "RSI")
	limitStr := c.DefaultQuery("limit", "50")

	limit, _ := strconv.Atoi(limitStr)
	if limit > 500 {
		limit = 500
	}
	if limit <= 0 {
		limit = 50
	}

	// Validate timeframe
	validTimeframes := map[string]bool{
		"1m": true, "5m": true, "15m": true, "30m": true,
		"1h": true, "4h": true, "1d": true, "1w": true,
	}
	if !validTimeframes[timeframe] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timeframe"})
		return
	}

	// Validate indicator type
	validIndicators := map[string]bool{
		"RSI": true, "EMA": true, "SMA": true, "SuperTrend": true,
		"MACD": true, "BB": true, "Stochastic": true,
	}
	if !validIndicators[indicatorType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid indicator type"})
		return
	}

	indicators, err := h.techRepo.GetBySymbol(c.Request.Context(), symbol, timeframe, indicatorType, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch technical indicators"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":         symbol,
		"timeframe":      timeframe,
		"indicator_type": indicatorType,
		"data":           indicators,
		"count":          len(indicators),
	})
}
