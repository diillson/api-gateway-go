package cache

import (
	"context"
	"encoding/json"
	"github.com/diillson/api-gateway-go/internal/infra/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
	tracer  trace.Tracer
}

// NewMemoryCache cria uma nova instância de MemoryCache
func NewMemoryCache(defaultExpiration, cleanupInterval time.Duration, metrics *metrics.APIMetrics, logger *zap.Logger) *MemoryCache {
	// Obter tracer para o cache
	tracer := otel.GetTracerProvider().Tracer("api-gateway.cache.memory")
	return &MemoryCache{
		cache:   cache.New(defaultExpiration, cleanupInterval),
		logger:  logger,
		metrics: metrics,
		tracer:  tracer,
	}
}

// Set armazena um valor no cache
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// Criar span para a operação de cache
	ctx, span := c.tracer.Start(
		ctx,
		"MemoryCache.Set",
		trace.WithAttributes(
			attribute.String("cache.key", key),
			attribute.String("cache.operation", "set"),
			attribute.Int64("cache.expiration_ms", expiration.Milliseconds()),
		),
	)
	defer span.End()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache.Set(key, value, expiration)

	// Finalizar span com sucesso
	span.SetStatus(codes.Ok, "")
	return nil
}

// Get recupera um valor do cache
func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	// Criar span para a operação de cache
	ctx, span := c.tracer.Start(
		ctx,
		"MemoryCache.Get",
		trace.WithAttributes(
			attribute.String("cache.key", key),
			attribute.String("cache.operation", "get"),
		),
	)
	defer span.End()

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	value, found := c.cache.Get(key)
	if !found {
		atomic.AddInt64(&c.misses, 1)
		updateCacheMetrics(c.hits, c.misses, "memory", c.metrics)

		// Registrar cache miss no span
		span.SetAttributes(attribute.Bool("cache.hit", false))
		span.SetStatus(codes.Ok, "cache miss")
		return false, nil
	}

	atomic.AddInt64(&c.hits, 1)
	updateCacheMetrics(c.hits, c.misses, "memory", c.metrics)

	// Registrar cache hit no span
	span.SetAttributes(attribute.Bool("cache.hit", true))

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

		// Registrar erro no span
		span.SetStatus(codes.Error, "serialization error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)
		return true, err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		c.logger.Error("falha ao deserializar para o destino", zap.Error(err))

		// Registrar erro no span
		span.SetStatus(codes.Error, "deserialization error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)
		return true, err
	}

	span.SetStatus(codes.Ok, "")
	return true, nil
}

// Delete remove um valor do cache
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	// Criar span para a operação de cache
	ctx, span := c.tracer.Start(
		ctx,
		"MemoryCache.Delete",
		trace.WithAttributes(
			attribute.String("cache.key", key),
			attribute.String("cache.operation", "delete"),
		),
	)
	defer span.End()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache.Delete(key)
	span.SetStatus(codes.Ok, "")
	return nil
}

// Clear remove todos os valores do cache
func (c *MemoryCache) Clear(ctx context.Context) error {
	// Criar span para a operação de cache
	ctx, span := c.tracer.Start(
		ctx,
		"MemoryCache.Clear",
		trace.WithAttributes(
			attribute.String("cache.clear", "true"),
			attribute.String("cache.operation", "clear"),
		),
	)
	defer span.End()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache.Flush()
	span.SetStatus(codes.Ok, "")
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
