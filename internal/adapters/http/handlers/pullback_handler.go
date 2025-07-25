package handlers

import (
	"net/http"

	"github.com/growthfolio/go-priceguard-api/internal/application/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// PullbackHandler handles pullback entry signal requests
type PullbackHandler struct {
	pullbackService *services.PullbackEntryService
	logger          *logrus.Logger
}

// NewPullbackHandler creates a new pullback handler
func NewPullbackHandler(pullbackService *services.PullbackEntryService, logger *logrus.Logger) *PullbackHandler {
	return &PullbackHandler{
		pullbackService: pullbackService,
		logger:          logger,
	}
}

// AnalyzePullbackEntry analyzes pullback entry signals for a symbol and timeframe
// @Summary Analyze Pullback Entry
// @Description Analyze pullback entry signals for a specific symbol and timeframe
// @Tags pullback
// @Accept json
// @Produce json
// @Param symbol path string true "Cryptocurrency symbol"
// @Param timeframe query string true "Timeframe (5m, 15m, 1h, 4h, 1d)"
// @Success 200 {object} services.PullbackEntry
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/pullback/{symbol}/analyze [get]
func (h *PullbackHandler) AnalyzePullbackEntry(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.Query("timeframe")

	if symbol == "" || timeframe == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol and timeframe are required",
		})
		return
	}

	entry, err := h.pullbackService.AnalyzePullbackEntry(c.Request.Context(), symbol, timeframe)
	if err != nil {
		h.logger.WithError(err).Error("Failed to analyze pullback entry")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to analyze pullback entry",
		})
		return
	}

	c.JSON(http.StatusOK, entry)
}

// GetPullbackEntriesMultiTimeframe gets pullback entries for multiple timeframes
// @Summary Get Pullback Entries Multi Timeframe
// @Description Get pullback entry signals for a symbol across multiple timeframes
// @Tags pullback
// @Accept json
// @Produce json
// @Param symbol path string true "Cryptocurrency symbol"
// @Success 200 {object} map[string]services.PullbackEntry
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/pullback/{symbol}/multi [get]
func (h *PullbackHandler) GetPullbackEntriesMultiTimeframe(c *gin.Context) {
	symbol := c.Param("symbol")

	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol is required",
		})
		return
	}

	// Standard timeframes for pullback analysis
	timeframes := []string{"5m", "15m", "1h", "4h", "1d"}

	entries, err := h.pullbackService.GetPullbackEntriesForTimeframes(c.Request.Context(), symbol, timeframes)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get pullback entries")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get pullback entries",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":    symbol,
		"entries":   entries,
		"timestamp": entries,
	})
}
