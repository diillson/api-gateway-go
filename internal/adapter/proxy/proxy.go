package proxy

import (
	"context"
	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/diillson/api-gateway-go/internal/infra/metrics"
	"github.com/diillson/api-gateway-go/pkg/cache"
	"github.com/diillson/api-gateway-go/pkg/resilience"
	"go.uber.org/zap"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ReverseProxy oferece funcionalidade de proxy reverso com circuit breaker
type ReverseProxy struct {
	cache           cache.Cache
	circuitBreakers map[string]*resilience.CircuitBreaker
	cbLock          sync.RWMutex
	logger          *zap.Logger
	metrics         *metrics.APIMetrics
}

// NewReverseProxy cria um novo ReverseProxy
func NewReverseProxy(cache cache.Cache, logger *zap.Logger) *ReverseProxy {
	return &ReverseProxy{
		cache:           cache,
		circuitBreakers: make(map[string]*resilience.CircuitBreaker),
		logger:          logger,
	}
}

// SetMetrics configura as métricas para o proxy
func (p *ReverseProxy) SetMetrics(metrics *metrics.APIMetrics) {
	p.metrics = metrics
}

// ProxyRequest encaminha uma requisição para o backend
func (p *ReverseProxy) ProxyRequest(route *model.Route, w http.ResponseWriter, r *http.Request) error {
	// Verifica se o circuito está aberto para este serviço
	cb := p.getCircuitBreaker(route.ServiceURL)

	// Cria contexto com timeout para a requisição
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Executa a requisição através do circuit breaker
	_, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return p.doProxy(route, w, r.WithContext(ctx))
	})

	return err
}

// doProxy executa o proxy reverso real
func (p *ReverseProxy) doProxy(route *model.Route, w http.ResponseWriter, r *http.Request) (interface{}, error) {
	targetURL, err := url.Parse(route.ServiceURL)
	if err != nil {
		p.logger.Error("falha ao analisar URL do serviço",
			zap.String("serviceURL", route.ServiceURL),
			zap.Error(err))
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return nil, err
	}

	// Registrar métricas para esta requisição de proxy, se disponível
	if p.metrics != nil {
		p.metrics.RequestStarted(route.Path, r.Method)
	}

	// Cria o proxy reverso
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Modifica o diretor para ajustar a requisição
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Preserva o caminho e a query string
		req.URL.Path = r.URL.Path
		req.URL.RawQuery = r.URL.RawQuery

		// Preserva o IP original
		req.Header.Set("X-Forwarded-For", r.RemoteAddr)
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Host = targetURL.Host

		// Adiciona cabeçalhos personalizados, se necessário
		for _, header := range route.Headers {
			if val := r.Header.Get(header); val != "" {
				req.Header.Set(header, val)
			}
		}
	}

	// Manipula erros do proxy
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		p.logger.Error("erro no proxy",
			zap.String("path", r.URL.Path),
			zap.String("serviceURL", route.ServiceURL),
			zap.Error(err))

		// Determinar o tipo de erro com base na causa
		var errorType string
		var statusCode int

		// Analisar o erro para determinar o tipo apropriado
		if strings.Contains(err.Error(), "context deadline exceeded") {
			errorType = "timeout_error"
			statusCode = http.StatusGatewayTimeout
		} else if strings.Contains(err.Error(), "connection refused") {
			errorType = "connection_refused"
			statusCode = http.StatusServiceUnavailable
		} else if strings.Contains(err.Error(), "no such host") {
			errorType = "host_not_found"
			statusCode = http.StatusBadGateway
		} else {
			errorType = "proxy_error"
			statusCode = http.StatusBadGateway
		}

		// Registrar erro nas métricas com tipo específico
		if p.metrics != nil {
			p.metrics.RequestError(r.URL.Path, r.Method, errorType)
		}

		http.Error(w, "Erro ao encaminhar requisição: "+err.Error(), statusCode)
	}

	// Executa o proxy
	proxy.ServeHTTP(w, r)
	return nil, nil
}

// getCircuitBreaker obtém ou cria um circuit breaker para um serviço
func (p *ReverseProxy) getCircuitBreaker(serviceURL string) *resilience.CircuitBreaker {
	p.cbLock.RLock()
	cb, exists := p.circuitBreakers[serviceURL]
	p.cbLock.RUnlock()

	if exists {
		return cb
	}

	// Se não existe, cria um novo circuit breaker
	p.cbLock.Lock()
	defer p.cbLock.Unlock()

	// Verificar novamente em caso de race condition
	cb, exists = p.circuitBreakers[serviceURL]
	if exists {
		return cb
	}

	// Criar nova configuração de circuit breaker
	config := resilience.CircuitBreakerConfig{
		Name:        "cb-" + serviceURL,
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     30 * time.Second,
	}

	cb = resilience.NewCircuitBreaker(config, p.logger, p.metrics)
	p.circuitBreakers[serviceURL] = cb

	return cb
}
