package middleware

import (
	"context"
	"github.com/diillson/api-gateway-go/internal/app/auth"
	"github.com/diillson/api-gateway-go/internal/infra/metrics"
	"github.com/diillson/api-gateway-go/pkg/cache"
	"github.com/diillson/api-gateway-go/pkg/config"
	"github.com/diillson/api-gateway-go/pkg/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"net/http"
	"time"
)

// Middleware contém todos os middlewares da aplicação
type Middleware struct {
	logger              *zap.Logger
	authMiddleware      *AuthMiddleware
	recoveryMiddleware  *RecoveryMiddleware
	securityMiddleware  *SecurityMiddleware
	tracingMiddleware   *TracingMiddleware
	metricsMiddleware   *MetricsMiddleware
	rateLimitMiddleware *RateLimitMiddleware
}

// NewMiddleware cria um novo conjunto de middlewares
func NewMiddleware(logger *zap.Logger, authService *auth.AuthService, apiMetrics *metrics.APIMetrics) *Middleware {
	// Carregar configuração
	cfg, err := config.LoadConfig("./config")
	serviceName := "api-gateway"
	if err == nil && cfg.Tracing.Enabled {
		serviceName = cfg.Tracing.ServiceName
	}

	// Criar um cliente Redis com a configuração correta
	var redisClient *redis.Client

	// Verificar se o Redis está configurado
	if cfg.Cache.Type == "redis" && cfg.Cache.Redis.Address != "" {
		redisClient, err = cache.NewRedisClientWithConfig(&redis.Options{
			Addr:     cfg.Cache.Redis.Address,
			Password: cfg.Cache.Redis.Password,
			DB:       cfg.Cache.Redis.DB,
		}, logger)

		// Verificar a conexão
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			logger.Error("Erro ao conectar ao Redis para rate limiting, usando limitador em memória",
				zap.Error(err),
				zap.String("redis.address", cfg.Cache.Redis.Address))
			// Fallback para um limitador em memória
			redisClient = nil
		} else {
			logger.Info("Conectado ao Redis para rate limiting",
				zap.String("redis.address", cfg.Cache.Redis.Address))
		}
	} else {
		logger.Info("Redis não configurado para rate limiting, usando limitador em memória")
	}

	// Inicializar o limiter apropriado
	var limiter *ratelimit.RedisLimiter
	if redisClient != nil {
		limiter = ratelimit.NewRedisLimiter(redisClient, logger)
	} else {
		// Aqui poderíamos usar um limitador em memória como fallback
		// Para este exemplo, vamos criar um client Redis local
		localRedisClient := redis.NewClient(&redis.Options{
			Addr: "localhost:6379", // Endereço padrão local
		})
		limiter = ratelimit.NewRedisLimiter(localRedisClient, logger)
	}

	// Inicializar o middleware de rate limit
	rateLimitMiddleware := NewRateLimitMiddleware(limiter, apiMetrics, logger)
	tracingMiddleware := NewTracingMiddleware(logger, serviceName)

	return &Middleware{
		logger:              logger,
		authMiddleware:      NewAuthMiddleware(authService, logger),
		recoveryMiddleware:  NewRecoveryMiddleware(logger),
		securityMiddleware:  NewSecurityMiddleware(logger),
		tracingMiddleware:   tracingMiddleware,
		rateLimitMiddleware: rateLimitMiddleware,
	}
}

// SetMetricsMiddleware configura o middleware de métricas
func (m *Middleware) SetMetricsMiddleware(metricsMiddleware *MetricsMiddleware) {
	m.metricsMiddleware = metricsMiddleware
}

// Metrics retorna o middleware de métricas
func (m *Middleware) Metrics() gin.HandlerFunc {
	if m.metricsMiddleware != nil {
		return m.metricsMiddleware.Middleware()
	}
	return func(c *gin.Context) {
		c.Next() // No-op se não configurado
	}
}

// Authenticate middleware para autenticação de usuários
func (m *Middleware) Authenticate(c *gin.Context) {
	m.authMiddleware.Authenticate(c)
}

// AuthenticateAdmin middleware para autenticação de administradores
func (m *Middleware) AuthenticateAdmin(c *gin.Context) {
	m.authMiddleware.AuthenticateAdmin(c)
}

// Recovery middleware para recuperação de pânicos
func (m *Middleware) Recovery() gin.HandlerFunc {
	return m.recoveryMiddleware.Recovery()
}

// IgnoreFavicon é um middleware que ignora requisições para /favicon.ico
func (m *Middleware) IgnoreFavicon() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Se for uma requisição para /favicon.ico, retornar 204 (No Content)
		if c.Request.URL.Path == "/favicon.ico" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Logger middleware para logging de requisições
func (m *Middleware) Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Processar requisição
		c.Next()

		// Depois de processada
		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		m.logger.Info("request completed",
			zap.String("path", path),
			zap.String("method", method),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", clientIP),
		)
	}
}

// SecurityHeaders middleware para adicionar cabeçalhos de segurança
func (m *Middleware) SecurityHeaders() gin.HandlerFunc {
	return m.securityMiddleware.Headers()
}

// CORS middleware para configurar CORS
func (m *Middleware) CORS() gin.HandlerFunc {
	return m.securityMiddleware.CORS()
}

// Tracing retorna o middleware de tracing
func (m *Middleware) Tracing() gin.HandlerFunc {
	return m.tracingMiddleware.Middleware()
}
