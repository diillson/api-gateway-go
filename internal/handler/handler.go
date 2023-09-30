package handler

import (
	"go.uber.org/zap"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Handler struct {
	routes map[string]*url.URL
	logger *zap.Logger
}

func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{routes: make(map[string]*url.URL), logger: logger}
}

func (h *Handler) AddRoute(path, serviceURL, method string) {
	url, _ := url.Parse(serviceURL)
	h.routes[path] = url
	h.logger.Info("Route added", zap.String("path", path), zap.String("serviceURL", serviceURL))
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dest, exists := h.routes[r.URL.Path]
	if !exists {
		http.NotFound(w, r)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(dest)
	proxy.ServeHTTP(w, r)
}
