package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// LimitConfig configura o comportamento do limitador
type LimitConfig struct {
	Key         string        // Chave única para identificar o limite
	Limit       int           // Número máximo de requisições
	Period      time.Duration // Período de tempo para o limite
	BurstFactor float64       // Fator para permitir rajadas (1.0 = sem rajada)
}

// RedisLimiter implementa rate limiting usando Redis
type RedisLimiter struct {
	RedisClient *redis.Client
	logger      *zap.Logger
	tracer      trace.Tracer
}

// NewRedisLimiter cria um novo limitador baseado em Redis
func NewRedisLimiter(redisClient *redis.Client, logger *zap.Logger) *RedisLimiter {
	// Obter tracer para o rate limiter
	tracer := otel.GetTracerProvider().Tracer("api-gateway.ratelimit")

	return &RedisLimiter{
		RedisClient: redisClient,
		logger:      logger,
		tracer:      tracer,
	}
}

// Allow verifica se a requisição é permitida dentro do limite de taxa
// Retorna: permitido, limite, restante, tempo de reset, erro
func (r *RedisLimiter) Allow(ctx context.Context, config LimitConfig) (bool, int, int, time.Duration, error) {
	// Criar span para a operação de rate limiting
	ctx, span := r.tracer.Start(
		ctx,
		"RedisLimiter.Allow",
		trace.WithAttributes(
			attribute.String("ratelimit.key", config.Key),
			attribute.Int("ratelimit.limit", config.Limit),
			attribute.Int64("ratelimit.period_ms", config.Period.Milliseconds()),
			attribute.Float64("ratelimit.burst_factor", config.BurstFactor),
		),
	)
	defer span.End()

	if config.Limit <= 0 {
		span.SetStatus(codes.Error, "invalid limit")
		span.SetAttributes(attribute.Bool("error", true))
		return true, 0, 0, 0, errors.New("limite deve ser maior que zero")
	}

	if config.Period <= 0 {
		span.SetStatus(codes.Error, "invalid period")
		span.SetAttributes(attribute.Bool("error", true))
		return true, 0, 0, 0, errors.New("período deve ser maior que zero")
	}

	if config.BurstFactor <= 0 {
		config.BurstFactor = 1.0 // Default sem rajada
	}

	// Chave do Redis para este limite específico
	key := fmt.Sprintf("ratelimit:%s", config.Key)
	// Calcular o timestamp atual em segundos
	now := time.Now().Unix()
	// Calcular o timestamp de expiração deste período
	periodSeconds := int64(config.Period.Seconds())
	expireAt := now - (now % periodSeconds) + periodSeconds
	// Calcular o tempo restante até o reset
	resetAfter := time.Duration(expireAt-now) * time.Second

	// Executar o script de rate limiting no Redis
	script := redis.NewScript(`
            local key = KEYS[1]
            local limit = tonumber(ARGV[1])
            local expireAt = tonumber(ARGV[2])
            local ttl = expireAt - tonumber(ARGV[3])
    
            local count = redis.call('INCR', key)
            if count == 1 then
                redis.call('EXPIREAT', key, expireAt)
            end
    
            -- Calcular o número restante de requisições
            local remaining = limit - count
            
            return {count, remaining, ttl}
        `)

	// Executar o script
	result, err := script.Run(ctx, r.RedisClient, []string{key}, config.Limit, expireAt, now).Result()
	if err != nil {
		r.logger.Error("erro ao executar script de rate limit", zap.Error(err))
		// Registrar erro no span
		span.SetStatus(codes.Error, "redis script error")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", err.Error()),
		)

		return true, config.Limit, config.Limit, resetAfter, err
	}

	// Analisar o resultado
	resultArray, ok := result.([]interface{})
	if !ok || len(resultArray) != 3 {
		r.logger.Error("resultado inesperado do script de rate limit", zap.Any("result", result))

		// Registrar erro no span
		span.SetStatus(codes.Error, "unexpected result")
		span.SetAttributes(
			attribute.Bool("error", true),
			attribute.String("error.message", "resultado inválido do Redis"),
		)

		return true, config.Limit, config.Limit, resetAfter, errors.New("resultado inválido do Redis")
	}

	// Extrair os valores
	count, _ := strconv.Atoi(fmt.Sprintf("%v", resultArray[0]))
	remaining, _ := strconv.Atoi(fmt.Sprintf("%v", resultArray[1]))
	ttl, _ := strconv.ParseInt(fmt.Sprintf("%v", resultArray[2]), 10, 64)

	// Calcular o limite com burst
	burstLimit := int(float64(config.Limit) * config.BurstFactor)

	// Verificar se está dentro do limite (incluindo burst)
	allowed := count <= burstLimit

	// Adicionar resultados ao span
	span.SetAttributes(
		attribute.Int("ratelimit.count", count),
		attribute.Int("ratelimit.remaining", remaining),
		attribute.Int("ratelimit.burst_limit", burstLimit),
		attribute.Bool("ratelimit.allowed", allowed),
		attribute.Int64("ratelimit.reset_after_ms", int64(resetAfter.Milliseconds())),
	)

	if !allowed {
		span.SetStatus(codes.Error, "rate limit exceeded")
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return allowed, config.Limit, remaining, time.Duration(ttl) * time.Second, nil
}
