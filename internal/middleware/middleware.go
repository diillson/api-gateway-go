package middleware

import (
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"net/http"
	"time"
)

type Middleware struct {
	logger  *zap.Logger
	limiter *rate.Limiter
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
		if !m.limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) ValidateHeaders(next http.Handler) http.Handler {
	requiredHeaders := []string{"X-Request-ID", "X-Client-ID"} // Exemplo de headers requeridos

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, header := range requiredHeaders {
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
		m.logger.Info("Request processed",
			zap.String("path", r.URL.Path),
			zap.Duration("duration", duration))
	})
}
