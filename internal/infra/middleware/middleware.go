package middleware

import (
	"github.com/diillson/api-gateway-go/internal/app/auth"
	"github.com/diillson/api-gateway-go/internal/infra/metrics"
	"github.com/diillson/api-gateway-go/pkg/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
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
	// Criar um cliente Redis se necessário
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Inicializar o limiter
	limiter := ratelimit.NewRedisLimiter(redisClient, logger)

	// Inicializar o middleware de rate limit
	rateLimitMiddleware := NewRateLimitMiddleware(limiter, apiMetrics, logger)

	return &Middleware{
		logger:              logger,
		authMiddleware:      NewAuthMiddleware(authService, logger),
		recoveryMiddleware:  NewRecoveryMiddleware(logger),
		securityMiddleware:  NewSecurityMiddleware(logger),
		tracingMiddleware:   NewTracingMiddleware(logger),
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

// Tracing middleware para rastreamento de requisições
func (m *Middleware) Tracing() gin.HandlerFunc {
	return m.tracingMiddleware.Middleware()
}
