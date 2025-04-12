package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// APIMetrics gerencia métricas relacionadas à API
type APIMetrics struct {
	requestCounter     *prometheus.CounterVec
	requestDuration    *prometheus.HistogramVec
	requestSize        *prometheus.SummaryVec
	responseSize       *prometheus.SummaryVec
	activeRequests     *prometheus.GaugeVec
	errorsTotal        *prometheus.CounterVec
	circuitBreakerOpen *prometheus.GaugeVec
	rateLimited        *prometheus.CounterVec
	cacheHitRatio      *prometheus.GaugeVec
}

var (
	// DefaultRegistry é o registro padrão para métricas
	DefaultRegistry = prometheus.NewRegistry()
	// DefaultRegisterer é o registrador padrão para métricas
	DefaultRegisterer = prometheus.WrapRegistererWith(nil, DefaultRegistry)
	// Fábrica para criar métricas automaticamente
	factory = promauto.With(DefaultRegisterer)
)

// NewAPIMetrics cria e registra métricas do prometheus
func NewAPIMetrics() *APIMetrics {
	return &APIMetrics{
		requestCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_requests_total",
				Help: "Total number of HTTP requests by path, method, and status code",
			},
			[]string{"path", "method", "status"},
		),

		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_gateway_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"path", "method"},
		),

		requestSize: promauto.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       "api_gateway_request_size_bytes",
				Help:       "HTTP request size in bytes",
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			},
			[]string{"path", "method"},
		),

		responseSize: promauto.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       "api_gateway_response_size_bytes",
				Help:       "HTTP response size in bytes",
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			},
			[]string{"path", "method"},
		),

		activeRequests: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api_gateway_active_requests",
				Help: "Number of in-flight requests being processed",
			},
			[]string{"path", "method"},
		),

		errorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_errors_total",
				Help: "Total number of errors by type",
			},
			[]string{"path", "method", "error_type"},
		),

		circuitBreakerOpen: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api_gateway_circuit_breaker_open",
				Help: "Indicates if a circuit breaker is open (1) or closed (0)",
			},
			[]string{"service"},
		),

		rateLimited: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_rate_limited_requests_total",
				Help: "Total number of rate limited requests",
			},
			[]string{"path", "method", "limit_type"},
		),

		cacheHitRatio: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api_gateway_cache_hit_ratio",
				Help: "Cache hit ratio (0.0 to 1.0)",
			},
			[]string{"cache_type"},
		),
	}
}

// RequestStarted registra o início de uma requisição
func (m *APIMetrics) RequestStarted(path, method string) {
	m.activeRequests.WithLabelValues(path, method).Inc()
}

// RequestCompleted registra a conclusão de uma requisição
func (m *APIMetrics) RequestCompleted(path, method, status string, duration time.Duration, requestSize, responseSize int) {
	m.requestCounter.WithLabelValues(path, method, status).Inc()
	m.requestDuration.WithLabelValues(path, method).Observe(duration.Seconds())
	m.requestSize.WithLabelValues(path, method).Observe(float64(requestSize))
	m.responseSize.WithLabelValues(path, method).Observe(float64(responseSize))
	m.activeRequests.WithLabelValues(path, method).Dec()
}

// RequestError registra um erro de requisição
func (m *APIMetrics) RequestError(path, method, errorType string) {
	m.errorsTotal.WithLabelValues(path, method, errorType).Inc()
}

// CircuitBreakerStateChanged registra mudança no estado de um circuit breaker
func (m *APIMetrics) CircuitBreakerStateChanged(service string, isOpen bool) {
	value := 0.0
	if isOpen {
		value = 1.0
	}
	m.circuitBreakerOpen.WithLabelValues(service).Set(value)
}

// RateLimitExceeded registra quando um limite de taxa é excedido
func (m *APIMetrics) RateLimitExceeded(path, method, limitType string) {
	m.rateLimited.WithLabelValues(path, method, limitType).Inc()
}

// UpdateCacheHitRatio atualiza a taxa de acertos do cache
func (m *APIMetrics) UpdateCacheHitRatio(cacheType string, hitRatio float64) {
	m.cacheHitRatio.WithLabelValues(cacheType).Set(hitRatio)
}
