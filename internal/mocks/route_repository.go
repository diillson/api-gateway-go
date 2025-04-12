package mocks

import (
	"context"

	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/stretchr/testify/mock"
)

// MockRouteRepository é um mock para o repository.RouteRepository
type MockRouteRepository struct {
	mock.Mock
}

func (m *MockRouteRepository) GetRoutes(ctx context.Context) ([]*model.Route, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Route), args.Error(1)
}

func (m *MockRouteRepository) GetRouteByPath(ctx context.Context, path string) (*model.Route, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Route), args.Error(1)
}

func (m *MockRouteRepository) AddRoute(ctx context.Context, route *model.Route) error {
	args := m.Called(ctx, route)
	return args.Error(0)
}

func (m *MockRouteRepository) UpdateRoute(ctx context.Context, route *model.Route) error {
	args := m.Called(ctx, route)
	return args.Error(0)
}

func (m *MockRouteRepository) DeleteRoute(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

func (m *MockRouteRepository) UpdateMetrics(ctx context.Context, path string, callCount int64, totalResponseTime int64) error {
	args := m.Called(ctx, path, callCount, totalResponseTime)
	return args.Error(0)
}

func (m *MockRouteRepository) GetRoutesWithFilters(ctx context.Context, filters map[string]interface{}) ([]*model.Route, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Route), args.Error(1)
}
