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
	routes, err := db.GetRoutes() // Aqui está a correção, obtendo as rotas do banco de dados
	if err != nil {
		// Log ou trate o erro conforme necessário
		logger.Error("Failed to load routes", zap.Error(err))
	}

	routeMap := make(map[string]*config.Route)
	for _, route := range routes {
		routeMap[route.Path] = route
	}

	return &Handler{routes: routeMap, logger: logger, db: db}
}

func (h *Handler) AddRoute(path, serviceURL string, methods []string) {
	route := &config.Route{
		Path:       path,
		ServiceURL: serviceURL,
		Methods:    methods,
		// Inicialize outros campos conforme necessário
	}
	h.routes[path] = route // Corrigido para adicionar a rota, não a URL
	h.logger.Info("Route added", zap.String("path", path), zap.String("serviceURL", serviceURL))
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, exists := h.routes[r.URL.Path]
	if !exists || !contains(route.Methods, r.Method) {
		http.NotFound(w, r)
		return
	}

	dest, err := url.Parse(route.ServiceURL) // Corrigido para obter a URL do serviço da rota
	if err != nil {
		// Log ou trate o erro conforme necessário
		h.logger.Error("Invalid service URL", zap.String("url", route.ServiceURL), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
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
		h.logger.Error("Failed to add route to database", zap.Error(err))
		http.Error(w, "Failed to register the new API", http.StatusInternalServerError)
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

	routes := make([]*config.Route, 0, len(h.routes)) // Corrigido para usar config.Route
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
		h.logger.Error("Failed to update route in database", zap.Error(err))
		http.Error(w, "Failed to update the API", http.StatusInternalServerError)
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
		h.logger.Error("Failed to delete route from database", zap.Error(err))
		http.Error(w, "Failed to delete the API", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
