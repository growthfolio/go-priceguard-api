package config

import (
	"time"
)

// PerformanceConfig contém configurações otimizadas de performance
type PerformanceConfig struct {
	Database     DatabasePerformanceConfig     `mapstructure:"database"`
	Redis        RedisPerformanceConfig        `mapstructure:"redis"`
	HTTP         HTTPPerformanceConfig         `mapstructure:"http"`
	WebSocket    WebSocketPerformanceConfig    `mapstructure:"websocket"`
	Cache        CachePerformanceConfig        `mapstructure:"cache"`
	AlertEngine  AlertEnginePerformanceConfig  `mapstructure:"alert_engine"`
}

// DatabasePerformanceConfig configurações otimizadas para PostgreSQL
type DatabasePerformanceConfig struct {
	// Connection Pool
	MaxOpenConns    int           `mapstructure:"max_open_conns" default:"25"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" default:"10"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" default:"1h"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" default:"30m"`

	// Query Optimization
	QueryTimeout         time.Duration `mapstructure:"query_timeout" default:"30s"`
	SlowQueryThreshold   time.Duration `mapstructure:"slow_query_threshold" default:"100ms"`
	EnableQueryLogging   bool          `mapstructure:"enable_query_logging" default:"false"`
	EnablePreparedStmts  bool          `mapstructure:"enable_prepared_stmts" default:"true"`

	// Batch Operations
	BatchSize              int `mapstructure:"batch_size" default:"100"`
	MaxBatchInsertSize     int `mapstructure:"max_batch_insert_size" default:"1000"`
	EnableBatchOperations  bool `mapstructure:"enable_batch_operations" default:"true"`

	// Performance Tuning
	SharedPreloadLibraries string `mapstructure:"shared_preload_libraries" default:"pg_stat_statements"`
	WorkMem                string `mapstructure:"work_mem" default:"4MB"`
	MaintenanceWorkMem     string `mapstructure:"maintenance_work_mem" default:"64MB"`
	EffectiveCacheSize     string `mapstructure:"effective_cache_size" default:"1GB"`
}

// RedisPerformanceConfig configurações otimizadas para Redis
type RedisPerformanceConfig struct {
	// Connection Pool
	PoolSize        int           `mapstructure:"pool_size" default:"25"`
	MinIdleConns    int           `mapstructure:"min_idle_conns" default:"10"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" default:"20"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" default:"30m"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" default:"1h"`

	// Performance Tuning
	DialTimeout  time.Duration `mapstructure:"dial_timeout" default:"5s"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" default:"3s"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" default:"3s"`
	PoolTimeout  time.Duration `mapstructure:"pool_timeout" default:"4s"`

	// Pipeline Configuration
	EnablePipelining    bool `mapstructure:"enable_pipelining" default:"true"`
	MaxPipelineSize     int  `mapstructure:"max_pipeline_size" default:"100"`
	PipelineMultiplier  int  `mapstructure:"pipeline_multiplier" default:"8"`

	// Memory Optimization
	MaxMemoryPolicy     string `mapstructure:"max_memory_policy" default:"allkeys-lru"`
	EnableCompression   bool   `mapstructure:"enable_compression" default:"false"`
}

// HTTPPerformanceConfig configurações otimizadas para servidor HTTP
type HTTPPerformanceConfig struct {
	// Server Timeouts
	ReadTimeout       time.Duration `mapstructure:"read_timeout" default:"10s"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout" default:"10s"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout" default:"120s"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout" default:"5s"`

	// Connection Limits
	MaxHeaderBytes int `mapstructure:"max_header_bytes" default:"1048576"` // 1MB
	MaxConnsPerIP  int `mapstructure:"max_conns_per_ip" default:"100"`

	// Compression
	EnableCompression  bool `mapstructure:"enable_compression" default:"true"`
	CompressionLevel   int  `mapstructure:"compression_level" default:"6"`
	CompressionMinSize int  `mapstructure:"compression_min_size" default:"1024"`

	// Keep-Alive
	EnableKeepAlive      bool          `mapstructure:"enable_keep_alive" default:"true"`
	KeepAlivePeriod      time.Duration `mapstructure:"keep_alive_period" default:"3m"`
	DisableKeepAlives    bool          `mapstructure:"disable_keep_alives" default:"false"`
	MaxIdleConns         int           `mapstructure:"max_idle_conns" default:"100"`
	MaxIdleConnsPerHost  int           `mapstructure:"max_idle_conns_per_host" default:"10"`

	// Rate Limiting
	EnableRateLimit    bool          `mapstructure:"enable_rate_limit" default:"true"`
	RateLimitRPS       int           `mapstructure:"rate_limit_rps" default:"100"`
	RateLimitBurst     int           `mapstructure:"rate_limit_burst" default:"200"`
	RateLimitWindow    time.Duration `mapstructure:"rate_limit_window" default:"1m"`
}

