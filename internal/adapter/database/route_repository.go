package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"strings"
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
	tracer trace.Tracer
}

// NewRouteRepository cria um novo repositório de rotas
func NewRouteRepository(db *gorm.DB, logger *zap.Logger) repository.RouteRepository {
	// Obter tracer para o repositório
	tracer := otel.GetTracerProvider().Tracer("api-gateway.repository.route")

	return &RouteRepository{
		db:     db,
		logger: logger,
		tracer: tracer,
	}
}

// GetRoutes retorna todas as rotas ativas
func (r *RouteRepository) GetRoutes(ctx context.Context) ([]*model.Route, error) {
	// Criar span para a operação
	ctx, span := r.tracer.Start(
		ctx,
		"RouteRepository.GetRoutes",
		trace.WithAttributes(
			attribute.String("db.operation", "select"),
			attribute.String("db.table", "routes"),
		),
	)
	defer span.End()

	var entities []model.RouteEntity

	if err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&entities).Error; err != nil {
		r.logger.Error("falha ao buscar rotas", zap.Error(err))
		// Registrar erro no span
		span.SetStatus(codes.Error, "database error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)

		return nil, fmt.Errorf("falha ao buscar rotas: %w", err)
	}

	routes := make([]*model.Route, 0, len(entities))
	for _, entity := range entities {
		route, err := entityToModel(&entity, r.db)
		if err != nil {
			r.logger.Error("falha ao converter entidade para modelo", zap.Error(err))
			// Registrar erro de conversão no span, mas continuar
			span.AddEvent("error.conversion",
				trace.WithAttributes(
					attribute.String("entity.path", entity.Path),
					attribute.String("error.message", err.Error()),
				),
			)
			continue
		}
		routes = append(routes, route)
	}

	// Adicionar total de rotas como atributo
	span.SetAttributes(attribute.Int("routes.count", len(routes)))
	span.SetStatus(codes.Ok, "")
	return routes, nil
}

// GetRouteByPath obtém uma rota específica pelo caminho
func (r *RouteRepository) GetRouteByPath(ctx context.Context, path string) (*model.Route, error) {
	// Garantir que temos um contexto válido
	if ctx == nil {
		ctx = context.Background()
	}

	// Criar span para a operação
	ctx, span := r.tracer.Start(
		ctx,
		"RouteRepository.GetRouteByPath",
		trace.WithAttributes(
			attribute.String("db.operation", "select"),
			attribute.String("db.table", "routes"),
			attribute.String("route.path", path),
		),
	)
	defer span.End()

	var entity model.RouteEntity

	if err := r.db.WithContext(ctx).Where("path = ?", path).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Registrar not found no span
			span.SetStatus(codes.Error, "route not found")
			span.SetAttributes(attribute.Bool("route.found", false))

			return nil, repository.ErrRouteNotFound
		}
		r.logger.Error("falha ao buscar rota por caminho",
			zap.String("path", path),
			zap.Error(err))

		// Registrar erro no span
		span.SetStatus(codes.Error, "database error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)

		return nil, fmt.Errorf("falha ao buscar rota: %w", err)
	}

	// Registrar que a rota foi encontrada
	span.SetAttributes(attribute.Bool("route.found", true))

	route, err := entityToModel(&entity, r.db)
	if err != nil {
		// Registrar erro no span
		span.SetStatus(codes.Error, "conversion error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)

		return nil, err
	}

	// Adicionar detalhes da rota ao span
	span.SetAttributes(
		attribute.String("route.service_url", route.ServiceURL),
		attribute.Bool("route.is_active", route.IsActive),
		attribute.Int("route.methods_count", len(route.Methods)),
	)
	span.SetStatus(codes.Ok, "")

	return route, nil
}

