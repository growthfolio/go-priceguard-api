package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics
var (
	// HTTP request metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being served",
		},
	)

	// WebSocket metrics
	websocketConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "websocket_connections_active",
			Help: "Current number of active WebSocket connections",
		},
	)

	websocketMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_messages_total",
			Help: "Total number of WebSocket messages",
		},
		[]string{"direction", "type"},
	)

	// Database metrics
	databaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Current number of active database connections",
		},
	)

	databaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Duration of database queries in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"query_type"},
	)

	// Redis metrics
	redisOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_operations_total",
			Help: "Total number of Redis operations",
		},
		[]string{"operation", "status"},
	)

	redisOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_operation_duration_seconds",
			Help:    "Duration of Redis operations in seconds",
			Buckets: []float64{.0001, .0005, .001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"operation"},
	)

	// Alert system metrics
	alertsProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerts_processed_total",
			Help: "Total number of alerts processed",
		},
		[]string{"type", "status"},
	)

	alertEvaluationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "alert_evaluation_duration_seconds",
			Help:    "Duration of alert evaluation in seconds",
			Buckets: []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
	)

	// Notification metrics
	notificationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notifications_total",
			Help: "Total number of notifications sent",
		},
		[]string{"channel", "status"},
	)

	notificationQueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "notification_queue_size",
			Help: "Current size of notification queue",
		},
	)
)

// PrometheusMiddleware middleware para coletar métricas HTTP
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Incrementa contador de requests em flight
		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		start := time.Now()

		c.Next()

		// Coleta métricas
		duration := time.Since(start).Seconds()
		method := c.Request.Method
		path := c.FullPath()
		status := strconv.Itoa(c.Writer.Status())

		// Se não tem path definido, usa a URL raw
		if path == "" {
			path = c.Request.URL.Path
		}

		// Incrementa contador total de requests
		httpRequestsTotal.WithLabelValues(method, path, status).Inc()

		// Registra duração da request
		httpRequestDuration.WithLabelValues(method, path, status).Observe(duration)
	}
}

// GetMetricsCollectors retorna os coletores de métricas para uso externo
func GetMetricsCollectors() *MetricsCollectors {
	return &MetricsCollectors{
		HTTPRequestsTotal:          httpRequestsTotal,
		HTTPRequestDuration:        httpRequestDuration,
		HTTPRequestsInFlight:       httpRequestsInFlight,
		WebSocketConnectionsActive: websocketConnectionsActive,
		WebSocketMessagesTotal:     websocketMessagesTotal,
		DatabaseConnectionsActive:  databaseConnectionsActive,
		DatabaseQueryDuration:      databaseQueryDuration,
		RedisOperationsTotal:       redisOperationsTotal,
		RedisOperationDuration:     redisOperationDuration,
		AlertsProcessedTotal:       alertsProcessedTotal,
		AlertEvaluationDuration:    alertEvaluationDuration,
		NotificationsTotal:         notificationsTotal,
		NotificationQueueSize:      notificationQueueSize,
	}
}

// MetricsCollectors estrutura que agrupa todos os coletores de métricas
type MetricsCollectors struct {
	HTTPRequestsTotal          *prometheus.CounterVec
	HTTPRequestDuration        *prometheus.HistogramVec
	HTTPRequestsInFlight       prometheus.Gauge
	WebSocketConnectionsActive prometheus.Gauge
	WebSocketMessagesTotal     *prometheus.CounterVec
	DatabaseConnectionsActive  prometheus.Gauge
	DatabaseQueryDuration      *prometheus.HistogramVec
	RedisOperationsTotal       *prometheus.CounterVec
	RedisOperationDuration     *prometheus.HistogramVec
	AlertsProcessedTotal       *prometheus.CounterVec
	AlertEvaluationDuration    prometheus.Histogram
	NotificationsTotal         *prometheus.CounterVec
	NotificationQueueSize      prometheus.Gauge
}
