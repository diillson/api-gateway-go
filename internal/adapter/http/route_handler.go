package http

import (
	"context"
	"github.com/diillson/api-gateway-go/internal/adapter/proxy"
	"github.com/diillson/api-gateway-go/internal/app/route"
	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/diillson/api-gateway-go/internal/infra/metrics"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"
)

// RouteHandler implementa os handlers para gerenciamento de rotas
type RouteHandler struct {
	routeService *route.Service
	logger       *zap.Logger
	metrics      *metrics.APIMetrics
}

// NewRouteHandler cria um novo handler de rotas
func NewRouteHandler(routeService *route.Service, logger *zap.Logger) *RouteHandler {
	return &RouteHandler{
		routeService: routeService,
		logger:       logger,
	}
}

// SetMetrics configura o objeto de métricas
func (h *RouteHandler) SetMetrics(metrics *metrics.APIMetrics) {
	h.metrics = metrics
}

// RegisterAPI registra uma nova rota da API
func (h *RouteHandler) RegisterAPI(c *gin.Context) {
	var route model.Route
	if err := c.ShouldBindJSON(&route); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	if err := route.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.routeService.AddRoute(c.Request.Context(), &route); err != nil {
		h.logger.Error("Falha ao registrar API", zap.Error(err))
		if h.metrics != nil {
			h.metrics.RequestError(c.FullPath(), c.Request.Method, "add_route_error")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao registrar API"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "API registrada com sucesso", "path": route.Path})
}

// ListAPIs lista todas as rotas cadastradas
func (h *RouteHandler) ListAPIs(c *gin.Context) {
	routes, err := h.routeService.GetRoutes(c.Request.Context())
	if err != nil {
		h.logger.Error("Falha ao listar APIs", zap.Error(err))
		if h.metrics != nil {
			h.metrics.RequestError(c.FullPath(), c.Request.Method, "list_apis_error")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao listar APIs"})
		return
	}

	c.JSON(http.StatusOK, routes)
}

// UpdateAPI atualiza uma rota existente
func (h *RouteHandler) UpdateAPI(c *gin.Context) {
	var route model.Route
	if err := c.ShouldBindJSON(&route); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: " + err.Error()})
		return
	}

	if err := route.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.routeService.UpdateRoute(c.Request.Context(), &route); err != nil {
		h.logger.Error("Falha ao atualizar API", zap.Error(err))
		if h.metrics != nil {
			h.metrics.RequestError(c.FullPath(), c.Request.Method, "update_route_error")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao atualizar API"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API atualizada com sucesso", "path": route.Path})
}

// DeleteAPI remove uma rota
func (h *RouteHandler) DeleteAPI(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parâmetro 'path' é obrigatório"})
		return
	}

	if err := h.routeService.DeleteRoute(c.Request.Context(), path); err != nil {
		h.logger.Error("Falha ao excluir API", zap.Error(err))
		if h.metrics != nil {
			h.metrics.RequestError(c.FullPath(), c.Request.Method, "delete_route_error")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao excluir API"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API excluída com sucesso"})
}

// GetMetrics obtém métricas das rotas
func (h *RouteHandler) GetMetrics(c *gin.Context) {
	routes, err := h.routeService.GetRoutes(c.Request.Context())
	if err != nil {
		h.logger.Error("Falha ao obter métricas", zap.Error(err))
		if h.metrics != nil {
			h.metrics.RequestError(c.FullPath(), c.Request.Method, "get_metrics_error")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao obter métricas"})
		return
	}

	metrics := make(map[string]interface{})
	for _, route := range routes {
		metrics[route.Path] = gin.H{
			"callCount":       route.CallCount,
			"avgResponseTime": route.AverageResponseTime().String(),
			"methods":         route.Methods,
			"isActive":        route.IsActive,
		}
	}

	c.JSON(http.StatusOK, metrics)
}

// Handler representa o manipulador HTTP principal
type Handler struct {
	routeHandler  *RouteHandler
	healthChecker *HealthChecker
	proxy         *proxy.ReverseProxy
	logger        *zap.Logger
	routeService  *route.Service
	metrics       *metrics.APIMetrics
}

func NewHandler(routeService *route.Service, proxy *proxy.ReverseProxy, db DatabaseChecker, cache CacheChecker, logger *zap.Logger) *Handler {
	routeHandler := NewRouteHandler(routeService, logger)
	healthChecker := NewHealthChecker(routeService, db, cache, logger)

	return &Handler{
		routeHandler:  routeHandler,
		healthChecker: healthChecker,
		proxy:         proxy,
		logger:        logger,
		routeService:  routeService,
	}
}

// SetMetrics configura as métricas para o handler e seus componentes
func (h *Handler) SetMetrics(metrics *metrics.APIMetrics) {
	h.metrics = metrics
	h.routeHandler.SetMetrics(metrics)
}

func (h *Handler) HealthCheck(c *gin.Context) {
	h.healthChecker.LivenessCheck(c)
}

func (h *Handler) ReadinessCheck(c *gin.Context) {
	h.healthChecker.ReadinessCheck(c)
}

func (h *Handler) DetailedHealth(c *gin.Context) {
	h.healthChecker.DetailedHealth(c)
}

func (h *Handler) RegisterAPI(c *gin.Context) {
	h.routeHandler.RegisterAPI(c)
}

func (h *Handler) ListAPIs(c *gin.Context) {
	h.routeHandler.ListAPIs(c)
}

func (h *Handler) UpdateAPI(c *gin.Context) {
	h.routeHandler.UpdateAPI(c)
}

func (h *Handler) ClearCache(c *gin.Context) {
	h.routeHandler.ClearCache(c)
}

func (h *Handler) DiagnoseRoute(c *gin.Context) {
	h.routeHandler.DiagnoseRoute(c)
}

func (h *Handler) DeleteAPI(c *gin.Context) {
	h.routeHandler.DeleteAPI(c)
}

func (h *Handler) GetMetrics(c *gin.Context) {
	h.routeHandler.GetMetrics(c)
}

func (h *Handler) ServeAPI(c *gin.Context) {
	// Obter a rota para o caminho atual
	path := c.Request.URL.Path

	h.logger.Info("Recebendo requisição para rota dinâmica",
		zap.String("path", path),
		zap.String("method", c.Request.Method))

	// Debugar o caminho completo
	h.logger.Debug("Detalhes da requisição",
		zap.String("full_path", c.FullPath()),
		zap.String("raw_path", c.Request.URL.RawPath),
		zap.String("raw_query", c.Request.URL.RawQuery))

	ctx := c.Request.Context()
	route, err := h.routeService.GetRouteByPath(ctx, path)
	if err != nil {
		h.logger.Error("Rota não encontrada",
			zap.String("path", path),
			zap.Error(err))

		if h.metrics != nil {
			h.metrics.RequestError(path, c.Request.Method, "route_not_found")
		}

		// Verifique se é um erro específico de rota não encontrada
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "API not found",
			"details": "A rota requisitada não está registrada no API Gateway",
			"path":    path,
		})
		return
	}

	// Logar a rota encontrada
	h.logger.Info("Rota encontrada",
		zap.String("path", route.Path),
		zap.String("serviceURL", route.ServiceURL),
		zap.Strings("methods", route.Methods),
		zap.Bool("isActive", route.IsActive))

	// Verificar se a rota está ativa
	if !route.IsActive {
		h.logger.Warn("Tentativa de acessar rota inativa",
			zap.String("path", path))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "API não disponível",
			"details": "Esta rota está temporariamente desativada",
		})
		return
	}

	// Verificar se o método é permitido
	if !route.IsMethodAllowed(c.Request.Method) {
		h.logger.Warn("Método não permitido",
			zap.String("path", path),
			zap.String("method", c.Request.Method),
			zap.Strings("allowed_methods", route.Methods))

		if h.metrics != nil {
			h.metrics.RequestError(path, c.Request.Method, "method_not_allowed")
		}

		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error":           "Method not allowed",
			"allowed_methods": route.Methods,
		})
		return
	}

	// Verificar cabeçalhos obrigatórios
	if len(route.RequiredHeaders) > 0 {
		headers := make(map[string]string)
		for _, header := range route.RequiredHeaders {
			if value := c.GetHeader(header); value != "" {
				headers[header] = value
			}
		}

		if !route.HasRequiredHeaders(headers) {
			h.logger.Warn("Cabeçalhos obrigatórios ausentes",
				zap.String("path", path),
				zap.Strings("required_headers", route.RequiredHeaders))

			if h.metrics != nil {
				h.metrics.RequestError(path, c.Request.Method, "missing_headers")
			}

			c.JSON(http.StatusBadRequest, gin.H{
				"error":            "Missing required headers",
				"required_headers": route.RequiredHeaders,
			})
			return
		}
	}

	// Registrar a chamada da rota nas métricas
	if h.metrics != nil {
		h.metrics.RequestStarted(path, c.Request.Method)
	}

	// Encaminhar a requisição para o proxy
	start := time.Now()
	h.logger.Info("Encaminhando requisição para o proxy",
		zap.String("target", route.ServiceURL))

	if err := h.proxy.ProxyRequest(route, c.Writer, c.Request); err != nil {
		h.logger.Error("Erro ao encaminhar requisição",
			zap.String("path", path),
			zap.Error(err))

		if h.metrics != nil {
			h.metrics.RequestError(path, c.Request.Method, "proxy_error")
		}
		return
	}

	// Atualizar métricas após a requisição ser processada
	duration := time.Since(start)
	if h.metrics != nil {
		h.metrics.RequestCompleted(path, c.Request.Method,
			strconv.Itoa(c.Writer.Status()), duration,
			int(c.Request.ContentLength), c.Writer.Size())
	}

	// Atualizar métricas da rota de forma assíncrona
	go func() {
		if err := h.routeService.UpdateMetrics(context.Background(),
			path, 1, int64(duration)); err != nil {
			h.logger.Error("Erro ao atualizar métricas", zap.Error(err))
		}
	}()
}

