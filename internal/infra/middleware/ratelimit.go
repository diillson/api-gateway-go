package middleware

import (
	"fmt"
	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/diillson/api-gateway-go/internal/infra/metrics"
	"net/http"
	"strconv"
	"time"

	"github.com/diillson/api-gateway-go/pkg/ratelimit"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RateLimitMiddleware gerencia rate limiting
type RateLimitMiddleware struct {
	limiter             *ratelimit.RedisLimiter
	logger              *zap.Logger
	metrics             *metrics.APIMetrics
	rateLimitMiddleware *RateLimitMiddleware
}

// NewRateLimitMiddleware cria um novo middleware de rate limiting
func NewRateLimitMiddleware(limiter *ratelimit.RedisLimiter, metrics *metrics.APIMetrics, logger *zap.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
		logger:  logger,
		metrics: metrics,
	}
}

// IPRateLimit limita requisições por IP
func (m *RateLimitMiddleware) IPRateLimit() gin.HandlerFunc {
	if m.rateLimitMiddleware != nil {
		return m.rateLimitMiddleware.IPRateLimit()
	}
	return func(c *gin.Context) {
		// Obtém o IP do cliente
		clientIP := c.ClientIP()

		// Configuração do limitador para este IP
		config := ratelimit.LimitConfig{
			Key:         clientIP,
			Limit:       100,         // 100 requisições
			Period:      time.Minute, // por minuto
			BurstFactor: 1.5,         // permite até 50% mais em picos
		}

		blockKey := fmt.Sprintf("ratelimit:blocked:%s", clientIP)
		blocked, _ := m.limiter.RedisClient.Get(c, blockKey).Bool()
		if blocked {
			c.Header("Retry-After", "600") // 10 minutos
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "IP temporariamente bloqueado devido a excesso de requisições",
				"retry_after": 600,
			})
			return
		}

		// Verifica se a requisição é permitida
		allowed, limit, remaining, resetAfter, err := m.limiter.Allow(c.Request.Context(), config)
		if err != nil {
			m.logger.Error("erro ao verificar rate limit", zap.Error(err))
			c.Next() // Em caso de erro, permite a requisição
			return
		}

		if !allowed && remaining < -100 { // Valor negativo alto indica muitas requisições excedentes
			// Registrar evento de rate limiting
			if m.metrics != nil {
				// Obtém o caminho da requisição atual
				path := c.FullPath()
				if path == "" {
					path = c.Request.URL.Path
				}
				m.metrics.RateLimitExceeded(path, c.Request.Method, "ip_limit")
			}
			m.logger.Warn("Possível ataque detectado - alto volume de requisições",
				zap.String("ip", clientIP),
				zap.Int("requests", limit-remaining),
				zap.Int("threshold", limit*3))

			// Bloquear IP por período mais longo (10 minutos)
			blockKey := fmt.Sprintf("ratelimit:blocked:%s", clientIP)
			m.limiter.RedisClient.Set(c, blockKey, true, 10*time.Minute)

			c.Header("Retry-After", "600") // 10 minutos
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Limite de requisições excedido significativamente",
				"retry_after": 600,
			})
			return
		}

		// Adiciona cabeçalhos de rate limit
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(resetAfter).Unix(), 10))

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(resetAfter.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "taxa de requisições excedida",
				"retry_after": int(resetAfter.Seconds()),
			})
			return
		}

		c.Next()
	}
}

// APIRateLimit limita requisições para uma API específica
func (m *RateLimitMiddleware) APIRateLimit(limit int, period time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtém o caminho da API
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		m.metrics.RateLimitExceeded(path, c.Request.Method, "ip_limit")

		// Configuração do limitador para esta API
		config := ratelimit.LimitConfig{
			Key:         "api:" + path,
			Limit:       limit,
			Period:      period,
			BurstFactor: 1.2, // permite até 20% mais em picos
		}

		// Verifica se a requisição é permitida
		allowed, limit, remaining, resetAfter, err := m.limiter.Allow(c.Request.Context(), config)
		if err != nil {
			m.logger.Error("erro ao verificar rate limit da API", zap.Error(err))
			c.Next() // Em caso de erro, permite a requisição
			return
		}

		// Adiciona cabeçalhos de rate limit
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(resetAfter).Unix(), 10))

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(resetAfter.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "taxa de requisições para esta API excedida",
				"retry_after": int(resetAfter.Seconds()),
			})
			return
		}

		c.Next()
	}
}

// UserRateLimit limita requisições por usuário (requer autenticação)
func (m *RateLimitMiddleware) UserRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtém o usuário do contexto (se autenticado)
		userVal, exists := c.Get("user")
		if !exists {
			c.Next() // Se não houver usuário, passa adiante
			return
		}

		user, ok := userVal.(*model.User)
		if !ok {
			c.Next() // Se não puder converter o usuário, passa adiante
			return
		}

		// Configuração do limitador para este usuário
		config := ratelimit.LimitConfig{
			Key:         "user:" + user.ID,
			Limit:       1000,        // 1000 requisições
			Period:      time.Minute, // por minuto
			BurstFactor: 1.5,         // permite até 50% mais em picos
		}

		// Verifica se a requisição é permitida
		allowed, limit, remaining, resetAfter, err := m.limiter.Allow(c.Request.Context(), config)
		if err != nil {
			m.logger.Error("erro ao verificar rate limit do usuário", zap.Error(err))
			c.Next() // Em caso de erro, permite a requisição
			return
		}

		// Adiciona cabeçalhos de rate limit
		c.Header("X-RateLimit-User-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-User-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-User-Reset", strconv.FormatInt(time.Now().Add(resetAfter).Unix(), 10))

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(resetAfter.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "taxa de requisições do usuário excedida",
				"retry_after": int(resetAfter.Seconds()),
			})
			return
		}

		c.Next()
	}
}
