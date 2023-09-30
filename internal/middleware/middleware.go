package middleware

import (
	"go.uber.org/zap"
	"net/http"
)

type MiddlewareChain struct {
	logger *zap.Logger
}

func NewMiddlewareChain(logger *zap.Logger) *MiddlewareChain {
	return &MiddlewareChain{logger: logger}
}

func (mc *MiddlewareChain) Then(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mc.logger.Info("Request received", zap.String("path", r.URL.Path))
		handler.ServeHTTP(w, r)
	})
}
