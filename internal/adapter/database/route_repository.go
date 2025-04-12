package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/diillson/api-gateway-go/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RouteRepository implementa repository.RouteRepository
type RouteRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewRouteRepository cria um novo repositório de rotas
func NewRouteRepository(db *gorm.DB, logger *zap.Logger) repository.RouteRepository {
	return &RouteRepository{
		db:     db,
		logger: logger,
	}
}

// GetRoutes retorna todas as rotas ativas
func (r *RouteRepository) GetRoutes(ctx context.Context) ([]*model.Route, error) {
	var entities []model.RouteEntity

	if err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&entities).Error; err != nil {
		r.logger.Error("falha ao buscar rotas", zap.Error(err))
		return nil, fmt.Errorf("falha ao buscar rotas: %w", err)
	}

	routes := make([]*model.Route, 0, len(entities))
	for _, entity := range entities {
		route, err := entityToModel(&entity)
		if err != nil {
			r.logger.Error("falha ao converter entidade para modelo", zap.Error(err))
			continue
		}
		routes = append(routes, route)
	}

	return routes, nil
}

// GetRouteByPath obtém uma rota específica pelo caminho
func (r *RouteRepository) GetRouteByPath(ctx context.Context, path string) (*model.Route, error) {
	var entity model.RouteEntity

	if err := r.db.WithContext(ctx).Where("path = ?", path).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrRouteNotFound
		}
		r.logger.Error("falha ao buscar rota por caminho", zap.String("path", path), zap.Error(err))
		return nil, fmt.Errorf("falha ao buscar rota: %w", err)
	}

	return entityToModel(&entity)
}

// AddRoute adiciona uma nova rota
func (r *RouteRepository) AddRoute(ctx context.Context, route *model.Route) error {
	entity, err := modelToEntity(route)
	if err != nil {
		return fmt.Errorf("falha ao converter modelo para entidade: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		r.logger.Error("falha ao adicionar rota", zap.String("path", route.Path), zap.Error(err))
		return fmt.Errorf("falha ao adicionar rota: %w", err)
	}

	return nil
}

// UpdateRoute atualiza uma rota existente
func (r *RouteRepository) UpdateRoute(ctx context.Context, route *model.Route) error {
	entity, err := modelToEntity(route)
	if err != nil {
		return fmt.Errorf("falha ao converter modelo para entidade: %w", err)
	}

	if err := r.db.WithContext(ctx).Where("path = ?", route.Path).Updates(entity).Error; err != nil {
		r.logger.Error("falha ao atualizar rota", zap.String("path", route.Path), zap.Error(err))
		return fmt.Errorf("falha ao atualizar rota: %w", err)
	}

	return nil
}

// DeleteRoute remove uma rota pelo caminho
func (r *RouteRepository) DeleteRoute(ctx context.Context, path string) error {
	result := r.db.WithContext(ctx).Where("path = ?", path).Delete(&model.RouteEntity{})

	if result.Error != nil {
		r.logger.Error("falha ao excluir rota", zap.String("path", path), zap.Error(result.Error))
		return fmt.Errorf("falha ao excluir rota: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return repository.ErrRouteNotFound
	}

	return nil
}

// UpdateMetrics atualiza as métricas de uma rota
func (r *RouteRepository) UpdateMetrics(ctx context.Context, path string, callCount int64, totalResponseTime int64) error {
	result := r.db.WithContext(ctx).Model(&model.RouteEntity{}).
		Where("path = ?", path).
		Updates(map[string]interface{}{
			"call_count":      callCount,
			"total_response":  totalResponseTime,
			"last_updated_at": time.Now(),
		})

	if result.Error != nil {
		r.logger.Error("falha ao atualizar métricas", zap.String("path", path), zap.Error(result.Error))
		return fmt.Errorf("falha ao atualizar métricas: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return repository.ErrRouteNotFound
	}

	return nil
}

// GetRoutesWithFilters obtém rotas com filtros aplicados
func (r *RouteRepository) GetRoutesWithFilters(ctx context.Context, filters map[string]interface{}) ([]*model.Route, error) {
	var entities []model.RouteEntity

	// Construir a consulta com os filtros
	query := r.db.WithContext(ctx)

	// Aplicar filtros
	for key, value := range filters {
		query = query.Where(key, value)
	}

	// Executar a consulta
	if err := query.Find(&entities).Error; err != nil {
		r.logger.Error("falha ao buscar rotas com filtros", zap.Any("filters", filters), zap.Error(err))
		return nil, fmt.Errorf("falha ao buscar rotas: %w", err)
	}

	routes := make([]*model.Route, 0, len(entities))
	for _, entity := range entities {
		route, err := entityToModel(&entity)
		if err != nil {
			r.logger.Error("falha ao converter entidade para modelo", zap.Error(err))
			continue
		}
		routes = append(routes, route)
	}

	return routes, nil
}

// entityToModel converte uma entidade em um modelo
func entityToModel(entity *model.RouteEntity) (*model.Route, error) {
	var methods []string
	if err := json.Unmarshal([]byte(entity.MethodsJSON), &methods); err != nil {
		return nil, err
	}

	var headers []string
	if err := json.Unmarshal([]byte(entity.HeadersJSON), &headers); err != nil {
		return nil, err
	}

	var requiredHeaders []string
	if err := json.Unmarshal([]byte(entity.RequiredHeadersJSON), &requiredHeaders); err != nil {
		return nil, err
	}

	return &model.Route{
		Path:            entity.Path,
		ServiceURL:      entity.ServiceURL,
		Methods:         methods,
		Headers:         headers,
		Description:     entity.Description,
		IsActive:        entity.IsActive,
		CallCount:       entity.CallCount,
		TotalResponse:   time.Duration(entity.TotalResponse),
		RequiredHeaders: requiredHeaders,
		CreatedAt:       entity.CreatedAt,
		UpdatedAt:       entity.UpdatedAt,
	}, nil
}

// modelToEntity converte um modelo em uma entidade
func modelToEntity(route *model.Route) (*model.RouteEntity, error) {
	methodsJSON, err := json.Marshal(route.Methods)
	if err != nil {
		return nil, err
	}

	headersJSON, err := json.Marshal(route.Headers)
	if err != nil {
		return nil, err
	}

	requiredHeadersJSON, err := json.Marshal(route.RequiredHeaders)
	if err != nil {
		return nil, err
	}

	entity := &model.RouteEntity{
		Path:                route.Path,
		ServiceURL:          route.ServiceURL,
		MethodsJSON:         string(methodsJSON),
		HeadersJSON:         string(headersJSON),
		Description:         route.Description,
		IsActive:            route.IsActive,
		CallCount:           route.CallCount,
		TotalResponse:       int64(route.TotalResponse),
		RequiredHeadersJSON: string(requiredHeadersJSON),
	}

	return entity, nil
}
