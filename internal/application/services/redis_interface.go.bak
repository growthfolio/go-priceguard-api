package services

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// RedisClientInterface defines the interface for Redis operations used by services
type RedisClientInterface interface {
	ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd
	ZCard(ctx context.Context, key string) *redis.IntCmd
	ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	ZRemRangeByScore(ctx context.Context, key, min, max string) *redis.IntCmd
	ZRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.ZSliceCmd
}

// RedisClientWrapper wraps redis.Client to implement RedisClientInterface
type RedisClientWrapper struct {
	client *redis.Client
}

func NewRedisClientWrapper(client *redis.Client) RedisClientInterface {
	return &RedisClientWrapper{client: client}
}

func (w *RedisClientWrapper) ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	return w.client.ZAdd(ctx, key, members...)
}

func (w *RedisClientWrapper) ZCard(ctx context.Context, key string) *redis.IntCmd {
	return w.client.ZCard(ctx, key)
}

func (w *RedisClientWrapper) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return w.client.ZRem(ctx, key, members...)
}

func (w *RedisClientWrapper) ZRemRangeByScore(ctx context.Context, key, min, max string) *redis.IntCmd {
	return w.client.ZRemRangeByScore(ctx, key, min, max)
}

func (w *RedisClientWrapper) ZRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.ZSliceCmd {
	return w.client.ZRangeByScoreWithScores(ctx, key, opt)
}
