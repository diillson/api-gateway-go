package middleware

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"strconv"
	"time"

	"github.com/diillson/api-gateway-go/internal/infra/metrics"
	"github.com/gin-gonic/gin"
)

// MetricsMiddleware fornece middleware para coletar métricas
type MetricsMiddleware struct {
	metrics *metrics.APIMetrics
	logger  *zap.Logger
}

// NewMetricsMiddleware cria um novo middleware de métricas
func NewMetricsMiddleware(metrics *metrics.APIMetrics, logger *zap.Logger) *MetricsMiddleware {
	return &MetricsMiddleware{
		metrics: metrics,
		logger:  logger,
	}
}

// MetricsHandler gerencia os endpoints de métricas
type MetricsHandler struct {
	Metrics *metrics.APIMetrics
	Logger  *zap.Logger
}

// GetMetrics retorna o objeto APIMetrics para uso em outras partes da aplicação
func (h *MetricsHandler) GetMetrics() *metrics.APIMetrics {
	return h.Metrics
}

// RegisterEndpoint registra o endpoint para expor métricas do Prometheus
func (h *MetricsHandler) RegisterEndpoint(router *gin.Engine) {
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	h.Logger.Info("Endpoint de métricas Prometheus registrado em /metrics")
}

// NewMetricsHandler cria um novo handler de métricas
func NewMetricsHandler(metrics *metrics.APIMetrics, logger *zap.Logger) *MetricsHandler {
	return &MetricsHandler{
		Metrics: metrics,
		Logger:  logger,
	}
}

// Middleware registra métricas para cada requisição
func (m *MetricsMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obter o caminho e método
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		method := c.Request.Method

		// Registrar o início da requisição
		m.metrics.RequestStarted(path, method)

		// Registrar o tamanho da requisição
		var requestSize int
		if c.Request.ContentLength > 0 {
			requestSize = int(c.Request.ContentLength)
		}

		// Registrar tempo de início
		start := time.Now()

		// Envolver o ResponseWriter para capturar o tamanho da resposta
		blw := &bodyLogWriter{body: []byte{}, ResponseWriter: c.Writer}
		c.Writer = blw

		// Processar a requisição
		c.Next()

		// Calcular a duração
		duration := time.Since(start)

		// Registrar a conclusão da requisição
		status := strconv.Itoa(c.Writer.Status())
		responseSize := blw.size

		m.metrics.RequestCompleted(path, method, status, duration, requestSize, responseSize)

		// Registrar erros, se houver
		if c.Writer.Status() >= 400 {
			errorType := "client_error"
			if c.Writer.Status() >= 500 {
				errorType = "server_error"
			}
			m.metrics.RequestError(path, method, errorType)
		}
	}
}

// bodyLogWriter é um wrapper para gin.ResponseWriter para capturar o tamanho do corpo
type bodyLogWriter struct {
	gin.ResponseWriter
	body []byte
	size int
}

// Write implementa a interface io.Writer
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}

// WriteString implementa a interface io.StringWriter
func (w *bodyLogWriter) WriteString(s string) (int, error) {
	size, err := w.ResponseWriter.WriteString(s)
	w.size += size
	return size, err
}
