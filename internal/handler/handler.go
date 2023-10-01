package handler

import (
	"encoding/json"
	"github.com/diillson/api-gateway-go/internal/config"
	"github.com/diillson/api-gateway-go/internal/database"
	"go.uber.org/zap"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Handler struct {
	routes map[string]*config.Route
	logger *zap.Logger
	db     *database.Database
}

func NewHandler(db *database.Database, logger *zap.Logger) *Handler {
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

func (h *Handler) RegisterAPI(w http.ResponseWriter, r *http.Request) {
	var newRoute config.Route
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&newRoute)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.db.AddRoute(&newRoute)
	if err != nil {
		http.Error(w, "Failed to add route", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newRoute)
}

func (h *Handler) ListAPIs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	routes := make([]*Route, 0, len(h.routes))
	for _, route := range h.routes {
		routes = append(routes, route)
	}

	json.NewEncoder(w).Encode(routes)
}

func (h *Handler) UpdateAPI(w http.ResponseWriter, r *http.Request) {
	var updatedRoute config.Route
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&updatedRoute)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.db.UpdateRoute(&updatedRoute)
	if err != nil {
		http.Error(w, "Failed to update route", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedRoute)
}

func (h *Handler) DeleteAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path query parameter required", http.StatusBadRequest)
		return
	}

	err := h.db.DeleteRoute(path)
	if err != nil {
		http.Error(w, "Failed to delete route", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
