package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitConfig configuração do rate limiting
type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize         int
	KeyGenerator      func(*gin.Context) string
}

// DefaultRateLimitConfig configuração padrão
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
		KeyGenerator: func(c *gin.Context) string {
			// Rate limit por IP
			return "rate_limit:" + c.ClientIP()
		},
	}
}

// AuthenticatedUserRateLimitConfig configuração para usuários autenticados
func AuthenticatedUserRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute: 300, // Mais permissivo para usuários autenticados
		BurstSize:         20,
		KeyGenerator: func(c *gin.Context) string {
			// Rate limit por usuário autenticado
			if userID, exists := c.Get("user_id"); exists {
				return fmt.Sprintf("rate_limit:user:%v", userID)
			}
			// Fallback para IP se não autenticado
			return "rate_limit:" + c.ClientIP()
		},
	}
}

// RateLimitMiddleware middleware de rate limiting usando Redis
func RateLimitMiddleware(rdb *redis.Client, config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := config.KeyGenerator(c)
		ctx := context.Background()

		// Implementa sliding window rate limiting
		now := time.Now()
		window := time.Minute

		// Remove entradas antigas
		cutoff := now.Add(-window)
		rdb.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(cutoff.UnixNano(), 10))

		// Conta requests no window atual
		count, err := rdb.ZCard(ctx, key).Result()
		if err != nil {
			// Em caso de erro do Redis, permite a requisição (fail open)
			c.Next()
			return
		}

		// Verifica se excedeu o limite
		if int(count) >= config.RequestsPerMinute {
			c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerMinute))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(now.Add(window).Unix(), 10))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     fmt.Sprintf("Maximum %d requests per minute allowed", config.RequestsPerMinute),
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		// Adiciona a requisição atual
		rdb.ZAdd(ctx, key, redis.Z{
			Score:  float64(now.UnixNano()),
			Member: now.UnixNano(),
		})

		// Define TTL para a key
		rdb.Expire(ctx, key, window)

		// Headers de rate limit
		remaining := config.RequestsPerMinute - int(count) - 1
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(now.Add(window).Unix(), 10))

		c.Next()
	}
}