func (h *RouteHandler) ClearCache(c *gin.Context) {
	if err := h.routeService.ClearCache(c.Request.Context()); err != nil {
		h.logger.Error("Falha ao limpar cache", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao limpar cache"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cache limpo com sucesso"})
}

// DiagnoseRoute diagnostica problemas em uma rota específica
func (h *RouteHandler) DiagnoseRoute(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parâmetro 'path' é obrigatório"})
		return
	}

	h.logger.Info("Diagnosticando rota", zap.String("path", path))

	// Verificar se a rota existe no repositório
	ctx := c.Request.Context()
	route, err := h.routeService.GetRouteByPath(ctx, path)

	if err != nil {
		h.logger.Error("Rota não encontrada no repositório", zap.String("path", path), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Rota não encontrada",
			"path":    path,
			"details": err.Error(),
			"suggestions": []string{
				"Verifique se a rota foi registrada corretamente",
				"Certifique-se de que o caminho está exatamente igual ao registrado",
				"Tente limpar o cache de rotas",
			},
		})
		return
	}

	// Verificar se a rota está ativa
	if !route.IsActive {
		c.JSON(http.StatusOK, gin.H{
			"status":     "inactive",
			"route":      route,
			"message":    "A rota existe, mas está inativa",
			"suggestion": "Ative a rota para que possa ser acessada",
		})
		return
	}

	// Testar conectividade com o serviço de destino
	client := http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", route.ServiceURL, nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":     "url_invalid",
			"route":      route,
			"error":      err.Error(),
			"suggestion": "A URL do serviço parece ser inválida, corrija-a",
		})
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":     "service_unreachable",
			"route":      route,
			"error":      err.Error(),
			"suggestion": "O serviço de destino não está acessível. Verifique se está online e se a URL está correta.",
		})
		return
	}
	defer resp.Body.Close()

	c.JSON(http.StatusOK, gin.H{
		"status":         "ok",
		"route":          route,
		"service_status": resp.StatusCode,
		"message":        "A rota está configurada corretamente e o serviço de destino está acessível",
	})
}
