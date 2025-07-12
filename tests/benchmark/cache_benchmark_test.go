package benchmark

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

// BenchmarkRedisOperations testa performance de operações Redis
func BenchmarkRedisOperations(b *testing.B) {
	client := setupRedisClient(b)
	defer cleanupRedis(client)

	b.Run("SetString", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("test_key_%d", i)
			value := fmt.Sprintf("test_value_%d", i)
			err := client.Set(context.Background(), key, value, time.Hour).Err()
			require.NoError(b, err)
		}
	})

	b.Run("GetString", func(b *testing.B) {
		// Setup - criar algumas keys primeiro
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("get_test_key_%d", i)
			value := fmt.Sprintf("get_test_value_%d", i)
			_ = client.Set(context.Background(), key, value, time.Hour).Err()
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("get_test_key_%d", i%1000)
			_, err := client.Get(context.Background(), key).Result()
			require.NoError(b, err)
		}
	})

	b.Run("PipelineOperations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pipe := client.Pipeline()
			
			// Adicionar múltiplas operações ao pipeline
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("pipe_key_%d_%d", i, j)
				value := fmt.Sprintf("pipe_value_%d_%d", i, j)
				pipe.Set(context.Background(), key, value, time.Hour)
			}
			
			_, err := pipe.Exec(context.Background())
			require.NoError(b, err)
		}
	})

	b.Run("ConcurrentReads", func(b *testing.B) {
		// Setup
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("concurrent_key_%d", i)
			value := fmt.Sprintf("concurrent_value_%d", i)
			_ = client.Set(context.Background(), key, value, time.Hour).Err()
		}

		b.ResetTimer()
		var wg sync.WaitGroup
		concurrency := 10

		for i := 0; i < b.N; i++ {
			wg.Add(concurrency)
			for j := 0; j < concurrency; j++ {
				go func(worker int) {
					defer wg.Done()
					key := fmt.Sprintf("concurrent_key_%d", worker%100)
					_, err := client.Get(context.Background(), key).Result()
					require.NoError(b, err)
				}(j)
			}
			wg.Wait()
		}
	})

	b.Run("ListOperations", func(b *testing.B) {
		listKey := "benchmark_list"
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			value := fmt.Sprintf("list_item_%d", i)
			err := client.LPush(context.Background(), listKey, value).Err()
			require.NoError(b, err)
		}
	})

	b.Run("HashOperations", func(b *testing.B) {
		hashKey := "benchmark_hash"
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			field := fmt.Sprintf("field_%d", i)
			value := fmt.Sprintf("hash_value_%d", i)
			err := client.HSet(context.Background(), hashKey, field, value).Err()
			require.NoError(b, err)
		}
	})
}

// BenchmarkConnectionPooling testa performance com diferentes configurações de pool
func BenchmarkRedisConnectionPooling(b *testing.B) {
	configs := []struct {
		name     string
		poolSize int
		minIdle  int
	}{
		{"Small_Pool", 10, 5},
		{"Medium_Pool", 25, 10},
		{"Large_Pool", 50, 20},
	}

	for _, config := range configs {
		b.Run(config.name, func(b *testing.B) {
			client := setupRedisClientWithConfig(b, config.poolSize, config.minIdle)
			defer cleanupRedis(client)

			b.ResetTimer()
			var wg sync.WaitGroup
			concurrency := 20

			for i := 0; i < b.N; i++ {
				wg.Add(concurrency)
				for j := 0; j < concurrency; j++ {
					go func(iteration, worker int) {
						defer wg.Done()
						key := fmt.Sprintf("pool_test_%d_%d_%d", config.poolSize, iteration, worker)
						value := fmt.Sprintf("pool_value_%d_%d_%d", config.poolSize, iteration, worker)
						err := client.Set(context.Background(), key, value, time.Minute).Err()
						require.NoError(b, err)
					}(i, j)
				}
				wg.Wait()
			}
		})
	}
}

// BenchmarkCacheStrategies testa diferentes estratégias de cache
func BenchmarkCacheStrategies(b *testing.B) {
	client := setupRedisClient(b)
	defer cleanupRedis(client)

	// Cache de preços de crypto
	b.Run("CryptoPriceCache", func(b *testing.B) {
		symbols := []string{"BTCUSDT", "ETHUSDT", "ADAUSDT", "BNBUSDT", "SOLUSDT"}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			symbol := symbols[i%len(symbols)]
			key := fmt.Sprintf("price:%s", symbol)
			price := fmt.Sprintf("%.2f", float64(50000+i))
			
			// Set com TTL de 1 minuto (cache de preços)
			err := client.Set(context.Background(), key, price, time.Minute).Err()
			require.NoError(b, err)
		}
	})

	b.Run("UserSessionCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sessionID := fmt.Sprintf("session:%d", i)
			userID := fmt.Sprintf("user-%d", i%1000)
			
			// Set com TTL de 24 horas (cache de sessão)
			err := client.Set(context.Background(), sessionID, userID, 24*time.Hour).Err()
			require.NoError(b, err)
		}
	})

	b.Run("AlertResultCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			alertID := fmt.Sprintf("alert_result:%d", i)
			result := fmt.Sprintf(`{"triggered": %t, "timestamp": %d}`, i%2 == 0, time.Now().Unix())
			
			// Set com TTL de 5 minutos (cache de resultado de alerta)
			err := client.Set(context.Background(), alertID, result, 5*time.Minute).Err()
			require.NoError(b, err)
		}
	})
}

// BenchmarkMemoryCache testa cache em memória local vs Redis
func BenchmarkMemoryVsRedisCache(b *testing.B) {
	client := setupRedisClient(b)
	defer cleanupRedis(client)

	// Cache em memória simples
	memCache := make(map[string]string)
	var memMutex sync.RWMutex

	b.Run("MemoryCache_Write", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("mem_key_%d", i)
			value := fmt.Sprintf("mem_value_%d", i)
			
			memMutex.Lock()
			memCache[key] = value
			memMutex.Unlock()
		}
	})

	b.Run("MemoryCache_Read", func(b *testing.B) {
		// Setup
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("mem_read_key_%d", i)
			value := fmt.Sprintf("mem_read_value_%d", i)
			memCache[key] = value
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("mem_read_key_%d", i%1000)
			
			memMutex.RLock()
			_ = memCache[key]
			memMutex.RUnlock()
		}
	})

	b.Run("RedisCache_Write", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("redis_key_%d", i)
			value := fmt.Sprintf("redis_value_%d", i)
			
			err := client.Set(context.Background(), key, value, time.Hour).Err()
			require.NoError(b, err)
		}
	})

	b.Run("RedisCache_Read", func(b *testing.B) {
		// Setup
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("redis_read_key_%d", i)
			value := fmt.Sprintf("redis_read_value_%d", i)
			_ = client.Set(context.Background(), key, value, time.Hour).Err()
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("redis_read_key_%d", i%1000)
			_, err := client.Get(context.Background(), key).Result()
			require.NoError(b, err)
		}
	})
}

// Funções auxiliares
func setupRedisClient(b *testing.B) *redis.Client {
	return setupRedisClientWithConfig(b, 25, 10)
}

func setupRedisClientWithConfig(b *testing.B, poolSize, minIdleConns int) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           1, // Usar DB 1 para benchmarks
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
	})

	// Testar conexão
	_, err := client.Ping(context.Background()).Result()
	require.NoError(b, err)

	return client
}

func cleanupRedis(client *redis.Client) {
	// Limpar dados de teste
	client.FlushDB(context.Background())
	client.Close()
}
