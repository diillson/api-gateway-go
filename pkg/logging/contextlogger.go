package logging

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ContextLogger estende o zap.Logger com métodos que utilizam contexto
type ContextLogger struct {
	*zap.Logger
}

func NewLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	logger, err := config.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, err
	}
	return logger, nil
}

// With adiciona campos ao logger
func (l *ContextLogger) With(fields ...zap.Field) *ContextLogger {
	return &ContextLogger{Logger: l.Logger.With(fields...)}
}

// InfoCtx registra mensagens no nível info com contexto de rastreamento
func (l *ContextLogger) InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	l.Info(msg, l.addTraceFields(ctx, fields)...)
}

// ErrorCtx registra mensagens no nível error com contexto de rastreamento
func (l *ContextLogger) ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	l.Error(msg, l.addTraceFields(ctx, fields)...)
}

// WarnCtx registra mensagens no nível warn com contexto de rastreamento
func (l *ContextLogger) WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	l.Warn(msg, l.addTraceFields(ctx, fields)...)
}

// DebugCtx registra mensagens no nível debug com contexto de rastreamento
func (l *ContextLogger) DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	l.Debug(msg, l.addTraceFields(ctx, fields)...)
}

// FatalCtx registra mensagens no nível fatal com contexto de rastreamento
func (l *ContextLogger) FatalCtx(ctx context.Context, msg string, fields ...zap.Field) {
	l.Fatal(msg, l.addTraceFields(ctx, fields)...)
}

// addTraceFields adiciona informações de rastreamento aos campos do log
func (l *ContextLogger) addTraceFields(ctx context.Context, fields []zap.Field) []zap.Field {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()

		fields = append(fields,
			zap.String("trace_id", traceID),
			zap.String("span_id", spanID),
		)
	}

	return fields
}
