package monitoring

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

// PerformanceMonitor monitora métricas de performance
type PerformanceMonitor struct {
	logger        *logrus.Logger
	collectors    []MetricCollector
	httpMetrics   *HTTPMetrics
	dbMetrics     *DatabaseMetrics
	cacheMetrics  *CacheMetrics
	systemMetrics *SystemMetrics
	wsMetrics     *WebSocketMetrics
	alertMetrics  *AlertEngineMetrics
	stopChan      chan bool
	wg            sync.WaitGroup
}

// MetricCollector interface para coletores de métricas
type MetricCollector interface {
	Collect(ctx context.Context) error
	Name() string
}

// HTTPMetrics métricas HTTP
type HTTPMetrics struct {
	RequestsTotal     prometheus.CounterVec
	RequestDuration   prometheus.HistogramVec
	ResponseSizeBytes prometheus.HistogramVec
	ActiveConnections prometheus.Gauge
	ErrorsTotal       prometheus.CounterVec
}

// DatabaseMetrics métricas de banco de dados
type DatabaseMetrics struct {
	QueriesTotal      prometheus.CounterVec
	QueryDuration     prometheus.HistogramVec
	ConnectionsActive prometheus.Gauge
	ConnectionsIdle   prometheus.Gauge
	ConnectionsMax    prometheus.Gauge
	TransactionsTotal prometheus.CounterVec
	SlowQueriesTotal  prometheus.Counter
}

// CacheMetrics métricas de cache
type CacheMetrics struct {
	HitsTotal      prometheus.CounterVec
	MissesTotal    prometheus.CounterVec
	SetsTotal      prometheus.CounterVec
	DeletesTotal   prometheus.CounterVec
	EvictionsTotal prometheus.CounterVec
	HitRatio       prometheus.GaugeVec
	Size           prometheus.GaugeVec
	AccessDuration prometheus.HistogramVec
}

// SystemMetrics métricas do sistema
type SystemMetrics struct {
	MemoryUsageBytes  prometheus.Gauge
	CPUUsagePercent   prometheus.Gauge
	GoroutinesActive  prometheus.Gauge
	GCDurationSeconds prometheus.Histogram
	GCRunsTotal       prometheus.Counter
	HeapSizeBytes     prometheus.Gauge
	HeapObjectsTotal  prometheus.Gauge
}

// WebSocketMetrics métricas WebSocket
type WebSocketMetrics struct {
	ConnectionsActive   prometheus.Gauge
	ConnectionsTotal    prometheus.Counter
	DisconnectionsTotal prometheus.CounterVec
	MessagesReceived    prometheus.CounterVec
	MessagesSent        prometheus.CounterVec
	BroadcastsTotal     prometheus.Counter
	BroadcastDuration   prometheus.Histogram
}

// AlertEngineMetrics métricas do motor de alertas
type AlertEngineMetrics struct {
	AlertsEvaluated    prometheus.CounterVec
	AlertsTriggered    prometheus.CounterVec
	EvaluationDuration prometheus.Histogram
	QueueSize          prometheus.Gauge
	WorkersActive      prometheus.Gauge
	ErrorsTotal        prometheus.CounterVec
}

// NewPerformanceMonitor cria novo monitor de performance
func NewPerformanceMonitor(logger *logrus.Logger) *PerformanceMonitor {
	pm := &PerformanceMonitor{
		logger:   logger,
		stopChan: make(chan bool),
	}

	pm.initializeMetrics()
	return pm
}

