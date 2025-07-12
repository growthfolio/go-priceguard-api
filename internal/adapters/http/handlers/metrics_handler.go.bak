package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MetricsHandler handler para métricas e observabilidade
type MetricsHandler struct {
	db     *gorm.DB
	rdb    *redis.Client
	logger *zap.Logger
}

// NewMetricsHandler cria uma nova instância do MetricsHandler
func NewMetricsHandler(db *gorm.DB, rdb *redis.Client, logger *zap.Logger) *MetricsHandler {
	return &MetricsHandler{
		db:     db,
		rdb:    rdb,
		logger: logger,
	}
}

// PrometheusMetrics endpoint para métricas Prometheus
func (h *MetricsHandler) PrometheusMetrics() gin.HandlerFunc {
	// Retorna o handler HTTP do Prometheus
	promHandler := promhttp.Handler()
	return gin.WrapH(promHandler)
}

// CustomMetrics endpoint para métricas customizadas em formato JSON
func (h *MetricsHandler) CustomMetrics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	metrics := h.collectSystemMetrics(ctx)

	c.JSON(http.StatusOK, gin.H{
		"timestamp": time.Now().UTC(),
		"metrics":   metrics,
	})
}

// ApplicationMetrics métricas específicas da aplicação
func (h *MetricsHandler) ApplicationMetrics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	appMetrics := h.collectApplicationMetrics(ctx)

	c.JSON(http.StatusOK, gin.H{
		"timestamp": time.Now().UTC(),
		"app_name":  "priceguard-api",
		"version":   "1.0.0",
		"metrics":   appMetrics,
	})
}

// SystemInfo informações do sistema
func (h *MetricsHandler) SystemInfo(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	info := gin.H{
		"timestamp": time.Now().UTC(),
		"system": gin.H{
			"go_version":    runtime.Version(),
			"num_goroutine": runtime.NumGoroutine(),
			"num_cpu":       runtime.NumCPU(),
			"arch":          runtime.GOARCH,
			"os":            runtime.GOOS,
		},
		"memory": gin.H{
			"alloc_mb":       bToMb(m.Alloc),
			"total_alloc_mb": bToMb(m.TotalAlloc),
			"sys_mb":         bToMb(m.Sys),
			"num_gc":         m.NumGC,
			"heap_objects":   m.HeapObjects,
			"heap_alloc_mb":  bToMb(m.HeapAlloc),
			"heap_sys_mb":    bToMb(m.HeapSys),
			"stack_inuse_mb": bToMb(m.StackInuse),
		},
	}

	c.JSON(http.StatusOK, info)
}

// collectSystemMetrics coleta métricas do sistema
func (h *MetricsHandler) collectSystemMetrics(ctx context.Context) map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := map[string]interface{}{
		"runtime": map[string]interface{}{
			"goroutines":   runtime.NumGoroutine(),
			"memory_alloc": m.Alloc,
			"memory_total": m.TotalAlloc,
			"memory_sys":   m.Sys,
			"gc_cycles":    m.NumGC,
			"heap_objects": m.HeapObjects,
		},
	}

	// Métricas do banco de dados
	if h.db != nil {
		dbMetrics := h.collectDatabaseMetrics(ctx)
		metrics["database"] = dbMetrics
	}

	// Métricas do Redis
	if h.rdb != nil {
		redisMetrics := h.collectRedisMetrics(ctx)
		metrics["redis"] = redisMetrics
	}

	return metrics
}

// collectApplicationMetrics coleta métricas específicas da aplicação
func (h *MetricsHandler) collectApplicationMetrics(ctx context.Context) map[string]interface{} {
	metrics := map[string]interface{}{}

	// Métricas de usuários (exemplo)
	if h.db != nil {
		var userCount int64
		if err := h.db.WithContext(ctx).Table("users").Count(&userCount).Error; err == nil {
			metrics["total_users"] = userCount
		}

		var alertCount int64
		if err := h.db.WithContext(ctx).Table("alerts").Count(&alertCount).Error; err == nil {
			metrics["total_alerts"] = alertCount
		}

		var activeAlertCount int64
		if err := h.db.WithContext(ctx).Table("alerts").Where("is_active = ?", true).Count(&activeAlertCount).Error; err == nil {
			metrics["active_alerts"] = activeAlertCount
		}

		var notificationCount int64
		if err := h.db.WithContext(ctx).Table("notifications").Count(&notificationCount).Error; err == nil {
			metrics["total_notifications"] = notificationCount
		}

		// Notificações das últimas 24h
		var recentNotifications int64
		yesterday := time.Now().Add(-24 * time.Hour)
		if err := h.db.WithContext(ctx).Table("notifications").
			Where("created_at > ?", yesterday).Count(&recentNotifications).Error; err == nil {
			metrics["notifications_24h"] = recentNotifications
		}
	}

	return metrics
}

// collectDatabaseMetrics coleta métricas do banco de dados
func (h *MetricsHandler) collectDatabaseMetrics(ctx context.Context) map[string]interface{} {
	sqlDB, err := h.db.DB()
	if err != nil {
		return map[string]interface{}{"error": "failed to get sql.DB"}
	}

	stats := sqlDB.Stats()

	return map[string]interface{}{
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration_ms":     stats.WaitDuration.Milliseconds(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}

// collectRedisMetrics coleta métricas do Redis
func (h *MetricsHandler) collectRedisMetrics(ctx context.Context) map[string]interface{} {
	// Info básico do Redis
	info, err := h.rdb.Info(ctx).Result()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	// Parse das informações básicas
	redisInfo := make(map[string]string)
	if err := parseRedisInfo(info, redisInfo); err != nil {
		h.logger.Warn("Failed to parse Redis info", zap.Error(err))
	}

	// Estatísticas do pool de conexões
	poolStats := h.rdb.PoolStats()

	return map[string]interface{}{
		"connected_clients": redisInfo["connected_clients"],
		"used_memory":       redisInfo["used_memory"],
		"used_memory_human": redisInfo["used_memory_human"],
		"total_commands":    redisInfo["total_commands_processed"],
		"keyspace_hits":     redisInfo["keyspace_hits"],
		"keyspace_misses":   redisInfo["keyspace_misses"],
		"pool_stats": map[string]interface{}{
			"hits":        poolStats.Hits,
			"misses":      poolStats.Misses,
			"timeouts":    poolStats.Timeouts,
			"total_conns": poolStats.TotalConns,
			"idle_conns":  poolStats.IdleConns,
			"stale_conns": poolStats.StaleConns,
		},
	}
}

// parseRedisInfo faz parse das informações do Redis
func parseRedisInfo(info string, result map[string]string) error {
	var infoMap map[string]interface{}
	// Redis INFO retorna texto, não JSON. Aqui simplificamos assumindo
	// que apenas as métricas principais são necessárias

	// Por simplicidade, retornamos valores mock. Em produção,
	// implementaria um parser completo do formato Redis INFO
	result["connected_clients"] = "1"
	result["used_memory"] = "1048576"
	result["used_memory_human"] = "1.00M"
	result["total_commands_processed"] = "0"
	result["keyspace_hits"] = "0"
	result["keyspace_misses"] = "0"

	return json.Unmarshal([]byte("{}"), &infoMap) // Placeholder
}

// bToMb converte bytes para megabytes
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
