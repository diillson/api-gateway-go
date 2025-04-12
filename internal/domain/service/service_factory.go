package service

import (
	"github.com/diillson/api-gateway-go/internal/app/auth"
	"github.com/diillson/api-gateway-go/internal/app/route"
	"github.com/diillson/api-gateway-go/internal/domain/repository"
	"github.com/diillson/api-gateway-go/pkg/cache"
	"github.com/diillson/api-gateway-go/pkg/security"
	"go.uber.org/zap"
)

// Services contém todos os serviços da aplicação
type Services struct {
	RouteService *route.Service
	AuthService  *auth.AuthService
}

// NewServices cria todos os serviços necessários
func NewServices(routeRepo repository.RouteRepository, userRepo auth.UserRepository, cache cache.Cache, logger *zap.Logger) (*Services, error) {
	// Criar gerenciador de chaves
	keyManager, err := security.NewKeyManager(logger)
	if err != nil {
		return nil, err
	}

	// Criar serviço de autenticação
	authService := auth.NewAuthService(keyManager, userRepo, logger)

	// Criar serviço de rotas
	routeService := route.NewService(routeRepo, cache, logger)

	return &Services{
		RouteService: routeService,
		AuthService:  authService,
	}, nil
}
