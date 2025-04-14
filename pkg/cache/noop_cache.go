package cache

import (
	"context"
	"time"
)

// NoOpCache é uma implementação de Cache que não faz nada
// Usada quando o cache está desabilitado na configuração
type NoOpCache struct{}

// Set no-op: não faz nada
func (c *NoOpCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return nil
}

// Get no-op: sempre retorna cache miss
func (c *NoOpCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	return false, nil
}

// Delete no-op: não faz nada
func (c *NoOpCache) Delete(ctx context.Context, key string) error {
	return nil
}

// Clear no-op: não faz nada
func (c *NoOpCache) Clear(ctx context.Context) error {
	return nil
}

// Ping no-op: sempre retorna sucesso
func (c *NoOpCache) Ping(ctx context.Context) error {
	return nil
}
