package handler

import (
	"github.com/diillson/api-gateway-go/internal/database"
	"github.com/diillson/api-gateway-go/pkg/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type Handler struct {
	routes map[string]*config.Route
	logger *zap.Logger
	db     *database.Database
}

type RouteMetrics struct {
	CallCount     int           `json:"callCount"`
	TotalResponse time.Duration `json:"totalResponse"`
	ServiceURL    string        `json:"serviceURL"`
	Path          string        `json:"path"`
}

func NewHandler(db *database.Database, logger *zap.Logger) *Handler {
	routes, err := db.GetRoutes()
	if err != nil {
		logger.Error("Failed to load routes", zap.Error(err))
	}

	routeMap := make(map[string]*config.Route)
	for _, route := range routes {
		routeMap[route.Path] = route
	}

	return &Handler{routes: routeMap, logger: logger, db: db}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h.updateRoutes(); err != nil {
		h.logger.Error("Failed to update routes", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	route, exists := h.routes[r.URL.Path]
	if !exists || !route.IsMethodAllowed(r.Method) {
		http.NotFound(w, r)
		return
	}

	// Parse the service URL
	target, err := url.Parse(route.ServiceURL)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create a new reverse proxy to forward the request to the service
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Modify the request
	r.URL.Host = target.Host
	r.URL.Scheme = target.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = target.Host

	// Serve the request
	proxy.ServeHTTP(w, r)
}

func (h *Handler) updateRoutes() error {
	routes, err := h.db.GetRoutes()
	if err != nil {
		h.logger.Error("Failed to load routes", zap.Error(err))
		return err
	}

	routeMap := make(map[string]*config.Route)
	for _, route := range routes {
		routeMap[route.Path] = route
	}

	h.routes = routeMap
	return nil
}

func (h *Handler) GetMetrics(c *gin.Context) {
	path := c.Query("path")

	// Se o path não for especificado, retorne métricas para todas as rotas
	if path == "" {
		var allMetrics []RouteMetrics
		for _, route := range h.routes {
			allMetrics = append(allMetrics, RouteMetrics{
				CallCount:     int(route.CallCount),
				TotalResponse: route.TotalResponse,
				ServiceURL:    route.ServiceURL,
				Path:          route.Path,
				// Mapeie outros campos conforme necessário
			})
		}
		c.JSON(http.StatusOK, allMetrics)
		return
	}

	// Se um path específico for especificado, retorne métricas apenas para essa rota
	route, exists := h.routes[path]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Route not found"})
		return
	}

	specificMetrics := RouteMetrics{
		CallCount:     int(route.CallCount),
		TotalResponse: route.TotalResponse,
		ServiceURL:    route.ServiceURL,
		Path:          route.Path,
		// Mapeie outros campos conforme necessário
	}

	c.JSON(http.StatusOK, specificMetrics)
}

func (h *Handler) RegisterAPI(c *gin.Context) {
	var newRoutes []config.Route
	err := c.BindJSON(&newRoutes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, newRoute := range newRoutes {
		err = h.db.AddRoute(&newRoute)
		if err != nil {
			h.logger.Error("Failed to add route to database", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register the new API, Is Incorrect or Route already exists"})
			return
		}
	}

	if err := h.updateRoutes(); err != nil {
		h.logger.Error("Failed to update routes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update routes"})
		return
	}

	// Log info about the registered routes
	h.logger.Info("Routes registered successfully",
		zap.Int("totalRoutes", len(newRoutes)),
		zap.Strings("routes", getRoutePaths(newRoutes))) // Example of logging the paths of the registered routes

	c.JSON(http.StatusCreated, newRoutes)
}

// Helper function to extract paths from the routes
func getRoutePaths(routes []config.Route) []string {
	var paths []string
	for _, route := range routes {
		paths = append(paths, route.Path)
	}
	return paths
}

func RouteExists(engine *gin.Engine, methods []string, path string) bool {
	for _, route := range engine.Routes() {
		for _, method := range methods {
			if route.Method == method && route.Path == path {
				return true
			}
		}
	}
	return false
}

func (h *Handler) ListAPIs(c *gin.Context) {
	routes, err := h.db.GetRoutes()
	if err != nil {
		h.logger.Error("Failed to get routes from database", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get routes"})
		return
	}

	c.JSON(http.StatusOK, routes)
}

func (h *Handler) UpdateAPI(c *gin.Context) {
	var updatedRoute config.Route
	err := c.BindJSON(&updatedRoute)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.db.UpdateRoute(&updatedRoute)
	if err != nil {
		h.logger.Error("Failed to update route in database", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update the API"})
		return
	}

	if err := h.updateRoutes(); err != nil {
		h.logger.Error("Failed to update routes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update routes"})
		return
	}

	c.JSON(http.StatusOK, updatedRoute)
}

func (h *Handler) DeleteAPI(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path query parameter required"})
		return
	}

	err := h.db.DeleteRoute(path)
	if err != nil {
		h.logger.Error("Failed to delete route from database", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete the API"})
		return
	}

	if err := h.updateRoutes(); err != nil {
		h.logger.Error("Failed to update routes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update routes"})
		return
	}

	c.Status(http.StatusNoContent)
}