// AddRoute adiciona uma nova rota
func (r *RouteRepository) AddRoute(ctx context.Context, route *model.Route) error {
	// Criar span para a operação
	ctx, span := r.tracer.Start(
		ctx,
		"RouteRepository.AddRoute",
		trace.WithAttributes(
			attribute.String("db.operation", "insert"),
			attribute.String("db.table", "routes"),
			attribute.String("route.path", route.Path),
			attribute.String("route.service_url", route.ServiceURL),
		),
	)
	defer span.End()

	entity, err := modelToEntity(route, r.db)
	if err != nil {
		span.SetStatus(codes.Error, "conversion error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)
		return fmt.Errorf("falha ao converter modelo para entidade: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		r.logger.Error("falha ao adicionar rota",
			zap.String("path", route.Path),
			zap.Error(err))

		span.SetStatus(codes.Error, "database error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)

		return fmt.Errorf("falha ao adicionar rota: %w", err)
	}

	// Operação bem-sucedida
	span.SetStatus(codes.Ok, "")
	return nil
}

// UpdateRoute atualiza uma rota existente
func (r *RouteRepository) UpdateRoute(ctx context.Context, route *model.Route) error {
	// Criar span para a operação
	ctx, span := r.tracer.Start(
		ctx,
		"RouteRepository.UpdateRoute",
		trace.WithAttributes(
			attribute.String("db.operation", "update"),
			attribute.String("db.table", "routes"),
			attribute.String("route.path", route.Path),
			attribute.String("route.service_url", route.ServiceURL),
			attribute.Bool("route.is_active", route.IsActive),
		),
	)
	defer span.End()

	entity, err := modelToEntity(route, r.db)
	if err != nil {
		span.SetStatus(codes.Error, "conversion error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)
		return fmt.Errorf("falha ao converter modelo para entidade: %w", err)
	}

	result := r.db.WithContext(ctx).Where("path = ?", route.Path).Updates(entity)
	if result.Error != nil {
		r.logger.Error("falha ao atualizar rota",
			zap.String("path", route.Path),
			zap.Error(result.Error))

		span.SetStatus(codes.Error, "database error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", result.Error.Error()),
		)

		return fmt.Errorf("falha ao atualizar rota: %w", result.Error)
	}

	// Adicionar informação sobre linhas afetadas
	span.SetAttributes(attribute.Int64("db.rows_affected", result.RowsAffected))

	// Verificar se alguma linha foi afetada
	if result.RowsAffected == 0 {
		span.SetStatus(codes.Error, "no rows affected")
		span.SetAttributes(attribute.Bool("route.found", false))
		return repository.ErrRouteNotFound
	}

	// Operação bem-sucedida
	span.SetStatus(codes.Ok, "")
	return nil
}

// DeleteRoute remove uma rota pelo caminho
func (r *RouteRepository) DeleteRoute(ctx context.Context, path string) error {
	// Criar span para a operação
	ctx, span := r.tracer.Start(
		ctx,
		"RouteRepository.DeleteRoute",
		trace.WithAttributes(
			attribute.String("db.operation", "delete"),
			attribute.String("db.table", "routes"),
			attribute.String("route.path", path),
		),
	)
	defer span.End()

	result := r.db.WithContext(ctx).Where("path = ?", path).Delete(&model.RouteEntity{})

	if result.Error != nil {
		r.logger.Error("falha ao excluir rota",
			zap.String("path", path),
			zap.Error(result.Error))

		span.SetStatus(codes.Error, "database error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", result.Error.Error()),
		)

		return fmt.Errorf("falha ao excluir rota: %w", result.Error)
	}

	// Adicionar informação sobre linhas afetadas
	span.SetAttributes(attribute.Int64("db.rows_affected", result.RowsAffected))

	if result.RowsAffected == 0 {
		span.SetStatus(codes.Error, "no rows affected")
		span.SetAttributes(attribute.Bool("route.found", false))
		return repository.ErrRouteNotFound
	}

	// Operação bem-sucedida
	span.SetStatus(codes.Ok, "")
	return nil
}

// UpdateMetrics atualiza as métricas de uma rota
func (r *RouteRepository) UpdateMetrics(ctx context.Context, path string, callCount int64, totalResponseTime int64) error {
	// Criar span para a operação
	ctx, span := r.tracer.Start(
		ctx,
		"RouteRepository.UpdateMetrics",
		trace.WithAttributes(
			attribute.String("db.operation", "update"),
			attribute.String("db.table", "routes"),
			attribute.String("route.path", path),
			attribute.Int64("metrics.call_count", callCount),
			attribute.Int64("metrics.response_time", totalResponseTime),
		),
	)
	defer span.End()

	result := r.db.WithContext(ctx).Model(&model.RouteEntity{}).
		Where("path = ?", path).
		Updates(map[string]interface{}{
			"call_count":      callCount,
			"total_response":  totalResponseTime,
			"last_updated_at": time.Now(),
		})

	if result.Error != nil {
		r.logger.Error("falha ao atualizar métricas",
			zap.String("path", path),
			zap.Error(result.Error))

		span.SetStatus(codes.Error, "database error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", result.Error.Error()),
		)
		return fmt.Errorf("falha ao atualizar métricas: %w", result.Error)
	}

	// Adicionar informação sobre linhas afetadas
	span.SetAttributes(attribute.Int64("db.rows_affected", result.RowsAffected))

	if result.RowsAffected == 0 {
		span.SetStatus(codes.Error, "no rows affected")
		span.SetAttributes(attribute.Bool("route.found", false))
		return repository.ErrRouteNotFound
	}

	// Operação bem-sucedida
	span.SetStatus(codes.Ok, "")
	return nil
}

