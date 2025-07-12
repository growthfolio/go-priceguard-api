package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for our application
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	Google     GoogleOAuthConfig
	Binance    BinanceConfig
	WebSocket  WebSocketConfig
	App        AppConfig
	RateLimit  RateLimitConfig
	Email      EmailConfig
	Monitoring MonitoringConfig
}

type ServerConfig struct {
	Port int
	Host string
	Mode string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type JWTConfig struct {
	Secret            string
	Expiration        time.Duration
	RefreshExpiration time.Duration
}

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type BinanceConfig struct {
	APIKey    string
	APISecret string
	TestNet   bool
}

type WebSocketConfig struct {
	Path           string
	UpdateInterval time.Duration
	MaxConnections int
}

type AppConfig struct {
	Environment        string
	LogLevel           string
	CORSAllowedOrigins []string
}

type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
}

type EmailConfig struct {
	SMTPHost string
	SMTPPort int
	From     string
	Password string
}

type MonitoringConfig struct {
	EnableMetrics  bool
	MetricsPort    int
	EnableTracing  bool
	JaegerEndpoint string
}

// LoadConfig loads configuration from environment variables and .env file
func LoadConfig() (*Config, error) {
	// Set default configuration file
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./")

	// Enable reading from environment variables
	viper.AutomaticEnv()

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	config := &Config{}

	// Load server configuration
	config.Server = ServerConfig{
		Port: getIntEnv("PORT", 8080),
		Host: getStringEnv("HOST", "localhost"),
		Mode: getStringEnv("GIN_MODE", "debug"),
	}

	// Load database configuration
	config.Database = DatabaseConfig{
		Host:     getStringEnv("DB_HOST", "localhost"),
		Port:     getIntEnv("DB_PORT", 5432),
		User:     getStringEnv("DB_USER", "postgres"),
		Password: getStringEnv("DB_PASSWORD", "password"),
		Name:     getStringEnv("DB_NAME", "priceguard"),
		SSLMode:  getStringEnv("DB_SSL_MODE", "disable"),
	}

	// Load Redis configuration
	config.Redis = RedisConfig{
		Host:     getStringEnv("REDIS_HOST", "localhost"),
		Port:     getIntEnv("REDIS_PORT", 6379),
		Password: getStringEnv("REDIS_PASSWORD", ""),
		DB:       getIntEnv("REDIS_DB", 0),
	}

	// Load JWT configuration
	jwtExpiration, err := time.ParseDuration(getStringEnv("JWT_EXPIRATION", "24h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRATION format: %w", err)
	}

	jwtRefreshExpiration, err := time.ParseDuration(getStringEnv("JWT_REFRESH_EXPIRATION", "168h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_EXPIRATION format: %w", err)
	}

	config.JWT = JWTConfig{
		Secret:            getStringEnv("JWT_SECRET", ""),
		Expiration:        jwtExpiration,
		RefreshExpiration: jwtRefreshExpiration,
	}

	// Load Google OAuth configuration
	config.Google = GoogleOAuthConfig{
		ClientID:     getStringEnv("GOOGLE_CLIENT_ID", ""),
		ClientSecret: getStringEnv("GOOGLE_CLIENT_SECRET", ""),
		RedirectURL:  getStringEnv("GOOGLE_REDIRECT_URL", ""),
	}

	// Load Binance configuration
	config.Binance = BinanceConfig{
		APIKey:    getStringEnv("BINANCE_API_KEY", ""),
		APISecret: getStringEnv("BINANCE_API_SECRET", ""),
		TestNet:   getBoolEnv("BINANCE_TESTNET", true),
	}

	// Load WebSocket configuration
	wsUpdateInterval, err := time.ParseDuration(getStringEnv("WS_UPDATE_INTERVAL", "1000ms"))
	if err != nil {
		return nil, fmt.Errorf("invalid WS_UPDATE_INTERVAL format: %w", err)
	}

	config.WebSocket = WebSocketConfig{
		Path:           getStringEnv("WS_PATH", "/ws/dashboard"),
		UpdateInterval: wsUpdateInterval,
		MaxConnections: getIntEnv("WS_MAX_CONNECTIONS", 1000),
	}

	// Load app configuration
	corsOrigins := strings.Split(getStringEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"), ",")
	config.App = AppConfig{
		Environment:        getStringEnv("APP_ENV", "development"),
		LogLevel:           getStringEnv("LOG_LEVEL", "debug"),
		CORSAllowedOrigins: corsOrigins,
	}

	// Load rate limit configuration
	config.RateLimit = RateLimitConfig{
		RequestsPerMinute: getIntEnv("RATE_LIMIT_REQUESTS_PER_MINUTE", 60),
		Burst:             getIntEnv("RATE_LIMIT_BURST", 10),
	}

	// Load email configuration
	config.Email = EmailConfig{
		SMTPHost: getStringEnv("EMAIL_SMTP_HOST", ""),
		SMTPPort: getIntEnv("EMAIL_SMTP_PORT", 587),
		From:     getStringEnv("EMAIL_FROM", ""),
		Password: getStringEnv("EMAIL_PASSWORD", ""),
	}

	// Load monitoring configuration
	config.Monitoring = MonitoringConfig{
		EnableMetrics:  getBoolEnv("ENABLE_METRICS", true),
		MetricsPort:    getIntEnv("METRICS_PORT", 9090),
		EnableTracing:  getBoolEnv("ENABLE_TRACING", false),
		JaegerEndpoint: getStringEnv("JAEGER_ENDPOINT", ""),
	}

	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate checks if all required configuration values are set
func (c *Config) Validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	// In development, allow placeholder values for Google OAuth
	if c.App.Environment != "development" {
		if c.Google.ClientID == "" || c.Google.ClientSecret == "" {
			return fmt.Errorf("Google OAuth credentials are required")
		}
	}

	return nil
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetRedisAddr returns the Redis connection address
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// Helper functions to get environment variables with defaults
func getStringEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
