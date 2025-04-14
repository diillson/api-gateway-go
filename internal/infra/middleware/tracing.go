package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TracingMiddleware fornece rastreamento de requisições
type TracingMiddleware struct {
	logger      *zap.Logger
	serviceName string
}

// NewTracingMiddleware cria um novo middleware de rastreamento
func NewTracingMiddleware(logger *zap.Logger, serviceName string) *TracingMiddleware {
	if serviceName == "" {
		serviceName = "api-gateway"
	}
	return &TracingMiddleware{
		logger:      logger,
		serviceName: serviceName,
	}
}

// Middleware inicia um span para cada requisição HTTP
func (m *TracingMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obter o tracer
		tracer := otel.GetTracerProvider().Tracer(m.serviceName)

		// Extrair o contexto de propagação do cabeçalho HTTP
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Criar um nome para o span baseado na rota
		spanName := c.FullPath()
		if spanName == "" {
			spanName = c.Request.URL.Path
		}
		spanName = fmt.Sprintf("HTTP %s %s", c.Request.Method, spanName)

		// Iniciar um novo span
		ctx, span := tracer.Start(
			ctx,
			spanName,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// Adicionar atributos ao span
		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.scheme", c.Request.URL.Scheme),
			attribute.String("http.host", c.Request.Host),
			attribute.String("http.user_agent", c.Request.UserAgent()),
			attribute.String("http.client_ip", c.ClientIP()),
			attribute.String("http.flavor", fmt.Sprintf("%d.%d", c.Request.ProtoMajor, c.Request.ProtoMinor)),
		)

		// Adicionar atributos de headers importantes para o contexto
		// Exemplo: headers de autorização de maneira segura
		if c.Request.Header.Get("Authorization") != "" {
			span.SetAttributes(attribute.Bool("http.has_auth", true))
		}

		// Injetar o span no contexto da requisição
		c.Request = c.Request.WithContext(ctx)

		// Capturar erros ou panics durante o processamento
		defer func() {
			if r := recover(); r != nil {
				// Marcar o span como erro em caso de panic
				span.SetStatus(codes.Error, fmt.Sprintf("panic: %v", r))
				span.SetAttributes(attribute.Bool("error", true))
				// Propagar o panic após registrar no span
				panic(r)
			}
		}()

		// Processar o resto da cadeia de middleware
		c.Next()

		// Adicionar atributos de resposta
		statusCode := c.Writer.Status()
		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
			attribute.Int("http.response_size", c.Writer.Size()),
		)

		// Marcar o span como erro se o status for >= 400
		if statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP status code: %d", statusCode))
			span.SetAttributes(attribute.Bool("error", true))
			// Adicionar detalhes de erro se disponíveis (do contexto)
			if errMsg, exists := c.Get("error"); exists {
				span.SetAttributes(attribute.String("error.message", fmt.Sprintf("%v", errMsg)))
			}
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}
