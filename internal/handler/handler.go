package handler

import (
	"github.com/diillson/api-gateway-go/internal/config"
	"go.uber.org/zap"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Handler struct {
	routes map[string]*config.Route
	logger *zap.Logger
}

func NewHandler(routes []*config.Route, logger *zap.Logger) *Handler {
	routeMap := make(map[string]*config.Route)
	for _, route := range routes {
		routeMap[route.Path] = route
	}

	return &Handler{routes: routeMap, logger: logger}
}

func (h *Handler) AddRoute(path, serviceURL, method string) {
	url, _ := url.Parse(serviceURL)
	h.routes[path] = url
	h.logger.Info("Route added", zap.String("path", path), zap.String("serviceURL", serviceURL))
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, exists := h.routes[r.URL.Path]
	if !exists || !contains(route.Methods, r.Method) {
		http.NotFound(w, r)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(dest)
	w.Header().Add("X-Api-Gateway", "MyApiGateway")
	proxy.ServeHTTP(w, r)
}

// Função auxiliar para verificar se uma slice contém um valor específico
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