// initializeMetrics inicializa todas as métricas Prometheus
func (pm *PerformanceMonitor) initializeMetrics() {
	// HTTP Metrics
	pm.httpMetrics = &HTTPMetrics{
		RequestsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		RequestDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		ResponseSizeBytes: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000},
			},
			[]string{"method", "endpoint"},
		),
		ActiveConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_active_connections",
				Help: "Number of active HTTP connections",
			},
		),
		ErrorsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_errors_total",
				Help: "Total number of HTTP errors",
			},
			[]string{"method", "endpoint", "error_type"},
		),
	}

	// Database Metrics
	pm.dbMetrics = &DatabaseMetrics{
		QueriesTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation", "table", "status"},
		),
		QueryDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
			},
			[]string{"operation", "table"},
		),
		ConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_active",
				Help: "Number of active database connections",
			},
		),
		ConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_idle",
				Help: "Number of idle database connections",
			},
		),
		ConnectionsMax: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_max",
				Help: "Maximum number of database connections",
			},
		),
		TransactionsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_transactions_total",
				Help: "Total number of database transactions",
			},
			[]string{"status"},
		),
		SlowQueriesTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "db_slow_queries_total",
				Help: "Total number of slow database queries",
			},
		),
	}

	// Cache Metrics
	pm.cacheMetrics = &CacheMetrics{
		HitsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_type", "key_pattern"},
		),
		MissesTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_type", "key_pattern"},
		),
		SetsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_sets_total",
				Help: "Total number of cache sets",
			},
			[]string{"cache_type", "key_pattern"},
		),
		DeletesTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_deletes_total",
				Help: "Total number of cache deletes",
			},
			[]string{"cache_type", "key_pattern"},
		),
		EvictionsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_evictions_total",
				Help: "Total number of cache evictions",
			},
			[]string{"cache_type", "reason"},
		),
		HitRatio: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "cache_hit_ratio",
				Help: "Cache hit ratio",
			},
			[]string{"cache_type"},
		),
		Size: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "cache_size",
				Help: "Cache size",
			},
			[]string{"cache_type"},
		),
		AccessDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "cache_access_duration_seconds",
				Help:    "Cache access duration in seconds",
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
			},
			[]string{"cache_type", "operation"},
		),
	}

	// System Metrics
	pm.systemMetrics = &SystemMetrics{
		MemoryUsageBytes: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_memory_usage_bytes",
				Help: "System memory usage in bytes",
			},
		),
		CPUUsagePercent: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_cpu_usage_percent",
				Help: "System CPU usage percentage",
			},
		),
		GoroutinesActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_goroutines_active",
				Help: "Number of active goroutines",
			},
		),
		GCDurationSeconds: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "system_gc_duration_seconds",
				Help:    "Garbage collection duration in seconds",
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
			},
		),
		GCRunsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "system_gc_runs_total",
				Help: "Total number of garbage collection runs",
			},
		),
		HeapSizeBytes: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_heap_size_bytes",
				Help: "Heap size in bytes",
			},
		),
		HeapObjectsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_heap_objects_total",
				Help: "Total number of heap objects",
			},
		),
	}

	// WebSocket Metrics
	pm.wsMetrics = &WebSocketMetrics{
		ConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "websocket_connections_active",
				Help: "Number of active WebSocket connections",
			},
		),
		ConnectionsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "websocket_connections_total",
				Help: "Total number of WebSocket connections",
			},
		),
		DisconnectionsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "websocket_disconnections_total",
				Help: "Total number of WebSocket disconnections",
			},
			[]string{"reason"},
		),
		MessagesReceived: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "websocket_messages_received_total",
				Help: "Total number of WebSocket messages received",
			},
			[]string{"message_type"},
		),
		MessagesSent: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "websocket_messages_sent_total",
				Help: "Total number of WebSocket messages sent",
			},
			[]string{"message_type"},
		),
		BroadcastsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "websocket_broadcasts_total",
				Help: "Total number of WebSocket broadcasts",
			},
		),
		BroadcastDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "websocket_broadcast_duration_seconds",
				Help:    "WebSocket broadcast duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
			},
		),
	}

	// Alert Engine Metrics
	pm.alertMetrics = &AlertEngineMetrics{
		AlertsEvaluated: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alerts_evaluated_total",
				Help: "Total number of alerts evaluated",
			},
			[]string{"alert_type", "status"},
		),
		AlertsTriggered: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alerts_triggered_total",
				Help: "Total number of alerts triggered",
			},
			[]string{"alert_type", "symbol"},
		),
		EvaluationDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "alert_evaluation_duration_seconds",
				Help:    "Alert evaluation duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
			},
		),
		QueueSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "alert_queue_size",
				Help: "Alert evaluation queue size",
			},
		),
		WorkersActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "alert_workers_active",
				Help: "Number of active alert workers",
			},
		),
		ErrorsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_errors_total",
				Help: "Total number of alert processing errors",
			},
			[]string{"error_type"},
		),
	}
}

// Start inicia o monitoramento de performance
func (pm *PerformanceMonitor) Start(ctx context.Context, interval time.Duration) {
	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pm.collectSystemMetrics()
				pm.collectRuntimeMetrics()
			case <-pm.stopChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	pm.logger.Info("Performance monitoring started")
}

// Stop para o monitoramento
func (pm *PerformanceMonitor) Stop() {
	close(pm.stopChan)
	pm.wg.Wait()
	pm.logger.Info("Performance monitoring stopped")
}

// collectSystemMetrics coleta métricas do sistema
func (pm *PerformanceMonitor) collectSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Métricas de memória
	pm.systemMetrics.MemoryUsageBytes.Set(float64(m.Alloc))
	pm.systemMetrics.HeapSizeBytes.Set(float64(m.HeapAlloc))
	pm.systemMetrics.HeapObjectsTotal.Set(float64(m.HeapObjects))

	// Goroutines
	pm.systemMetrics.GoroutinesActive.Set(float64(runtime.NumGoroutine()))

	// GC metrics
	pm.systemMetrics.GCRunsTotal.Add(float64(m.NumGC))
}

// collectRuntimeMetrics coleta métricas de runtime
func (pm *PerformanceMonitor) collectRuntimeMetrics() {
	// CPU usage (simplified)
	// Em produção, usar library mais sofisticada como gopsutil
	pm.systemMetrics.CPUUsagePercent.Set(0) // Placeholder
}

// GetHTTPMetrics retorna métricas HTTP
func (pm *PerformanceMonitor) GetHTTPMetrics() *HTTPMetrics {
	return pm.httpMetrics
}

// GetDatabaseMetrics retorna métricas de banco
func (pm *PerformanceMonitor) GetDatabaseMetrics() *DatabaseMetrics {
	return pm.dbMetrics
}

// GetCacheMetrics retorna métricas de cache
func (pm *PerformanceMonitor) GetCacheMetrics() *CacheMetrics {
	return pm.cacheMetrics
}

// GetWebSocketMetrics retorna métricas WebSocket
func (pm *PerformanceMonitor) GetWebSocketMetrics() *WebSocketMetrics {
	return pm.wsMetrics
}

// GetAlertMetrics retorna métricas do motor de alertas
func (pm *PerformanceMonitor) GetAlertMetrics() *AlertEngineMetrics {
	return pm.alertMetrics
}

// AddCollector adiciona um coletor de métricas personalizado
func (pm *PerformanceMonitor) AddCollector(collector MetricCollector) {
	pm.collectors = append(pm.collectors, collector)
}
