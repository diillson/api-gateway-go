package app

import (
	"context"
	"fmt"
	"github.com/diillson/api-gateway-go/internal/adapter/database"
	"github.com/diillson/api-gateway-go/internal/adapter/http"
	"github.com/diillson/api-gateway-go/internal/adapter/proxy"
	"github.com/diillson/api-gateway-go/internal/app/auth"
	"github.com/diillson/api-gateway-go/internal/app/route"
	"github.com/diillson/api-gateway-go/internal/domain/service"
	"github.com/diillson/api-gateway-go/internal/infra/metrics"
	"github.com/diillson/api-gateway-go/internal/infra/middleware"
	"github.com/diillson/api-gateway-go/pkg/cache"
	"github.com/diillson/api-gateway-go/pkg/config"
	"github.com/diillson/api-gateway-go/pkg/security"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	net2 "net/http"
	"time"
)

type App struct {
	Logger         *zap.Logger
	DB             *database.Database
	Handler        *http.Handler
	Middleware     *middleware.Middleware
	Services       *service.Services
	Cache          cache.Cache
	MetricsHandler *middleware.MetricsHandler
	APIMetrics     *metrics.APIMetrics
}

// NewApp cria uma nova instância da aplicação com todas as dependências injetadas
func NewApp(logger *zap.Logger) (*App, error) {
	// Carregar a configuração do arquivo
	cfg, err := config.LoadConfig("./config")
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	// Configurações do banco de dados baseadas no arquivo config.yaml
	dbConfig := database.Config{
		Driver:          cfg.Database.Driver,
		DSN:             cfg.Database.DSN,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		LogLevel:        4,
		SlowThreshold:   cfg.Database.SlowThreshold,
		MigrationDir:    cfg.Database.MigrationDir,
	}

	// Inicializar banco de dados
	ctx := context.Background()
	db, err := database.NewDatabase(ctx, dbConfig, logger)
	if err != nil {
		return nil, err
	}

	// Inicializar métricas
	apiMetrics := metrics.NewAPIMetrics()
	metricsHandler := &middleware.MetricsHandler{
		Metrics: apiMetrics,
		Logger:  logger,
	}
	// Inicializar cache em memória
	memCache := cache.NewMemoryCache(5*time.Minute, 10*time.Minute, apiMetrics, logger)

	// Inicializar repositórios
	routeRepo := database.NewRouteRepository(db.DB(), logger)
	userRepo := database.NewUserRepository(db.DB())

	// Inicializar gerenciador de chaves JWT
	keyManager, err := security.NewKeyManager(logger)
	if err != nil {
		return nil, err
	}

	// Inicializar serviços
	authService := auth.NewAuthService(keyManager, userRepo, logger)
	routeService := route.NewService(routeRepo, memCache, logger)

	// Inicializar serviços de domínio
	services, err := service.NewServices(routeRepo, userRepo, memCache, logger)
	if err != nil {
		return nil, err
	}

	// Inicializar proxy reverso com métricas
	reverseProxy := proxy.NewReverseProxy(memCache, logger)
	reverseProxy.SetMetrics(apiMetrics)

	// Inicializar middleware com as métricas já criadas
	metricsMiddleware := middleware.NewMetricsMiddleware(apiMetrics, logger)
	middlewares := middleware.NewMiddleware(logger, authService, apiMetrics)

	// Adicionar middleware de métricas ao conjunto de middlewares
	middlewares.SetMetricsMiddleware(metricsMiddleware)

	// Inicializar handlers HTTP com métricas
	handler := http.NewHandler(routeService, reverseProxy, db, memCache, logger)

	// Configurar métricas no handler
	handler.SetMetrics(apiMetrics)

	return &App{
		Logger:         logger,
		DB:             db,
		Handler:        handler,
		Middleware:     middlewares,
		Services:       services,
		Cache:          memCache,
		MetricsHandler: metricsHandler,
		APIMetrics:     apiMetrics,
	}, nil
}

// RegisterRoutes registra todas as rotas no router
func (a *App) RegisterRoutes(router *gin.Engine) {
	// Configurar middleware global
	router.Use(a.Middleware.Recovery())
	router.Use(a.Middleware.Logger())
	router.Use(a.Middleware.Metrics()) // Adicionar middleware de métricas globalmente

	userHandler := http.NewUserHandler(a.DB.DB(), a.Logger)

	// Adicionar rotas de autenticação e usuários
	auth := router.Group("/auth")
	{
		auth.POST("/login", userHandler.Login)
	}

	// Expor endpoint de métricas para Prometheus
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	a.Logger.Info("Endpoint de métricas Prometheus registrado em /metrics")

	// Rotas públicas
	router.GET("/health", a.Handler.HealthCheck)
	router.GET("/health/readiness", a.Handler.ReadinessCheck)
	router.GET("/health/liveness", a.Handler.HealthCheck)

	// Rotas administrativas
	admin := router.Group("/admin")
	admin.Use(a.Middleware.Authenticate)
	{
		admin.POST("/register", a.Handler.RegisterAPI)
		admin.GET("/apis", a.Handler.ListAPIs)
		admin.PUT("/update", a.Handler.UpdateAPI)
		admin.DELETE("/delete", a.Handler.DeleteAPI)
		admin.GET("/metrics", a.Handler.GetMetrics)
		admin.POST("/users", userHandler.RegisterUser)
		admin.GET("/clear-cache", a.Handler.ClearCache)
		admin.GET("/diagnose-route", a.Handler.DiagnoseRoute)
	}

	router.Any("/api/*path", a.Handler.ServeAPI)
	router.Any("/ws/*path", a.Handler.ServeAPI)

	router.NoRoute(func(c *gin.Context) {
		// Verificar se estamos em uma rota que já possui um handler
		path := c.Request.URL.Path

		// Log para debug
		a.Logger.Info("Rota não encontrada no router, tentando servir como API dinâmica",
			zap.String("path", path))

		// Verificar se a rota existe no serviço de rotas
		ctx := c.Request.Context()
		_, err := a.Services.RouteService.GetRouteByPath(ctx, path)
		if err == nil {
			// Rota encontrada no serviço, encaminhar para ServeAPI
			a.Handler.ServeAPI(c)
			return
		}

		// Rota não encontrada no serviço, retornar 404
		a.Logger.Info("Rota não encontrada no serviço de rotas",
			zap.String("path", path),
			zap.Error(err))

		c.JSON(net2.StatusNotFound, gin.H{
			"error": "Rota não encontrada",
			"path":  path,
		})
	})
}