// WebSocketPerformanceConfig configurações otimizadas para WebSocket
type WebSocketPerformanceConfig struct {
	// Connection Limits
	MaxConnections        int           `mapstructure:"max_connections" default:"10000"`
	MaxConnectionsPerIP   int           `mapstructure:"max_connections_per_ip" default:"10"`
	ConnectionTimeout     time.Duration `mapstructure:"connection_timeout" default:"60s"`

	// Message Handling
	MaxMessageSize        int64         `mapstructure:"max_message_size" default:"512"`     // 512 bytes
	ReadBufferSize        int           `mapstructure:"read_buffer_size" default:"1024"`
	WriteBufferSize       int           `mapstructure:"write_buffer_size" default:"1024"`
	WriteTimeout          time.Duration `mapstructure:"write_timeout" default:"10s"`
	ReadTimeout           time.Duration `mapstructure:"read_timeout" default:"60s"`

	// Ping/Pong Configuration
	PingPeriod            time.Duration `mapstructure:"ping_period" default:"54s"`
	PongWait              time.Duration `mapstructure:"pong_wait" default:"60s"`

	// Broadcasting
	BroadcastChannelSize  int           `mapstructure:"broadcast_channel_size" default:"1000"`
	BroadcastWorkers      int           `mapstructure:"broadcast_workers" default:"10"`
	BroadcastTimeout      time.Duration `mapstructure:"broadcast_timeout" default:"5s"`

	// Connection Pool
	EnableConnectionPool  bool          `mapstructure:"enable_connection_pool" default:"true"`
	PoolCleanupInterval   time.Duration `mapstructure:"pool_cleanup_interval" default:"5m"`
}

// CachePerformanceConfig configurações otimizadas para cache
type CachePerformanceConfig struct {
	// TTL Settings
	DefaultTTL           time.Duration `mapstructure:"default_ttl" default:"1h"`
	PriceCacheTTL        time.Duration `mapstructure:"price_cache_ttl" default:"1m"`
	SessionCacheTTL      time.Duration `mapstructure:"session_cache_ttl" default:"24h"`
	AlertResultCacheTTL  time.Duration `mapstructure:"alert_result_cache_ttl" default:"5m"`
	UserDataCacheTTL     time.Duration `mapstructure:"user_data_cache_ttl" default:"15m"`

	// Memory Cache (Local)
	EnableMemoryCache    bool `mapstructure:"enable_memory_cache" default:"true"`
	MemoryCacheSize      int  `mapstructure:"memory_cache_size" default:"1000"`
	MemoryCacheCleanup   time.Duration `mapstructure:"memory_cache_cleanup" default:"10m"`

	// Cache Strategies
	EnableWriteThrough   bool `mapstructure:"enable_write_through" default:"true"`
	EnableWriteBehind    bool `mapstructure:"enable_write_behind" default:"false"`
	EnableReadThrough    bool `mapstructure:"enable_read_through" default:"true"`

	// Cache Warming
	EnableCacheWarming   bool          `mapstructure:"enable_cache_warming" default:"true"`
	WarmupInterval       time.Duration `mapstructure:"warmup_interval" default:"5m"`
	WarmupBatchSize      int           `mapstructure:"warmup_batch_size" default:"100"`

	// Invalidation
	EnableSmartInvalidation bool          `mapstructure:"enable_smart_invalidation" default:"true"`
	InvalidationDelay      time.Duration `mapstructure:"invalidation_delay" default:"1s"`
}

// AlertEnginePerformanceConfig configurações otimizadas para motor de alertas
type AlertEnginePerformanceConfig struct {
	// Processing
	MaxWorkers               int           `mapstructure:"max_workers" default:"10"`
	WorkerQueueSize          int           `mapstructure:"worker_queue_size" default:"1000"`
	ProcessingInterval       time.Duration `mapstructure:"processing_interval" default:"5s"`
	BatchSize                int           `mapstructure:"batch_size" default:"100"`

	// Performance Tuning
	EnableConcurrentEvaluation  bool          `mapstructure:"enable_concurrent_evaluation" default:"true"`
	MaxConcurrentEvaluations    int           `mapstructure:"max_concurrent_evaluations" default:"50"`
	EvaluationTimeout           time.Duration `mapstructure:"evaluation_timeout" default:"30s"`

	// Memory Management
	EnableMemoryOptimization    bool          `mapstructure:"enable_memory_optimization" default:"true"`
	GCInterval                  time.Duration `mapstructure:"gc_interval" default:"30s"`
	MaxMemoryUsage              string        `mapstructure:"max_memory_usage" default:"1GB"`

	// Throttling
	EnableThrottling            bool          `mapstructure:"enable_throttling" default:"true"`
	ThrottleThreshold           int           `mapstructure:"throttle_threshold" default:"1000"`
	ThrottleRecoveryTime        time.Duration `mapstructure:"throttle_recovery_time" default:"1m"`

	// Circuit Breaker
	EnableCircuitBreaker        bool          `mapstructure:"enable_circuit_breaker" default:"true"`
	CircuitBreakerThreshold     int           `mapstructure:"circuit_breaker_threshold" default:"10"`
	CircuitBreakerTimeout       time.Duration `mapstructure:"circuit_breaker_timeout" default:"30s"`
	CircuitBreakerRecoveryTime  time.Duration `mapstructure:"circuit_breaker_recovery_time" default:"1m"`
}

