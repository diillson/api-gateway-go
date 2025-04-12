package telemetry

import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TracerProvider é um provedor de rastreamento com recursos de limpeza
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	logger   *zap.Logger
}

// NewTracerProvider inicializa e configura o OpenTelemetry
func NewTracerProvider(ctx context.Context, serviceName, collectorURL string, logger *zap.Logger) (*TracerProvider, error) {
	// Conectar ao coletor OTLP
	conn, err := grpc.DialContext(ctx, collectorURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	// Criar exportador de traces
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	// Criar recurso com atributos do serviço
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("environment", getEnvironment()),
		),
	)
	if err != nil {
		return nil, err
	}

	// Criar o TracerProvider com a configuração
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1)), // Amostragem de 10%
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Configurar propagação de contexto
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Configurar o provider global
	otel.SetTracerProvider(tp)

	return &TracerProvider{
		provider: tp,
		logger:   logger,
	}, nil
}

// Shutdown encerra o tracer provider de forma limpa
func (tp *TracerProvider) Shutdown(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := tp.provider.Shutdown(ctx); err != nil {
		tp.logger.Error("falha ao encerrar tracer provider", zap.Error(err))
	}
}

// Tracer retorna um tracer nomeado
func (tp *TracerProvider) Tracer(name string) trace.Tracer {
	return tp.provider.Tracer(name)
}

// getEnvironment retorna o ambiente atual (dev, staging, prod)
func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		return "development"
	}
	return env
}
