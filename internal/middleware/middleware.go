package middleware

import (
	"github.com/diillson/api-gateway-go/internal/auth"
	"github.com/diillson/api-gateway-go/internal/config"
	"github.com/diillson/api-gateway-go/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Middleware struct {
	logger  *zap.Logger
	limiter *rate.Limiter
	routes  map[string]*config.Route
	db      *database.Database
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var visitors = make(map[string]*visitor)
var mtx sync.Mutex

func NewMiddleware(logger *zap.Logger, routes map[string]*config.Route, db *database.Database) *Middleware {
	return &Middleware{
		logger:  logger,
		limiter: rate.NewLimiter(1, 5),
		routes:  routes,
		db:      db,
	}
}

func getVisitor(ip string) *rate.Limiter {
	mtx.Lock()
	defer mtx.Unlock()

	v, exists := visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(1, 15)
		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func (m *Middleware) Authenticate(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" || !strings.HasPrefix(token, "Bearer ") {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	c.Next()
}

func (m *Middleware) RateLimit(c *gin.Context) {
	limiter := getVisitor(c.ClientIP())
	if !limiter.Allow() {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too Many Requests"})
		return
	}

	c.Next()
}

func (m *Middleware) ValidateHeaders(c *gin.Context) {
	route, exists := m.routes[c.Request.URL.Path]
	if !exists {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}

	for _, header := range route.RequiredHeaders {
		if c.GetHeader(header) == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing Headers"})
			return
		}
	}

	c.Next()
}

func (m *Middleware) Analytics(c *gin.Context) {
	start := time.Now()
	c.Next()
	duration := time.Since(start)

	path := c.Request.URL.Path
	route, exists := m.routes[path]
	if exists {
		route.CallCount++
		route.TotalResponse += duration

		// Aqui atualizamos as métricas na base de dados
		err := m.updateMetricsInDB(path, int(route.CallCount), route.TotalResponse)
		if err != nil {
			m.logger.Error("Failed to update metrics in database", zap.Error(err))
		}
	}

	m.logger.Info("Request processed",
		zap.String("path", path),
		zap.Duration("duration", duration))
}

func (m *Middleware) updateMetricsInDB(path string, callCount int, totalResponse time.Duration) error {
	// Atualizando o banco de dados com as métricas coletadas
	route := &config.Route{
		Path:          path,
		CallCount:     int64(callCount),
		TotalResponse: totalResponse,
	}

	// Assume que você tem um método no seu objeto de banco de dados para atualizar as métricas
	if err := m.db.UpdateMetrics(route); err != nil {
		return err
	}
	return nil
}

func (m *Middleware) AuthenticateAdmin(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header not provided"})
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims := &auth.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.NewValidationError("unexpected signing method", jwt.ValidationErrorSignatureInvalid)
		}
		return auth.JwtKey, nil
	})

	if err != nil || !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	c.Next()
}

func (m *Middleware) RecoverPanic(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			m.logger.Error("Recovered from panic", zap.Any("error", err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
	}()
	c.Next()
}
