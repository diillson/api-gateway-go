package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockCache é um mock para a interface Cache
type MockCache struct {
	mock.Mock
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *MockCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	args := m.Called(ctx, key, dest)

	// Se o terceiro argumento foi fornecido, ele é uma função para modificar dest
	if fn, ok := args.Get(2).(func(interface{})); ok && fn != nil {
		fn(dest)
	}

	return args.Bool(0), args.Error(1)
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCache) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCache) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
