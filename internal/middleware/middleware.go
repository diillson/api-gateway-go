package middleware

import (
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"net/http"
	"sync"
	"time"
)

type Middleware struct {
	logger  *zap.Logger
	limiter *rate.Limiter
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var visitors = make(map[string]*visitor)
var mtx sync.Mutex

func getVisitor(ip string) *rate.Limiter {
	mtx.Lock()
	defer mtx.Unlock()

	v, exists := visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(1, 5) // Ajuste de acordo com suas necessidades
		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func NewMiddleware(logger *zap.Logger) *Middleware {
	return &Middleware{
		logger:  logger,
		limiter: rate.NewLimiter(1, 5), // por exemplo, 1 request por segundo com burst de 5
	}
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" || token != "Bearer your-token" { // Implementar lógica de autenticação real aqui
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := getVisitor(r.RemoteAddr)
		if !limiter.Allow() {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) ValidateHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, exists := m.routes[r.URL.Path]
		if !exists {
			http.NotFound(w, r)
			return
		}

		for _, header := range route.Headers {
			if r.Header.Get(header) == "" {
				http.Error(w, "Bad Request - Missing Headers", http.StatusBadRequest)
				m.logger.Error("Missing header", zap.String("header", header))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) Analytics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		duration := time.Since(start)
		route, exists := m.routes[r.URL.Path]
		if exists {
			route.CallCount++
			route.TotalResponse += duration
		}

		m.logger.Info("Request processed",
			zap.String("path", r.URL.Path),
			zap.Duration("duration", duration),
			zap.Int64("callCount", route.CallCount),
			zap.Duration("totalResponseTime", route.TotalResponse))
	})
}

func (m *Middleware) AuthenticateAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Aqui, você pode adicionar uma lógica mais complexa para validar se o usuário é um admin,
		// como verificar tokens JWT, OAuth, ou outros métodos de autenticação.
		adminToken := r.Header.Get("Admin-Token")
		if adminToken != "your-secret-admin-token" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) RecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.logger.Error("Recovered from panic", zap.Any("error", err))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
