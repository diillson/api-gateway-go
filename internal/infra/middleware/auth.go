package middleware

import (
	"net/http"
	"strings"

	"github.com/diillson/api-gateway-go/internal/app/auth"
	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthMiddleware gerencia middlewares de autenticação
type AuthMiddleware struct {
	authService *auth.AuthService
	logger      *zap.Logger
}

// NewAuthMiddleware cria uma nova instância do middleware de autenticação
func NewAuthMiddleware(authService *auth.AuthService, logger *zap.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		logger:      logger,
	}
}

// Authenticate verifica se o usuário está autenticado
func (m *AuthMiddleware) Authenticate(c *gin.Context) {
	// Skip autenticação para rotas públicas
	if isPublicRoute(c.Request.URL.Path) {
		c.Next()
		return
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header não fornecido"})
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Formato inválido do token"})
		return
	}

	user, err := m.authService.ValidateToken(tokenString)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token inválido ou expirado"})
		return
	}

	// Armazena o usuário no contexto para uso posterior
	c.Set("user", user)
	c.Next()
}

// AuthenticateAdmin verifica se o usuário é um administrador
func (m *AuthMiddleware) AuthenticateAdmin(c *gin.Context) {
	// Primeiro autentica o usuário
	m.Authenticate(c)

	// Se o fluxo foi abortado no middleware anterior, retorna
	if c.IsAborted() {
		return
	}

	// Obtém o usuário do contexto
	userValue, exists := c.Get("user")
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Falha ao obter informações do usuário"})
		return
	}

	user, ok := userValue.(*model.User)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Falha ao processar informações do usuário"})
		return
	}

	// Verifica se o usuário é administrador
	if !m.authService.IsAdmin(user) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Acesso negado: permissão de administrador necessária"})
		return
	}

	c.Next()
}

// isPublicRoute determina se uma rota é pública
func isPublicRoute(path string) bool {
	publicPaths := []string{
		"/health",
		"/login",
		"/swagger",
	}

	for _, publicPath := range publicPaths {
		if strings.HasPrefix(path, publicPath) {
			return true
		}
	}

	return false
}
