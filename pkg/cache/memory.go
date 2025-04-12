package cache

import (
	"context"
	"encoding/json"
	"github.com/diillson/api-gateway-go/internal/infra/metrics"
	"sync"
	"sync/atomic"
	"time"

	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// MemoryCache implementa a interface Cache usando armazenamento em memória
type MemoryCache struct {
	cache   *cache.Cache
	mutex   sync.RWMutex
	logger  *zap.Logger
	hits    int64
	misses  int64
	metrics *metrics.APIMetrics
}

// NewMemoryCache cria uma nova instância de MemoryCache
func NewMemoryCache(defaultExpiration, cleanupInterval time.Duration, metrics *metrics.APIMetrics, logger *zap.Logger) *MemoryCache {
	return &MemoryCache{
		cache:   cache.New(defaultExpiration, cleanupInterval),
		logger:  logger,
		metrics: metrics,
	}
}

// Set armazena um valor no cache
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache.Set(key, value, expiration)
	return nil
}

// Get recupera um valor do cache
func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	value, found := c.cache.Get(key)
	if !found {
		atomic.AddInt64(&c.misses, 1)
		updateCacheMetrics(c.hits, c.misses, "memory", c.metrics)
		return false, nil
	}

	atomic.AddInt64(&c.hits, 1)
	updateCacheMetrics(c.hits, c.misses, "memory", c.metrics)

	// Para tipos simples, atribuir diretamente
	switch dest := dest.(type) {
	case *string:
		if str, ok := value.(string); ok {
			*dest = str
			return true, nil
		}
	case *int:
		if i, ok := value.(int); ok {
			*dest = i
			return true, nil
		}
	case *bool:
		if b, ok := value.(bool); ok {
			*dest = b
			return true, nil
		}
	case *float64:
		if f, ok := value.(float64); ok {
			*dest = f
			return true, nil
		}
	}

	// Para estruturas, usar JSON como intermediário
	data, err := json.Marshal(value)
	if err != nil {
		c.logger.Error("falha ao serializar do cache", zap.Error(err))
		return true, err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		c.logger.Error("falha ao deserializar para o destino", zap.Error(err))
		return true, err
	}

	return true, nil
}

// Delete remove um valor do cache
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache.Delete(key)
	return nil
}

// Clear remove todos os valores do cache
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache.Flush()
	return nil
}

// Ping verifica se o cache está funcionando
func (c *MemoryCache) Ping(ctx context.Context) error {
	return nil // O cache em memória está sempre disponível
}

// Função auxiliar para atualizar métricas de cache
func updateCacheMetrics(hits, misses int64, cacheType string, metrics *metrics.APIMetrics) {
	if metrics == nil {
		return
	}

	total := hits + misses
	if total > 0 {
		hitRatio := float64(hits) / float64(total)
		metrics.UpdateCacheHitRatio(cacheType, hitRatio)
	}
}
