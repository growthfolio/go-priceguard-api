package cache_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/growthfolio/go-priceguard-api/internal/infrastructure/cache"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func newLayeredCache(t *testing.T) (*cache.LayeredCache, *miniredis.Miniredis, func()) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	lc := cache.NewLayeredCache(10, time.Second, rdb, cache.WriteThrough, logger)
	cleanup := func() {
		lc.Close()
		s.Close()
	}
	return lc, s, cleanup
}

func TestLayeredCache_SetGetWithTTL(t *testing.T) {
	lc, srv, cleanup := newLayeredCache(t)
	defer cleanup()
	ctx := context.Background()

	key := "ttl_key"
	value := map[string]string{"foo": "bar"}
	ttl := 50 * time.Millisecond

	err := lc.Set(ctx, key, value, ttl)
	assert.NoError(t, err)

	var out map[string]string
	found, err := lc.Get(ctx, key, &out)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, value, out)

	time.Sleep(ttl + 20*time.Millisecond)
	srv.FastForward(ttl + time.Millisecond)

	found, err = lc.Get(ctx, key, &out)
	assert.NoError(t, err)
	assert.False(t, found)

	metrics := lc.GetMetrics()
	assert.Equal(t, int64(1), metrics.L1Stats.Sets)
	assert.Equal(t, int64(1), metrics.L1Stats.Hits)
	assert.Equal(t, int64(1), metrics.L1Stats.Misses)
	assert.Equal(t, int64(1), metrics.L2Stats.Misses)
}

func TestLayeredCache_PromotionFromRedis(t *testing.T) {
	s := miniredis.RunT(t)
	key := "promo_key"
	value := map[string]string{"foo": "bar"}
	b, _ := json.Marshal(value)
	s.Set(key, string(b))
	ttl := time.Minute
	s.SetTTL(key, ttl)

	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	lc := cache.NewLayeredCache(10, time.Second, rdb, cache.WriteThrough, logger)
	defer func() {
		lc.Close()
		s.Close()
	}()

	ctx := context.Background()
	var out map[string]string
	found, err := lc.Get(ctx, key, &out)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, value, out)

	var out2 map[string]string
	found, err = lc.Get(ctx, key, &out2)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, value, out2)

	metrics := lc.GetMetrics()
	assert.Equal(t, int64(1), metrics.L1Stats.Hits)
	assert.Equal(t, int64(1), metrics.L1Stats.Misses)
	assert.Equal(t, int64(1), metrics.L2Stats.Hits)
	assert.Equal(t, int64(0), metrics.L2Stats.Misses)
}
