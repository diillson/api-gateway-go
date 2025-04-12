package cache

import (
	"context"
	"time"
)

// Cache define a interface para operações de cache
type Cache interface {
	// Set armazena um valor no cache com tempo de expiração
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	// Get recupera um valor do cache
	Get(ctx context.Context, key string, dest interface{}) (bool, error)

	// Delete remove um valor do cache
	Delete(ctx context.Context, key string) error

	// Clear remove todos os valores do cache
	Clear(ctx context.Context) error

	// Ping verifica se o cache está acessível
	Ping(ctx context.Context) error
}
