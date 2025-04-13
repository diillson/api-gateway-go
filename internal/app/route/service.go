package route

import (
	"context"
	"time"

	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/diillson/api-gateway-go/internal/domain/repository"
	"github.com/diillson/api-gateway-go/pkg/cache"
	"go.uber.org/zap"
)

type Service struct {
	repo   repository.RouteRepository
	cache  cache.Cache
	logger *zap.Logger
}

func NewService(repo repository.RouteRepository, cache cache.Cache, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

// GetRoutes retorna todas as rotas ativas
func (s *Service) GetRoutes(ctx context.Context) ([]*model.Route, error) {
	var routes []*model.Route

	// Tentar cache primeiro
	cacheKey := "routes"
	found, err := s.cache.Get(ctx, cacheKey, &routes)
	if err != nil {
		s.logger.Error("Erro ao buscar rotas do cache", zap.Error(err))
		return nil, err
	}

	if found {
		return routes, nil
	}

	// Se não estiver no cache, buscar do repositório
	routes, err = s.repo.GetRoutes(ctx)
	if err != nil {
		return nil, err
	}

	// Armazenar no cache para futuras requisições
	if err := s.cache.Set(ctx, cacheKey, routes, 5*time.Minute); err != nil {
		s.logger.Warn("Erro ao armazenar rotas no cache", zap.Error(err))
	}

	return routes, nil
}

// GetRouteByPath obtém uma rota pelo caminho
func (s *Service) GetRouteByPath(ctx context.Context, path string) (*model.Route, error) {
	// Adicione log para debug
	s.logger.Info("Buscando rota", zap.String("path", path))

	var route model.Route

	// Tentar cache primeiro
	cacheKey := "route:" + path
	found, err := s.cache.Get(ctx, cacheKey, &route)
	if err != nil {
		s.logger.Error("Erro ao buscar rota do cache", zap.String("path", path), zap.Error(err))
		// Continue para buscar do repositório em caso de erro
	} else if found {
		s.logger.Info("Rota encontrada no cache", zap.String("path", path))
		return &route, nil
	}

	// Se não estiver no cache, buscar todas as rotas do repositório
	s.logger.Info("Rota não encontrada no cache, buscando do repositório", zap.String("path", path))
	routes, err := s.repo.GetRoutes(ctx)
	if err != nil {
		s.logger.Error("Erro ao buscar rotas do repositório", zap.Error(err))
		return nil, err
	}

	// Percorrer todas as rotas e verificar correspondência
	for _, r := range routes {
		if model.MatchRoutePath(r.Path, path) {
			// Armazenar no cache para futuras requisições
			if err := s.cache.Set(ctx, cacheKey, r, 5*time.Minute); err != nil {
				s.logger.Warn("Erro ao armazenar rota no cache", zap.Error(err))
			}

			s.logger.Info("Rota encontrada no repositório com correspondência de padrão",
				zap.String("registeredPath", r.Path),
				zap.String("requestPath", path),
				zap.String("serviceURL", r.ServiceURL))
			return r, nil
		}
	}

	// Se não encontrou correspondência
	s.logger.Error("Nenhuma rota correspondente encontrada",
		zap.String("path", path))
	return nil, repository.ErrRouteNotFound
}

// ClearCache limpa o cache de rotas
func (s *Service) ClearCache(ctx context.Context) error {
	// Limpar cache de rotas
	if err := s.cache.Delete(ctx, "routes"); err != nil {
		s.logger.Error("Erro ao limpar cache de rotas", zap.Error(err))
		return err
	}

	// Buscar todas as rotas para limpar cache individual
	routes, err := s.repo.GetRoutes(ctx)
	if err != nil {
		s.logger.Error("Erro ao buscar rotas para limpar cache", zap.Error(err))
		return err
	}

	for _, route := range routes {
		cacheKey := "route:" + route.Path
		if err := s.cache.Delete(ctx, cacheKey); err != nil {
			s.logger.Warn("Erro ao limpar cache de rota",
				zap.String("path", route.Path),
				zap.Error(err))
		}
	}

	s.logger.Info("Cache de rotas limpo com sucesso")
	return nil
}

// AddRoute adiciona uma nova rota
func (s *Service) AddRoute(ctx context.Context, route *model.Route) error {
	if err := s.repo.AddRoute(ctx, route); err != nil {
		return err
	}

	// Invalidar cache de rotas
	if err := s.cache.Delete(ctx, "routes"); err != nil {
		s.logger.Warn("Erro ao invalidar cache de rotas", zap.Error(err))
	}

	return nil
}

// UpdateRoute atualiza uma rota existente
func (s *Service) UpdateRoute(ctx context.Context, route *model.Route) error {
	if err := s.repo.UpdateRoute(ctx, route); err != nil {
		return err
	}

	// Invalidar caches
	cacheKey := "route:" + route.Path
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		s.logger.Warn("Erro ao invalidar cache de rota", zap.Error(err))
	}

	if err := s.cache.Delete(ctx, "routes"); err != nil {
		s.logger.Warn("Erro ao invalidar cache de rotas", zap.Error(err))
	}

	return nil
}

// DeleteRoute remove uma rota
func (s *Service) DeleteRoute(ctx context.Context, path string) error {
	if err := s.repo.DeleteRoute(ctx, path); err != nil {
		return err
	}

	// Invalidar caches
	cacheKey := "route:" + path
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		s.logger.Warn("Erro ao invalidar cache de rota", zap.Error(err))
	}

	if err := s.cache.Delete(ctx, "routes"); err != nil {
		s.logger.Warn("Erro ao invalidar cache de rotas", zap.Error(err))
	}

	return nil
}

// UpdateMetrics atualiza as métricas de uma rota
func (s *Service) UpdateMetrics(ctx context.Context, path string, callCount int64, totalResponseTime int64) error {
	return s.repo.UpdateMetrics(ctx, path, callCount, totalResponseTime)
}

// IsMethodAllowed verifica se um método é permitido para uma rota
func (s *Service) IsMethodAllowed(ctx context.Context, path, method string) (bool, error) {
	route, err := s.GetRouteByPath(ctx, path)
	if err != nil {
		return false, err
	}

	for _, m := range route.Methods {
		if m == method {
			return true, nil
		}
	}

	return false, nil
}
