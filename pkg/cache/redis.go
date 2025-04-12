package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// RedisCache implementa a interface Cache usando Redis
type RedisCache struct {
	client *redis.Client
	logger *zap.Logger
}

// NewRedisCache cria uma nova instância de RedisCache
func NewRedisCache(addr string, password string, db int, logger *zap.Logger) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verificar a conexão
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisCache{
		client: client,
		logger: logger,
	}, nil
}

// Set armazena um valor no cache
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		c.logger.Error("falha ao serializar para cache", zap.Error(err))
		return err
	}

	return c.client.Set(ctx, key, data, expiration).Err()
}

// Get recupera um valor do cache
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil // Cache miss, não é erro
		}
		c.logger.Error("falha ao recuperar do cache", zap.String("key", key), zap.Error(err))
		return false, err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		c.logger.Error("falha ao deserializar do cache", zap.String("key", key), zap.Error(err))
		return false, err
	}

	return true, nil
}

// Delete remove um valor do cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Clear remove todos os valores do cache usando padrão de chave
func (c *RedisCache) Clear(ctx context.Context) error {
	// Use padrão para limpar somente chaves relacionadas à aplicação
	keys, err := c.client.Keys(ctx, "apigateway:*").Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}

	return nil
}

// Ping verifica se o Redis está acessível
func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
