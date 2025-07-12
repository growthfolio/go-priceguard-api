package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// CacheStrategy define estratégias de cache
type CacheStrategy string

const (
	WriteThrough CacheStrategy = "write_through"
	WriteBack    CacheStrategy = "write_back"
	WriteAround  CacheStrategy = "write_around"
)

// CacheLevel representa os níveis de cache
type CacheLevel int

const (
	L1Cache CacheLevel = iota // Memory cache
	L2Cache                   // Redis cache
	L3Cache                   // Database
)

// CacheItem representa um item no cache
type CacheItem struct {
	Key        string
	Value      interface{}
	ExpiredAt  time.Time
	CreatedAt  time.Time
	AccessedAt time.Time
	HitCount   int64
}

// CacheStats estatísticas do cache
type CacheStats struct {
	Hits          int64
	Misses        int64
	Sets          int64
	Deletes       int64
	Evictions     int64
	Size          int64
	HitRatio      float64
	AvgAccessTime time.Duration
}

// MemoryCache cache em memória local
type MemoryCache struct {
	data        map[string]*CacheItem
	mutex       sync.RWMutex
	maxSize     int
	stats       CacheStats
	janitor     *janitor
	stopJanitor chan bool
}

// NewMemoryCache cria novo cache em memória
func NewMemoryCache(maxSize int, cleanupInterval time.Duration) *MemoryCache {
	mc := &MemoryCache{
		data:        make(map[string]*CacheItem),
		maxSize:     maxSize,
		stopJanitor: make(chan bool),
	}

	// Iniciar janitor para limpeza automática
	mc.janitor = &janitor{
		interval: cleanupInterval,
		stop:     mc.stopJanitor,
	}
	go mc.janitor.run(mc)

	return mc
}

// Set adiciona item ao cache
func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// Verificar se precisa fazer eviction
	if len(mc.data) >= mc.maxSize {
		mc.evictLRU()
	}

	now := time.Now()
	mc.data[key] = &CacheItem{
		Key:        key,
		Value:      value,
		ExpiredAt:  now.Add(ttl),
		CreatedAt:  now,
		AccessedAt: now,
		HitCount:   0,
	}

	mc.stats.Sets++
	mc.stats.Size = int64(len(mc.data))
	return nil
}

// Get recupera item do cache
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	item, exists := mc.data[key]
	if !exists {
		mc.stats.Misses++
		return nil, false
	}

	// Verificar se expirou
	if time.Now().After(item.ExpiredAt) {
		mc.stats.Misses++
		delete(mc.data, key)
		mc.stats.Size = int64(len(mc.data))
		return nil, false
	}

	// Atualizar estatísticas de acesso
	item.AccessedAt = time.Now()
	item.HitCount++
	mc.stats.Hits++

	// Calcular hit ratio
	totalAccess := mc.stats.Hits + mc.stats.Misses
	if totalAccess > 0 {
		mc.stats.HitRatio = float64(mc.stats.Hits) / float64(totalAccess)
	}

	return item.Value, true
}

// Delete remove item do cache
func (mc *MemoryCache) Delete(key string) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	delete(mc.data, key)
	mc.stats.Deletes++
	mc.stats.Size = int64(len(mc.data))
	return nil
}

// GetStats retorna estatísticas do cache
func (mc *MemoryCache) GetStats() CacheStats {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.stats
}

// evictLRU remove o item menos recentemente usado
func (mc *MemoryCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range mc.data {
		if oldestKey == "" || item.AccessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.AccessedAt
		}
	}

	if oldestKey != "" {
		delete(mc.data, oldestKey)
		mc.stats.Evictions++
	}
}

// cleanup remove itens expirados
func (mc *MemoryCache) cleanup() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	now := time.Now()
	for key, item := range mc.data {
		if now.After(item.ExpiredAt) {
			delete(mc.data, key)
		}
	}
	mc.stats.Size = int64(len(mc.data))
}

// Close fecha o cache e para o janitor
func (mc *MemoryCache) Close() {
	close(mc.stopJanitor)
}

// janitor limpa itens expirados periodicamente
type janitor struct {
	interval time.Duration
	stop     chan bool
}

func (j *janitor) run(mc *MemoryCache) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.cleanup()
		case <-j.stop:
			return
		}
	}
}

// LayeredCache implementa cache em camadas (L1: Memory, L2: Redis)
type LayeredCache struct {
	l1Cache  *MemoryCache
	l2Cache  *redis.Client
	strategy CacheStrategy
	logger   *logrus.Logger
	metrics  *CacheMetrics
}

// CacheMetrics métricas detalhadas do cache
type CacheMetrics struct {
	L1Stats CacheStats
	L2Stats CacheStats
	mutex   sync.RWMutex
}

// NewLayeredCache cria novo cache em camadas
func NewLayeredCache(l1MaxSize int, l1CleanupInterval time.Duration,
	redisClient *redis.Client, strategy CacheStrategy, logger *logrus.Logger) *LayeredCache {

	return &LayeredCache{
		l1Cache:  NewMemoryCache(l1MaxSize, l1CleanupInterval),
		l2Cache:  redisClient,
		strategy: strategy,
		logger:   logger,
		metrics:  &CacheMetrics{},
	}
}