// GetDefaultPerformanceConfig retorna configuração padrão otimizada
func GetDefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		Database: DatabasePerformanceConfig{
			MaxOpenConns:           25,
			MaxIdleConns:           10,
			ConnMaxLifetime:        time.Hour,
			ConnMaxIdleTime:        30 * time.Minute,
			QueryTimeout:           30 * time.Second,
			SlowQueryThreshold:     100 * time.Millisecond,
			EnableQueryLogging:     false,
			EnablePreparedStmts:    true,
			BatchSize:              100,
			MaxBatchInsertSize:     1000,
			EnableBatchOperations:  true,
		},
		Redis: RedisPerformanceConfig{
			PoolSize:            25,
			MinIdleConns:        10,
			MaxIdleConns:        20,
			ConnMaxIdleTime:     30 * time.Minute,
			ConnMaxLifetime:     time.Hour,
			DialTimeout:         5 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			PoolTimeout:         4 * time.Second,
			EnablePipelining:    true,
			MaxPipelineSize:     100,
			PipelineMultiplier:  8,
		},
		HTTP: HTTPPerformanceConfig{
			ReadTimeout:            10 * time.Second,
			WriteTimeout:           10 * time.Second,
			IdleTimeout:            120 * time.Second,
			ReadHeaderTimeout:      5 * time.Second,
			MaxHeaderBytes:         1048576, // 1MB
			MaxConnsPerIP:          100,
			EnableCompression:      true,
			CompressionLevel:       6,
			CompressionMinSize:     1024,
			EnableKeepAlive:        true,
			KeepAlivePeriod:        3 * time.Minute,
			MaxIdleConns:           100,
			MaxIdleConnsPerHost:    10,
			EnableRateLimit:        true,
			RateLimitRPS:           100,
			RateLimitBurst:         200,
			RateLimitWindow:        time.Minute,
		},
		WebSocket: WebSocketPerformanceConfig{
			MaxConnections:        10000,
			MaxConnectionsPerIP:   10,
			ConnectionTimeout:     60 * time.Second,
			MaxMessageSize:        512,
			ReadBufferSize:        1024,
			WriteBufferSize:       1024,
			WriteTimeout:          10 * time.Second,
			ReadTimeout:           60 * time.Second,
			PingPeriod:            54 * time.Second,
			PongWait:              60 * time.Second,
			BroadcastChannelSize:  1000,
			BroadcastWorkers:      10,
			BroadcastTimeout:      5 * time.Second,
			EnableConnectionPool:  true,
			PoolCleanupInterval:   5 * time.Minute,
		},
		Cache: CachePerformanceConfig{
			DefaultTTL:              time.Hour,
			PriceCacheTTL:           time.Minute,
			SessionCacheTTL:         24 * time.Hour,
			AlertResultCacheTTL:     5 * time.Minute,
			UserDataCacheTTL:        15 * time.Minute,
			EnableMemoryCache:       true,
			MemoryCacheSize:         1000,
			MemoryCacheCleanup:      10 * time.Minute,
			EnableWriteThrough:      true,
			EnableReadThrough:       true,
			EnableCacheWarming:      true,
			WarmupInterval:          5 * time.Minute,
			WarmupBatchSize:         100,
			EnableSmartInvalidation: true,
			InvalidationDelay:       time.Second,
		},
		AlertEngine: AlertEnginePerformanceConfig{
			MaxWorkers:                  10,
			WorkerQueueSize:             1000,
			ProcessingInterval:          5 * time.Second,
			BatchSize:                   100,
			EnableConcurrentEvaluation:  true,
			MaxConcurrentEvaluations:    50,
			EvaluationTimeout:           30 * time.Second,
			EnableMemoryOptimization:    true,
			GCInterval:                  30 * time.Second,
			EnableThrottling:            true,
			ThrottleThreshold:           1000,
			ThrottleRecoveryTime:        time.Minute,
			EnableCircuitBreaker:        true,
			CircuitBreakerThreshold:     10,
			CircuitBreakerTimeout:       30 * time.Second,
			CircuitBreakerRecoveryTime:  time.Minute,
		},
	}
}

// GetProductionPerformanceConfig retorna configuração otimizada para produção
func GetProductionPerformanceConfig() *PerformanceConfig {
	config := GetDefaultPerformanceConfig()
	
	// Ajustes específicos para produção
	config.Database.MaxOpenConns = 50
	config.Database.MaxIdleConns = 25
	config.Database.EnableQueryLogging = false
	
	config.Redis.PoolSize = 50
	config.Redis.MinIdleConns = 20
	config.Redis.MaxIdleConns = 40
	
	config.HTTP.MaxConnsPerIP = 200
	config.HTTP.RateLimitRPS = 500
	config.HTTP.RateLimitBurst = 1000
	
	config.WebSocket.MaxConnections = 50000
	config.WebSocket.MaxConnectionsPerIP = 50
	config.WebSocket.BroadcastWorkers = 20
	
	config.AlertEngine.MaxWorkers = 25
	config.AlertEngine.WorkerQueueSize = 5000
	config.AlertEngine.MaxConcurrentEvaluations = 100
	
	return config
}
