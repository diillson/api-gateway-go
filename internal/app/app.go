package app

import (
	"context"
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
	"github.com/go-redis/redis/v8"
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
func NewApp(logger *zap.Logger, cfg *config.Config) (*App, error) {
	// Carregar a configuração do arquivo
	//cfg, err := config.LoadConfig("./config")
	//if err != nil {
	//	return nil, fmt.Errorf("erro ao carregar configuração: %w", err)
	//}

	//if cfg.Tracing.Enabled {
	//	tp, err := telemetry.NewTracerProvider(
	//		context.Background(),
	//		cfg.Tracing.ServiceName,
	//		cfg.Tracing.Endpoint,
	//		logger,
	//	)
	//	if err != nil {
	//		logger.Error("Falha ao inicializar tracer", zap.Error(err))
	//	} else {
	//		logger.Info("Tracer inicializado com sucesso",
	//			zap.String("provider", cfg.Tracing.Provider),
	//			zap.String("endpoint", cfg.Tracing.Endpoint))
	//		defer tp.Shutdown(context.Background())
	//	}
	//}

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

	logger.Info("Configuração de banco de dados",
		zap.String("driver", cfg.Database.Driver),
		zap.String("dsn", cfg.Database.DSN))

	// Inicializar métricas
	apiMetrics := metrics.NewAPIMetrics()
	metricsHandler := &middleware.MetricsHandler{
		Metrics: apiMetrics,
		Logger:  logger,
	}

	// Inicializar o cache apropriado com base na configuração
	var cacheInstance cache.Cache
	if !cfg.Cache.Enabled {
		logger.Info("Cache desabilitado pela configuração")
		cacheInstance = &cache.NoOpCache{}
	} else if cfg.Cache.Type == "redis" {
		logger.Info("Inicializando cache Redis",
			zap.String("address", cfg.Cache.Redis.Address),
			zap.Int("db", cfg.Cache.Redis.DB),
			zap.Duration("ttl", cfg.Cache.TTL))

		// Criar opções avançadas para o Redis
		redisOptions := &redis.Options{
			Addr:         cfg.Cache.Redis.Address,
			Password:     cfg.Cache.Redis.Password,
			DB:           cfg.Cache.Redis.DB,
			PoolSize:     cfg.Cache.Redis.PoolSize,
			MinIdleConns: cfg.Cache.Redis.MinIdleConns,
			MaxRetries:   cfg.Cache.Redis.MaxRetries,
			ReadTimeout:  cfg.Cache.Redis.ReadTimeout,
			WriteTimeout: cfg.Cache.Redis.WriteTimeout,
			DialTimeout:  cfg.Cache.Redis.DialTimeout,
			PoolTimeout:  cfg.Cache.Redis.PoolTimeout,
			IdleTimeout:  cfg.Cache.Redis.IdleTimeout,
			MaxConnAge:   cfg.Cache.Redis.MaxConnAge,
		}

		// Criar o cliente Redis usando as opções avançadas
		redisClient := redis.NewClient(redisOptions)

		// Verificar a conexão
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pingErr := redisClient.Ping(ctx).Err()
		cancel()

		if pingErr != nil {
			logger.Error("Falha ao conectar ao Redis, usando cache em memória",
				zap.Error(pingErr),
				zap.String("redis.address", cfg.Cache.Redis.Address))
			// Fallback para cache em memória em caso de erro
			cacheInstance = cache.NewMemoryCache(cfg.Cache.TTL, 10*time.Minute, apiMetrics, logger)
		} else {
			// Criar o cache Redis com o cliente já configurado e conectado
			cacheInstance, err = cache.NewRedisCache(
				cfg.Cache.Redis.Address,
				cfg.Cache.Redis.Password,
				cfg.Cache.Redis.DB,
				logger,
			)
			if err != nil {
				logger.Error("Falha ao inicializar cache Redis, usando cache em memória",
					zap.Error(err),
					zap.String("redis.address", cfg.Cache.Redis.Address))
				// Fallback para cache em memória
				cacheInstance = cache.NewMemoryCache(
					cfg.Cache.TTL,
					10*time.Minute,
					apiMetrics,
					logger,
				)
			} else {
				logger.Info("Cache Redis inicializado com sucesso")
			}
		}
	} else {
		// Cache em memória (padrão)
		logger.Info("Inicializando cache em memória",
			zap.Duration("ttl", cfg.Cache.TTL),
			zap.Int("maxItems", cfg.Cache.MaxItems))
		cacheInstance = cache.NewMemoryCache(cfg.Cache.TTL, 10*time.Minute, apiMetrics, logger)
	}

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
	routeService := route.NewService(routeRepo, cacheInstance, logger)

	// Inicializar serviços de domínio
	services, err := service.NewServices(routeRepo, userRepo, cacheInstance, logger)
	if err != nil {
		return nil, err
	}

	// Inicializar proxy reverso com métricas
	reverseProxy := proxy.NewReverseProxy(cacheInstance, logger)
	reverseProxy.SetMetrics(apiMetrics)

	// Inicializar middleware com as métricas já criadas
	metricsMiddleware := middleware.NewMetricsMiddleware(apiMetrics, logger)
	middlewares := middleware.NewMiddleware(logger, authService, apiMetrics)

	// Adicionar middleware de métricas ao conjunto de middlewares
	middlewares.SetMetricsMiddleware(metricsMiddleware)

	// Inicializar handlers HTTP com métricas
	handler := http.NewHandler(routeService, reverseProxy, db, cacheInstance, logger)

	// Carregar rotas do arquivo JSON se existir
	jsonLoader := database.NewJSONRouteLoader(db, logger)
	if err := jsonLoader.LoadRoutesFromJSON("./config/routes.json"); err != nil {
		logger.Error("Erro ao carregar rotas do arquivo JSON", zap.Error(err))
	}

	// Configurar métricas no handler
	handler.SetMetrics(apiMetrics)

	return &App{
		Logger:         logger,
		DB:             db,
		Handler:        handler,
		Middleware:     middlewares,
		Services:       services,
		Cache:          cacheInstance,
		MetricsHandler: metricsHandler,
		APIMetrics:     apiMetrics,
	}, nil
}

// RegisterRoutes registra todas as rotas no router
func (a *App) RegisterRoutes(router *gin.Engine) {
	// Configurar middleware global
	router.Use(a.Middleware.Recovery())
	router.Use(a.Middleware.Logger())
	router.Use(a.Middleware.Tracing())
	router.Use(a.Middleware.Metrics())

	userHandler := http.NewUserHandler(a.DB.DB(), a.Logger)

	// Adicionar rotas de autenticação e usuários
	auth := router.Group("/auth")
	{
		auth.POST("/login", userHandler.Login)
	}

	// Rotas para gerenciamento de usuários
	users := router.Group("/admin/users")
	users.Use(a.Middleware.Authenticate)
	{
		users.POST("", userHandler.RegisterUser)     // Criar novo usuário
		users.GET("", userHandler.GetUsers)          // Listar todos os usuários
		users.GET("/:id", userHandler.GetUserByID)   // Obter usuário por ID
		users.PUT("/:id", userHandler.UpdateUser)    // Atualizar usuário
		users.DELETE("/:id", userHandler.DeleteUser) // Excluir usuário
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
		//admin.POST("/users", userHandler.RegisterUser)
		admin.GET("/clear-cache", a.Handler.ClearCache)
		admin.GET("/diagnose-route", a.Handler.DiagnoseRoute)
		admin.GET("/health/detailed", a.Handler.DetailedHealth)
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