// Set adiciona item em ambas as camadas
func (lc *LayeredCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Serializar valor para JSON
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// L1 Cache (Memory)
	if err := lc.l1Cache.Set(key, value, ttl); err != nil {
		lc.logger.WithError(err).Warn("Failed to set L1 cache")
	}

	// L2 Cache (Redis)
	switch lc.strategy {
	case WriteThrough:
		// Escrever imediatamente no Redis
		if err := lc.l2Cache.Set(ctx, key, data, ttl).Err(); err != nil {
			lc.logger.WithError(err).Warn("Failed to set L2 cache")
			return err
		}
	case WriteBack:
		// Escrever no Redis de forma assíncrona (implementar queue)
		go func() {
			if err := lc.l2Cache.Set(context.Background(), key, data, ttl).Err(); err != nil {
				lc.logger.WithError(err).Warn("Failed to async set L2 cache")
			}
		}()
	case WriteAround:
		// Não escrever no cache, apenas no destino final
		// (não aplicável neste contexto)
	}

	lc.updateStats("set", L1Cache)
	return nil
}

// Get recupera item das camadas de cache
func (lc *LayeredCache) Get(ctx context.Context, key string, target interface{}) (bool, error) {
	// Tentar L1 Cache primeiro
	if value, found := lc.l1Cache.Get(key); found {
		if err := lc.copyValue(value, target); err == nil {
			lc.updateStats("hit", L1Cache)
			return true, nil
		}
	}

	lc.updateStats("miss", L1Cache)

	// Tentar L2 Cache (Redis)
	data, err := lc.l2Cache.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			lc.updateStats("miss", L2Cache)
			return false, nil
		}
		return false, fmt.Errorf("failed to get from L2 cache: %w", err)
	}

	// Deserializar valor
	if err := json.Unmarshal(data, target); err != nil {
		return false, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	// Promover para L1 cache
	ttl := lc.l2Cache.TTL(ctx, key).Val()
	if ttl > 0 {
		_ = lc.l1Cache.Set(key, target, ttl)
	}

	lc.updateStats("hit", L2Cache)
	return true, nil
}

// Delete remove item de ambas as camadas
func (lc *LayeredCache) Delete(ctx context.Context, key string) error {
	// Remover do L1
	_ = lc.l1Cache.Delete(key)

	// Remover do L2
	if err := lc.l2Cache.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete from L2 cache: %w", err)
	}

	lc.updateStats("delete", L1Cache)
	lc.updateStats("delete", L2Cache)
	return nil
}

// GetMetrics retorna métricas do cache
func (lc *LayeredCache) GetMetrics() CacheMetrics {
	lc.metrics.mutex.RLock()
	defer lc.metrics.mutex.RUnlock()

	return CacheMetrics{
		L1Stats: lc.l1Cache.GetStats(),
		L2Stats: lc.metrics.L2Stats,
	}
}

// Close fecha o cache em camadas
func (lc *LayeredCache) Close() error {
	lc.l1Cache.Close()
	return lc.l2Cache.Close()
}

// updateStats atualiza estatísticas do cache
func (lc *LayeredCache) updateStats(operation string, level CacheLevel) {
	lc.metrics.mutex.Lock()
	defer lc.metrics.mutex.Unlock()

	var stats *CacheStats
	switch level {
	case L1Cache:
		stats = &lc.metrics.L1Stats
	case L2Cache:
		stats = &lc.metrics.L2Stats
	default:
		return
	}

	switch operation {
	case "hit":
		stats.Hits++
	case "miss":
		stats.Misses++
	case "set":
		stats.Sets++
	case "delete":
		stats.Deletes++
	}

	// Calcular hit ratio
	totalAccess := stats.Hits + stats.Misses
	if totalAccess > 0 {
		stats.HitRatio = float64(stats.Hits) / float64(totalAccess)
	}
}

// copyValue copia valor de forma segura
func (lc *LayeredCache) copyValue(src, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

// CacheWarmer aquece o cache com dados frequentemente acessados
type CacheWarmer struct {
	cache     *LayeredCache
	warmupFns map[string]func(ctx context.Context) error
	interval  time.Duration
	batchSize int
	logger    *logrus.Logger
	stopChan  chan bool
}

// NewCacheWarmer cria novo cache warmer
func NewCacheWarmer(cache *LayeredCache, interval time.Duration, batchSize int, logger *logrus.Logger) *CacheWarmer {
	return &CacheWarmer{
		cache:     cache,
		warmupFns: make(map[string]func(ctx context.Context) error),
		interval:  interval,
		batchSize: batchSize,
		logger:    logger,
		stopChan:  make(chan bool),
	}
}

// AddWarmupFunction adiciona função de aquecimento
func (cw *CacheWarmer) AddWarmupFunction(name string, fn func(ctx context.Context) error) {
	cw.warmupFns[name] = fn
}

// Start inicia o processo de aquecimento
func (cw *CacheWarmer) Start(ctx context.Context) {
	ticker := time.NewTicker(cw.interval)
	defer ticker.Stop()

	// Aquecimento inicial
	cw.warmup(ctx)

	for {
		select {
		case <-ticker.C:
			cw.warmup(ctx)
		case <-cw.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop para o processo de aquecimento
func (cw *CacheWarmer) Stop() {
	close(cw.stopChan)
}

// warmup executa todas as funções de aquecimento
func (cw *CacheWarmer) warmup(ctx context.Context) {
	for name, fn := range cw.warmupFns {
		if err := fn(ctx); err != nil {
			cw.logger.WithError(err).WithField("warmer", name).Warn("Cache warmup failed")
		} else {
			cw.logger.WithField("warmer", name).Debug("Cache warmup completed")
		}
	}
}