// GetRoutesWithFilters obtém rotas com filtros aplicados
func (r *RouteRepository) GetRoutesWithFilters(ctx context.Context, filters map[string]interface{}) ([]*model.Route, error) {
	// Criar span para a operação
	ctx, span := r.tracer.Start(
		ctx,
		"RouteRepository.GetRoutesWithFilters",
		trace.WithAttributes(
			attribute.String("db.operation", "select"),
			attribute.String("db.table", "routes"),
		),
	)

	// Registrar os filtros como atributos do span
	for key, value := range filters {
		// Convertemos o valor para string para não ter problemas com tipos complexos
		span.SetAttributes(attribute.String("filter."+key, fmt.Sprintf("%v", value)))
	}

	defer span.End()

	var entities []model.RouteEntity

	// Construir a consulta com os filtros
	query := r.db.WithContext(ctx)

	// Aplicar filtros
	for key, value := range filters {
		query = query.Where(key, value)
	}

	// Executar a consulta
	if err := query.Find(&entities).Error; err != nil {
		r.logger.Error("falha ao buscar rotas com filtros",
			zap.Any("filters", filters),
			zap.Error(err))

		span.SetStatus(codes.Error, "database error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)

		return nil, fmt.Errorf("falha ao buscar rotas: %w", err)
	}

	routes := make([]*model.Route, 0, len(entities))
	conversionErrors := 0

	for _, entity := range entities {
		route, err := entityToModel(&entity, r.db)
		if err != nil {
			r.logger.Error("falha ao converter entidade para modelo", zap.Error(err))
			// Registrar evento de erro no span, mas continuar processando
			span.AddEvent("error.conversion",
				trace.WithAttributes(
					attribute.String("entity.path", entity.Path),
					attribute.String("error.message", err.Error()),
				),
			)
			conversionErrors++
			continue
		}
		routes = append(routes, route)
	}

	// Adicionar métricas ao span
	span.SetAttributes(
		attribute.Int("routes.count", len(routes)),
		attribute.Int("routes.conversion_errors", conversionErrors),
	)

	// Operação bem-sucedida com possíveis avisos
	if conversionErrors > 0 {
		span.SetStatus(codes.Ok, fmt.Sprintf("%d conversion errors", conversionErrors))
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return routes, nil
}

// entityToModel converte uma entidade em um modelo
func entityToModel(entity *model.RouteEntity, db *gorm.DB) (*model.Route, error) {
	var methods []string
	var headers []string
	var requiredHeaders []string
	var err error

	// Determinar o tipo de banco de dados
	dbType := "postgres" // valor padrão
	if db != nil {
		dbType = db.Dialector.Name()
	}

	// Processar o JSON de métodos com base no tipo de banco
	if dbType == "sqlite" || dbType == "sqlite3" {
		// Para SQLite: tratar possíveis strings vazias ou nulas
		if entity.MethodsJSON == "" {
			methods = []string{}
		} else {
			err = json.Unmarshal([]byte(entity.MethodsJSON), &methods)
			if err != nil {
				// Tentar deserializar string entre aspas (formato comum em SQLite)
				quotedJSON := strings.Trim(entity.MethodsJSON, "\"")
				err = json.Unmarshal([]byte(quotedJSON), &methods)
				if err != nil {
					return nil, fmt.Errorf("falha ao deserializar métodos (SQLite): %w", err)
				}
			}
		}
	} else {
		// PostgreSQL e outros bancos
		err = json.Unmarshal([]byte(entity.MethodsJSON), &methods)
		if err != nil {
			return nil, fmt.Errorf("falha ao deserializar métodos: %w", err)
		}
	}

	// Processar o JSON de cabeçalhos
	if dbType == "sqlite" || dbType == "sqlite3" {
		if entity.HeadersJSON == "" {
			headers = []string{}
		} else {
			err = json.Unmarshal([]byte(entity.HeadersJSON), &headers)
			if err != nil {
				// Tentar deserializar string entre aspas
				quotedJSON := strings.Trim(entity.HeadersJSON, "\"")
				err = json.Unmarshal([]byte(quotedJSON), &headers)
				if err != nil {
					return nil, fmt.Errorf("falha ao deserializar cabeçalhos (SQLite): %w", err)
				}
			}
		}
	} else {
		err = json.Unmarshal([]byte(entity.HeadersJSON), &headers)
		if err != nil {
			return nil, fmt.Errorf("falha ao deserializar cabeçalhos: %w", err)
		}
	}

	// Processar o JSON de cabeçalhos obrigatórios
	if dbType == "sqlite" || dbType == "sqlite3" {
		if entity.RequiredHeadersJSON == "" {
			requiredHeaders = []string{}
		} else {
			err = json.Unmarshal([]byte(entity.RequiredHeadersJSON), &requiredHeaders)
			if err != nil {
				// Tentar deserializar string entre aspas
				quotedJSON := strings.Trim(entity.RequiredHeadersJSON, "\"")
				err = json.Unmarshal([]byte(quotedJSON), &requiredHeaders)
				if err != nil {
					return nil, fmt.Errorf("falha ao deserializar cabeçalhos obrigatórios (SQLite): %w", err)
				}
			}
		}
	} else {
		err = json.Unmarshal([]byte(entity.RequiredHeadersJSON), &requiredHeaders)
		if err != nil {
			return nil, fmt.Errorf("falha ao deserializar cabeçalhos obrigatórios: %w", err)
		}
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

// modelToEntity converte um modelo em uma entidade, garantindo compatibilidade entre bancos
func modelToEntity(route *model.Route, db *gorm.DB) (*model.RouteEntity, error) {
	// Determinar o tipo de banco de dados
	dbType := "postgres" // valor padrão
	if db != nil {
		dbType = db.Dialector.Name()
	}

	// Garantir que arrays vazios sejam serializados corretamente para evitar problemas com NULL
	if route.Methods == nil {
		route.Methods = []string{}
	}
	if route.Headers == nil {
		route.Headers = []string{}
	}
	if route.RequiredHeaders == nil {
		route.RequiredHeaders = []string{}
	}

	// Serializar para JSON com tratamento de erro melhorado
	methodsJSON, err := json.Marshal(route.Methods)
	if err != nil {
		return nil, fmt.Errorf("falha ao serializar métodos: %w", err)
	}

	headersJSON, err := json.Marshal(route.Headers)
	if err != nil {
		return nil, fmt.Errorf("falha ao serializar cabeçalhos: %w", err)
	}

	requiredHeadersJSON, err := json.Marshal(route.RequiredHeaders)
	if err != nil {
		return nil, fmt.Errorf("falha ao serializar cabeçalhos obrigatórios: %w", err)
	}

	// Garantir que strings JSON vazias são "[]" em vez de ""
	methodsJSONStr := string(methodsJSON)
	if methodsJSONStr == "" || methodsJSONStr == "null" {
		methodsJSONStr = "[]"
	}

	headersJSONStr := string(headersJSON)
	if headersJSONStr == "" || headersJSONStr == "null" {
		headersJSONStr = "[]"
	}

	requiredHeadersJSONStr := string(requiredHeadersJSON)
	if requiredHeadersJSONStr == "" || requiredHeadersJSONStr == "null" {
		requiredHeadersJSONStr = "[]"
	}

	// Tratamento especial para SQLite, que pode ter problemas com alguns formatos JSON
	if dbType == "sqlite" || dbType == "sqlite3" {
		// Garante que o formato seja adequado para SQLite
		if methodsJSONStr == "[]" {
			methodsJSONStr = "\"[]\"" // para SQLite, pode ser melhor armazenar como string
		}
		if headersJSONStr == "[]" {
			headersJSONStr = "\"[]\""
		}
		if requiredHeadersJSONStr == "[]" {
			requiredHeadersJSONStr = "\"[]\""
		}
	}

	entity := &model.RouteEntity{
		Path:                route.Path,
		ServiceURL:          route.ServiceURL,
		MethodsJSON:         methodsJSONStr,
		HeadersJSON:         headersJSONStr,
		Description:         route.Description,
		IsActive:            route.IsActive,
		CallCount:           route.CallCount,
		TotalResponse:       int64(route.TotalResponse),
		RequiredHeadersJSON: requiredHeadersJSONStr,
	}

	// Preservar as datas se estiverem definidas
	if !route.CreatedAt.IsZero() {
		entity.CreatedAt = route.CreatedAt
	}
	if !route.UpdatedAt.IsZero() {
		entity.UpdatedAt = route.UpdatedAt
	}

	return entity, nil
}
