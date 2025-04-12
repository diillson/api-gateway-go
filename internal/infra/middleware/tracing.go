package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TracingMiddleware fornece rastreamento de requisições
type TracingMiddleware struct {
	logger *zap.Logger
}

// NewTracingMiddleware cria um novo middleware de rastreamento
func NewTracingMiddleware(logger *zap.Logger) *TracingMiddleware {
	return &TracingMiddleware{
		logger: logger,
	}
}

// Middleware inicia um span para cada requisição HTTP
func (m *TracingMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extrai o contexto de propagação do cabeçalho HTTP
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Inicia um novo span
		tracer := otel.Tracer("api-gateway")
		spanName := c.FullPath()
		if spanName == "" {
			spanName = c.Request.URL.Path
		}

		ctx, span := tracer.Start(
			ctx,
			"HTTP "+c.Request.Method+" "+spanName,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// Adiciona atributos ao span
		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.host", c.Request.Host),
			attribute.String("http.user_agent", c.Request.UserAgent()),
			attribute.String("http.client_ip", c.ClientIP()),
		)

		// Atualiza o contexto da requisição com o contexto de rastreamento
		c.Request = c.Request.WithContext(ctx)

		// Executa o próximo middleware/handler
		c.Next()

		// Adiciona atributos de resposta
		span.SetAttributes(
			attribute.Int("http.status_code", c.Writer.Status()),
			attribute.Int("http.response_size", c.Writer.Size()),
		)

		// Marca o span como erro se o status for >= 400
		if c.Writer.Status() >= 400 {
			span.SetAttributes(attribute.Bool("error", true))
		}
	}
}
