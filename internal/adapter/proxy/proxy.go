package proxy

import (
	"context"
	"fmt"
	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/diillson/api-gateway-go/internal/infra/metrics"
	"github.com/diillson/api-gateway-go/pkg/cache"
	"github.com/diillson/api-gateway-go/pkg/resilience"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
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
	tracer          trace.Tracer
}

// NewReverseProxy cria um novo ReverseProxy
func NewReverseProxy(cache cache.Cache, logger *zap.Logger) *ReverseProxy {
	// Criar o tracer para o proxy
	tracer := otel.Tracer("api-gateway.proxy")

	logger.Info("Inicializando Reverse Proxy com tracing habilitado")

	return &ReverseProxy{
		cache:           cache,
		circuitBreakers: make(map[string]*resilience.CircuitBreaker),
		logger:          logger,
		tracer:          tracer,
	}
}

// SetMetrics configura as métricas para o proxy
func (p *ReverseProxy) SetMetrics(metrics *metrics.APIMetrics) {
	p.metrics = metrics
}

// ProxyRequest encaminha uma requisição para o backend
func (p *ReverseProxy) ProxyRequest(route *model.Route, w http.ResponseWriter, r *http.Request) error {
	// Obter o contexto atual com o span
	ctx := r.Context()

	// Criar um novo span para esta operação de proxy
	ctx, span := p.tracer.Start(
		ctx,
		fmt.Sprintf("Proxy %s -> %s", r.URL.Path, route.ServiceURL),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	// Atualizar o request com o novo contexto
	r = r.WithContext(ctx)

	// Adicionar atributos relevantes ao span
	span.SetAttributes(
		attribute.String("proxy.target_url", route.ServiceURL),
		attribute.String("proxy.source_path", r.URL.Path),
		attribute.StringSlice("proxy.allowed_methods", route.Methods),
		attribute.Bool("proxy.is_active", route.IsActive),
	)
	// Verifica se o circuito está aberto para este serviço
	cb := p.getCircuitBreaker(route.ServiceURL)

	// Propagar o contexto de tracing para o serviço downstream
	propagator := otel.GetTextMapPropagator()
	carrier := propagation.HeaderCarrier(r.Header)
	propagator.Inject(ctx, carrier)

	// Cria contexto com timeout para a requisição
	ctxWithTimeout, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Executa a requisição através do circuit breaker
	_, err := cb.Execute(ctxWithTimeout, func(execCtx context.Context) (interface{}, error) {
		// Atualizar o request com o contexto de execução
		execRequest := r.WithContext(execCtx)

		// Criar span para a operação específica do proxy
		_, execSpan := p.tracer.Start(
			execCtx,
			"ProxyExecution",
			trace.WithSpanKind(trace.SpanKindClient),
		)
		defer execSpan.End()

		result, err := p.doProxy(route, w, execRequest)

		if err != nil {
			execSpan.SetStatus(codes.Error, err.Error())
			execSpan.SetAttributes(attribute.Bool("error", true))
			execSpan.SetAttributes(attribute.String("error.message", err.Error()))
		} else {
			execSpan.SetStatus(codes.Ok, "")
		}

		return result, err
	})

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.Bool("error", true))
		span.SetAttributes(attribute.String("error.message", err.Error()))
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// doProxy executa o proxy reverso real
func (p *ReverseProxy) doProxy(route *model.Route, w http.ResponseWriter, r *http.Request) (interface{}, error) {
	// Obter o contexto com span
	ctx := r.Context()

	// Criar span para esta operação
	ctx, span := p.tracer.Start(
		ctx,
		fmt.Sprintf("ProxyRequest:%s->%s", r.URL.Path, route.ServiceURL),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("proxy.target_url", route.ServiceURL),
			attribute.String("proxy.source_path", r.URL.Path),
			attribute.StringSlice("proxy.allowed_methods", route.Methods),
			attribute.Bool("proxy.is_active", route.IsActive),
		),
	)
	defer span.End()

	// Atualizar o request com o novo contexto
	r = r.WithContext(ctx)

	// Log para diagnóstico
	p.logger.Info("Executando proxy com trace",
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.String("span_id", span.SpanContext().SpanID().String()),
		zap.String("target", route.ServiceURL),
		zap.String("path", r.URL.Path))

	targetURL, err := url.Parse(route.ServiceURL)
	if err != nil {
		span.SetStatus(codes.Error, "failed to parse service URL")
		span.SetAttributes(attribute.Bool("error", true))
		span.SetAttributes(attribute.String("error.message", err.Error()))

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

	// Cria o proxy reverso com tracing
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Preservar o caminho e a query string
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.URL.Path = r.URL.Path
			req.URL.RawQuery = r.URL.RawQuery

			// Preservar o IP original
			req.Header.Set("X-Forwarded-For", r.RemoteAddr)
			req.Header.Set("X-Forwarded-Host", r.Host)
			req.Host = targetURL.Host

			// Adicionar cabeçalhos personalizados, se necessário
			for _, header := range route.Headers {
				if val := r.Header.Get(header); val != "" {
					req.Header.Set(header, val)
				}
			}

			// ADICIONADO: Propagar explicitamente o contexto de tracing para o serviço downstream
			otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

			// Adicionar span ID como cabeçalho para correlação
			spanContext := trace.SpanContextFromContext(ctx)
			if spanContext.IsValid() {
				req.Header.Set("X-Trace-ID", spanContext.TraceID().String())
				req.Header.Set("X-Span-ID", spanContext.SpanID().String())
			}
		},

		ModifyResponse: func(res *http.Response) error {
			// Adicionar informações da resposta ao span
			span.SetAttributes(
				attribute.Int("http.response.status_code", res.StatusCode),
				attribute.String("http.response.content_type", res.Header.Get("Content-Type")),
			)

			// Marcar como erro se o status for >= 400
			if res.StatusCode >= 400 {
				span.SetStatus(codes.Error, fmt.Sprintf("Response status code: %d", res.StatusCode))
				span.SetAttributes(attribute.Bool("error", true))
			} else {
				// Explicitamente marcar como OK se não for erro
				span.SetStatus(codes.Ok, "")
			}

			return nil
		},

		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			p.logger.Error("erro no proxy",
				zap.String("path", r.URL.Path),
				zap.String("serviceURL", route.ServiceURL),
				zap.Error(err))

			// Adicionar detalhes do erro ao span
			span.SetStatus(codes.Error, err.Error())
			span.SetAttributes(attribute.Bool("error", true))
			span.SetAttributes(attribute.String("error.message", err.Error()))
			span.RecordError(err) // ADICIONADO: Registrar o erro explicitamente

			// Determinar o tipo de erro e status HTTP apropriado
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
		},
	}

	// Executa o proxy
	proxy.ServeHTTP(w, r)

	// A resposta foi enviada com sucesso se chegou aqui
	// Garantir que o status OK seja definido (pode ter sido substituído por valores do ModifyResponse)
	if span.SpanContext().IsValid() {
		span.SetStatus(codes.Ok, "")
	}

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
