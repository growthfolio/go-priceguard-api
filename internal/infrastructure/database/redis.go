package database

import (
	"context"
	"fmt"
	"time"

	"github.com/felipe-macedo/go-priceguard-api/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// RedisClient wraps the Redis client with additional functionality
type RedisClient struct {
	client *redis.Client
	logger *logrus.Logger
}

// NewRedisClient creates a new Redis client connection
func NewRedisClient(cfg *config.Config, logger *logrus.Logger) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisAddr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Connected to Redis successfully")

	return &RedisClient{
		client: rdb,
		logger: logger,
	}, nil
}

// GetClient returns the underlying Redis client
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Ping tests the Redis connection
func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Cache methods for crypto data
func (r *RedisClient) SetCryptoData(ctx context.Context, symbol string, data interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("crypto:data:%s", symbol)
	return r.client.Set(ctx, key, data, ttl).Err()
}

func (r *RedisClient) GetCryptoData(ctx context.Context, symbol string) (string, error) {
	key := fmt.Sprintf("crypto:data:%s", symbol)
	return r.client.Get(ctx, key).Result()
}

// Cache methods for technical indicators
func (r *RedisClient) SetTechnicalIndicator(ctx context.Context, symbol, timeframe, indicator string, data interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("indicator:%s:%s:%s", symbol, timeframe, indicator)
	return r.client.Set(ctx, key, data, ttl).Err()
}

func (r *RedisClient) GetTechnicalIndicator(ctx context.Context, symbol, timeframe, indicator string) (string, error) {
	key := fmt.Sprintf("indicator:%s:%s:%s", symbol, timeframe, indicator)
	return r.client.Get(ctx, key).Result()
}

// Session management methods
func (r *RedisClient) SetSession(ctx context.Context, sessionID string, userID string, ttl time.Duration) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return r.client.Set(ctx, key, userID, ttl).Err()
}

func (r *RedisClient) GetSession(ctx context.Context, sessionID string) (string, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	return r.client.Get(ctx, key).Result()
}

func (r *RedisClient) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return r.client.Del(ctx, key).Err()
}

// JWT blacklist methods
func (r *RedisClient) BlacklistToken(ctx context.Context, tokenHash string, ttl time.Duration) error {
	key := fmt.Sprintf("blacklist:%s", tokenHash)
	return r.client.Set(ctx, key, "1", ttl).Err()
}

func (r *RedisClient) IsTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error) {
	key := fmt.Sprintf("blacklist:%s", tokenHash)
	result, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return result == "1", nil
}

// Rate limiting methods
func (r *RedisClient) IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error) {
	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)

	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, rateLimitKey)
	pipe.Expire(ctx, rateLimitKey, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return incr.Val(), nil
}

// WebSocket connection tracking
func (r *RedisClient) AddWebSocketConnection(ctx context.Context, userID, connectionID string) error {
	key := fmt.Sprintf("ws_connections:%s", userID)
	return r.client.SAdd(ctx, key, connectionID).Err()
}

func (r *RedisClient) RemoveWebSocketConnection(ctx context.Context, userID, connectionID string) error {
	key := fmt.Sprintf("ws_connections:%s", userID)
	return r.client.SRem(ctx, key, connectionID).Err()
}

func (r *RedisClient) GetWebSocketConnections(ctx context.Context, userID string) ([]string, error) {
	key := fmt.Sprintf("ws_connections:%s", userID)
	return r.client.SMembers(ctx, key).Result()
}

// Pub/Sub for real-time updates
func (r *RedisClient) PublishCryptoUpdate(ctx context.Context, symbol string, data interface{}) error {
	channel := fmt.Sprintf("crypto_updates:%s", symbol)
	return r.client.Publish(ctx, channel, data).Err()
}

func (r *RedisClient) SubscribeToCryptoUpdates(ctx context.Context, symbols ...string) *redis.PubSub {
	channels := make([]string, len(symbols))
	for i, symbol := range symbols {
		channels[i] = fmt.Sprintf("crypto_updates:%s", symbol)
	}
	return r.client.Subscribe(ctx, channels...)
}

// Cache invalidation
func (r *RedisClient) InvalidateCryptoCache(ctx context.Context, symbol string) error {
	pattern := fmt.Sprintf("crypto:data:%s*", symbol)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return r.client.Del(ctx, keys...).Err()
	}
	return nil
}

func (r *RedisClient) InvalidateIndicatorCache(ctx context.Context, symbol string) error {
	pattern := fmt.Sprintf("indicator:%s:*", symbol)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return r.client.Del(ctx, keys...).Err()
	}
	return nil
}
