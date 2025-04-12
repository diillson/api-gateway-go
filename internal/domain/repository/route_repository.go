package repository

import (
	"context"
	"errors"

	"github.com/diillson/api-gateway-go/internal/domain/model"
)

var (
	ErrRouteNotFound = errors.New("route not found")
	ErrRouteExists   = errors.New("route already exists")
)

// RouteRepository define a interface para armazenamento de rotas
type RouteRepository interface {
	// GetRoutes retorna todas as rotas ativas
	GetRoutes(ctx context.Context) ([]*model.Route, error)

	// GetRouteByPath obtém uma rota específica pelo caminho
	GetRouteByPath(ctx context.Context, path string) (*model.Route, error)

	// AddRoute adiciona uma nova rota
	AddRoute(ctx context.Context, route *model.Route) error

	// UpdateRoute atualiza uma rota existente
	UpdateRoute(ctx context.Context, route *model.Route) error

	// DeleteRoute remove uma rota pelo caminho
	DeleteRoute(ctx context.Context, path string) error

	// UpdateMetrics atualiza as métricas de uma rota
	UpdateMetrics(ctx context.Context, path string, callCount int64, totalResponseTime int64) error

	// GetRoutesWithFilters obtém rotas com filtros aplicados (opcional)
	GetRoutesWithFilters(ctx context.Context, filters map[string]interface{}) ([]*model.Route, error)
}
