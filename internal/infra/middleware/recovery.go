package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"runtime/debug"
)

// RecoveryMiddleware implementa recuperação de pânicos
type RecoveryMiddleware struct {
	logger *zap.Logger
}

// NewRecoveryMiddleware cria um novo middleware de recuperação
func NewRecoveryMiddleware(logger *zap.Logger) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		logger: logger,
	}
}

// Recovery recupera de pânicos com logs detalhados
func (m *RecoveryMiddleware) Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()

				m.logger.Error("recuperado de pânico",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.ByteString("stack", stack),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Erro interno do servidor",
				})
			}
		}()

		c.Next()
	}
}
