package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
)

// SecurityMiddleware implementa proteções de segurança
type SecurityMiddleware struct {
	logger *zap.Logger
}

// NewSecurityMiddleware cria uma nova instância do middleware de segurança
func NewSecurityMiddleware(logger *zap.Logger) *SecurityMiddleware {
	return &SecurityMiddleware{
		logger: logger,
	}
}

// Headers adiciona cabeçalhos de segurança
func (m *SecurityMiddleware) Headers() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Proteção contra clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Proteção contra MIME-sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Proteção contra XSS
		c.Header("X-XSS-Protection", "1; mode=block")

		// Política de segurança de conteúdo
		c.Header("Content-Security-Policy", "default-src 'self'")

		// HTTP Strict Transport Security (HSTS)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Não expõe informações de versão do servidor
		c.Header("Server", "API Gateway")

		// Proteção contra redirecionamento de URL
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Proteção contra rastreamento
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=()")

		// Proteção contra ataques de injeção de código
		c.Header("Cross-Origin-Embedder-Policy", "require-corp")

		// Proteção contra ataques de injeção de recursos
		c.Header("Cross-Origin-Opener-Policy", "same-origin")

		// Proteção contra vazamento de informações
		c.Header("Cross-Origin-Resource-Policy", "same-origin")

		c.Next()
	}
}

// CORS configura Cross-Origin Resource Sharing
func (m *SecurityMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
