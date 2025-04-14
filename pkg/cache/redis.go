package cache

import (
	"context"
	"encoding/json"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// RedisCache implementa a interface Cache usando Redis
type RedisCache struct {
	client *redis.Client
	logger *zap.Logger
	tracer trace.Tracer
}

// NewRedisCache cria uma nova instância de RedisCache
func NewRedisCache(addr string, password string, db int, logger *zap.Logger) (*RedisCache, error) {
	// Obter tracer para o cache Redis
	tracer := otel.GetTracerProvider().Tracer("api-gateway.cache.redis")

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Criar span para verificar a conexão Redis
	ctx, span := tracer.Start(
		ctx,
		"RedisCache.Init",
		trace.WithAttributes(
			attribute.String("redis.addr", addr),
			attribute.Int("redis.db", db),
			attribute.Bool("redis.password_set", password != ""),
		),
	)
	defer span.End()

	// Verificar a conexão
	if err := client.Ping(ctx).Err(); err != nil {
		span.SetStatus(codes.Error, "connection failure")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)
		return nil, err
	}

	span.SetStatus(codes.Ok, "connection successful")

	return &RedisCache{
		client: client,
		logger: logger,
		tracer: tracer,
	}, nil
}

func NewRedisClientWithConfig(config *redis.Options, logger *zap.Logger) (*redis.Client, error) {
	client := redis.NewClient(config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verificar a conexão
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("Falha ao conectar ao Redis",
			zap.String("addr", config.Addr),
			zap.Error(err))
		return nil, err
	}

	logger.Info("Conexão com Redis estabelecida com sucesso",
		zap.String("addr", config.Addr),
		zap.Int("db", config.DB))

	return client, nil
}

// Set armazena um valor no cache
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// Criar span para a operação
	ctx, span := c.tracer.Start(
		ctx,
		"RedisCache.Set",
		trace.WithAttributes(
			attribute.String("cache.key", key),
			attribute.String("cache.operation", "set"),
			attribute.Int64("cache.expiration_ms", expiration.Milliseconds()),
		),
	)
	defer span.End()

	data, err := json.Marshal(value)
	if err != nil {
		c.logger.Error("falha ao serializar para cache", zap.Error(err))
		span.SetStatus(codes.Error, "serialization failure")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)

		return err
	}

	// Registrar tamanho dos dados armazenados
	span.SetAttributes(attribute.Int("cache.data_size_bytes", len(data)))

	// Armazenar no Redis
	if err := c.client.Set(ctx, key, data, expiration).Err(); err != nil {
		c.logger.Error("falha ao armazenar no Redis",
			zap.String("key", key),
			zap.Error(err))

		span.SetStatus(codes.Error, "redis error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)

		return err
	}

	// Operação bem-sucedida
	span.SetStatus(codes.Ok, "")
	return nil
}

// Get recupera um valor do cache
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	// Criar span para a operação
	ctx, span := c.tracer.Start(
		ctx,
		"RedisCache.Get",
		trace.WithAttributes(
			attribute.String("cache.key", key),
			attribute.String("cache.operation", "get"),
		),
	)
	defer span.End()

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Cache miss não é erro, é comportamento normal
			span.SetStatus(codes.Ok, "cache miss")
			span.SetAttributes(attribute.Bool("cache.hit", false))
			return false, nil // Cache miss, não é erro
		}
		c.logger.Error("falha ao recuperar do cache",
			zap.String("key", key),
			zap.Error(err))

		span.SetStatus(codes.Error, "redis error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.Bool("cache.hit", false),
			attribute.String("error.message", err.Error()),
		)

		return false, err
	}

	// Registrar tamanho dos dados recuperados
	span.SetAttributes(
		attribute.Bool("cache.hit", true),
		attribute.Int("cache.data_size_bytes", len(data)),
	)

	// Deserializar os dados
	if err := json.Unmarshal(data, dest); err != nil {
		c.logger.Error("falha ao deserializar do cache",
			zap.String("key", key),
			zap.Error(err))

		span.SetStatus(codes.Error, "deserialization failure")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)

		return false, err
	}

	// Operação bem-sucedida
	span.SetStatus(codes.Ok, "cache hit")
	return true, nil
}

// Delete remove um valor do cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	// Criar span para a operação
	ctx, span := c.tracer.Start(
		ctx,
		"RedisCache.Delete",
		trace.WithAttributes(
			attribute.String("cache.key", key),
			attribute.String("cache.operation", "delete"),
		),
	)
	defer span.End()
	result, err := c.client.Del(ctx, key).Result()
	if err != nil {
		c.logger.Error("falha ao remover do cache",
			zap.String("key", key),
			zap.Error(err))

		span.SetStatus(codes.Error, "redis error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)

		return err
	}

	// Registrar se a chave foi encontrada e removida
	span.SetAttributes(attribute.Int64("cache.keys_removed", result))

	// Operação bem-sucedida
	span.SetStatus(codes.Ok, "")
	return nil
}

// Clear remove todos os valores do cache
func (c *RedisCache) Clear(ctx context.Context) error {
	// usar padrão "apigateway:*" como default
	return c.ClearPattern(ctx, "apigateway:*")
}

// ClearPattern remove valores do cache por padrão
func (c *RedisCache) ClearPattern(ctx context.Context, pattern string) error {
	// Criar span para a operação
	ctx, span := c.tracer.Start(
		ctx,
		"RedisCache.ClearPattern",
		trace.WithAttributes(
			attribute.String("cache.operation", "clear"),
			attribute.String("cache.pattern", pattern),
		),
	)
	defer span.End()

	// Obter todas as chaves correspondentes ao padrão
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		c.logger.Error("falha ao listar chaves do cache", zap.Error(err))

		span.SetStatus(codes.Error, "redis error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)

		return err
	}

	// Registrar número de chaves encontradas
	span.SetAttributes(attribute.Int("cache.keys_found", len(keys)))

	if len(keys) > 0 {
		// Remover as chaves encontradas
		result, err := c.client.Del(ctx, keys...).Result()
		if err != nil {
			c.logger.Error("falha ao remover chaves do cache",
				zap.Int("count", len(keys)),
				zap.Error(err))

			span.SetStatus(codes.Error, "redis delete error")
			span.SetAttributes(
				attribute.Bool("error", true),
				attribute.String("error.message", err.Error()),
			)

			return err
		}

		// Registrar número de chaves removidas
		span.SetAttributes(attribute.Int64("cache.keys_removed", result))
	}

	// Operação bem-sucedida
	span.SetStatus(codes.Ok, "")
	return nil
}

// Ping verifica se o Redis está acessível
func (c *RedisCache) Ping(ctx context.Context) error {
	// Criar span para a operação
	ctx, span := c.tracer.Start(
		ctx,
		"RedisCache.Ping",
		trace.WithAttributes(
			attribute.String("cache.operation", "ping"),
		),
	)
	defer span.End()

	// Tentar fazer ping no Redis
	if err := c.client.Ping(ctx).Err(); err != nil {
		c.logger.Error("falha ao fazer ping no Redis", zap.Error(err))

		span.SetStatus(codes.Error, "redis ping failure")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)

		return err
	}

	// Operação bem-sucedida
	span.SetStatus(codes.Ok, "")
	return nil
}
