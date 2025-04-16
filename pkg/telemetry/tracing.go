package telemetry

import (
	"context"
	"os"
	"time"

	"github.com/diillson/api-gateway-go/pkg/config"
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

	// Carregar configuração do arquivo
	cfg, err := loadTracingConfig(logger)
	if err != nil {
		logger.Warn("Erro ao carregar configuração do arquivo, usando valores padrão", zap.Error(err))
	}

	// Determinar qual exportador usar baseado na configuração
	traceExporter, err := getTraceExporter(ctx, collectorURL, cfg, logger)
	if err != nil {
		return nil, err
	}

	// Criar o TracerProvider com a configuração
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRatio)),
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)

	// Configurar propagação de contexto
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Configurar o provider global
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return &TracerProvider{
		provider: tp,
		logger:   logger,
	}, nil
}

// loadTracingConfig carrega a configuração de tracing do arquivo ou ambiente
func loadTracingConfig(logger *zap.Logger) (config.TracingConfig, error) {
	// Valor padrão
	defaultConfig := config.TracingConfig{
		Enabled:       true,
		Provider:      "otlp",
		SamplingRatio: 0.1,
	}

	// Tentar carregar do arquivo
	cfg, err := config.LoadConfig("./config")
	if err != nil {
		logger.Warn("Não foi possível carregar arquivo de configuração", zap.Error(err))
		return defaultConfig, err
	}

	// Sobrescrever com variáveis de ambiente se definidas
	if provider := os.Getenv("AG_TRACING_PROVIDER"); provider != "" {
		cfg.Tracing.Provider = provider
	}

	if endpoint := os.Getenv("AG_TRACING_ENDPOINT"); endpoint != "" {
		cfg.Tracing.Endpoint = endpoint
	}

	// Verificar se está habilitado via env (prioridade sobre arquivo)
	if enabled := os.Getenv("AG_TRACING_ENABLED"); enabled != "" {
		cfg.Tracing.Enabled = (enabled == "true" || enabled == "1")
	}

	return cfg.Tracing, nil
}

// getTraceExporter seleciona o exportador de trace baseado na configuração
func getTraceExporter(ctx context.Context, endpointURL string, cfg config.TracingConfig, logger *zap.Logger) (sdktrace.SpanExporter, error) {
	// Usar endpointURL se fornecido como parâmetro, caso contrário usar da configuração
	if endpointURL == "" {
		endpointURL = cfg.Endpoint
	}

	// Log da configuração usada
	logger.Info("Configurando exportador de tracing",
		zap.String("provider", cfg.Provider),
		zap.String("endpoint", endpointURL),
		zap.Float64("sampling_ratio", cfg.SamplingRatio))

	switch cfg.Provider {
	case "otlp", "opentelemetry":
		// Conectar ao coletor OTLP
		conn, err := grpc.DialContext(ctx, endpointURL,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock())
		if err != nil {
			return nil, err
		}
		return otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))

	default:
		logger.Warn("Provedor de tracing desconhecido, usando OTLP como padrão",
			zap.String("provider", cfg.Provider))
		conn, err := grpc.DialContext(ctx, endpointURL,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock())
		if err != nil {
			return nil, err
		}
		return otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	}
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
