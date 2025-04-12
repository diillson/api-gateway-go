package resilience

import (
	"context"
	"errors"
	"github.com/diillson/api-gateway-go/internal/infra/metrics"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	// ErrCircuitOpen é retornado quando o circuit breaker está aberto
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// CircuitState representa os estados possíveis do circuit breaker
type CircuitState int

const (
	StateClose CircuitState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreakerConfig contém a configuração do circuit breaker
type CircuitBreakerConfig struct {
	Name            string
	MaxRequestsFail int           // Número máximo de falhas antes de abrir o circuito
	Interval        time.Duration // Intervalo no qual contar falhas
	Timeout         time.Duration // Tempo que o circuito fica aberto antes de tentar half-open
	MaxRequests     int           // Número máximo de requisições no estado half-open
}

// CircuitBreaker implementa o pattern Circuit Breaker
type CircuitBreaker struct {
	name        string
	maxFails    int
	interval    time.Duration
	timeout     time.Duration
	maxRequests int

	mutex               sync.RWMutex
	state               CircuitState
	failCount           int
	lastStateChangeTime time.Time
	nextAttemptTime     time.Time
	halfOpenRequests    int

	logger  *zap.Logger
	metrics *metrics.APIMetrics
}

// NewCircuitBreaker cria um novo circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig, logger *zap.Logger, metrics *metrics.APIMetrics) *CircuitBreaker {
	// Valores padrão se não especificados
	if config.MaxRequestsFail <= 0 {
		config.MaxRequestsFail = 5
	}
	if config.Interval <= 0 {
		config.Interval = time.Minute
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRequests <= 0 {
		config.MaxRequests = 1
	}

	return &CircuitBreaker{
		name:                config.Name,
		maxFails:            config.MaxRequestsFail,
		interval:            config.Interval,
		timeout:             config.Timeout,
		maxRequests:         config.MaxRequests,
		state:               StateClose,
		lastStateChangeTime: time.Now(),
		logger:              logger,
		metrics:             metrics,
	}
}

// Execute executa a função com circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	if !cb.allowRequest() {
		return nil, ErrCircuitOpen
	}

	// Permitir a requisição
	result, err := fn(ctx)

	// Atualizar o estado do circuit breaker com base no resultado
	cb.recordResult(err == nil)

	return result, err
}

// allowRequest verifica se a requisição deve ser permitida com base no estado atual
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	now := time.Now()

	switch cb.state {
	case StateClose:
		return true

	case StateOpen:
		// Verificar se o tempo de timeout passou para tentar half-open
		if now.After(cb.nextAttemptTime) {
			// Precisamos adquirir um lock de escrita para mudar o estado
			cb.mutex.RUnlock()
			cb.mutex.Lock()
			defer cb.mutex.Unlock()

			// Verificar novamente para evitar race condition
			if cb.state == StateOpen && now.After(cb.nextAttemptTime) {
				cb.toHalfOpen(now)
				return true
			}
			return false
		}
		return false

	case StateHalfOpen:
		// Permitir um número limitado de requisições no estado half-open
		return cb.halfOpenRequests < cb.maxRequests
	}

	return false
}

// recordResult atualiza o estado do circuit breaker com base no resultado da requisição
func (cb *CircuitBreaker) recordResult(success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClose:
		if !success {
			// Incrementar contador de falhas
			cb.failCount++
			cb.logger.Debug("circuit breaker registrou falha",
				zap.String("name", cb.name),
				zap.Int("failCount", cb.failCount),
				zap.Int("maxFails", cb.maxFails))

			// Se passar do limite, abrir o circuito
			if cb.failCount >= cb.maxFails {
				cb.toOpen(now)
			}
		} else {
			// Reset contador de falhas em caso de sucesso
			cb.failCount = 0
		}

	case StateHalfOpen:
		if success {
			// Se sucesso no half-open, voltar para fechado
			cb.toClose(now)
		} else {
			// Se falha no half-open, voltar para aberto
			cb.toOpen(now)
		}
	}
}

// toOpen muda o estado para open
func (cb *CircuitBreaker) toOpen(now time.Time) {
	cb.state = StateOpen
	cb.lastStateChangeTime = now
	cb.nextAttemptTime = now.Add(cb.timeout)

	// Registrar a mudança de estado nas métricas
	if cb.metrics != nil {
		cb.metrics.CircuitBreakerStateChanged(cb.name, true)
	}

	cb.logger.Info("circuit breaker mudou para estado aberto",
		zap.String("name", cb.name),
		zap.Time("nextAttempt", cb.nextAttemptTime))
}

// toHalfOpen muda o estado para half-open
func (cb *CircuitBreaker) toHalfOpen(now time.Time) {
	cb.state = StateHalfOpen
	cb.lastStateChangeTime = now
	cb.halfOpenRequests = 0
	cb.logger.Info("circuit breaker mudou para estado meio-aberto", zap.String("name", cb.name))
}

// toClose muda o estado para close
func (cb *CircuitBreaker) toClose(now time.Time) {
	cb.state = StateClose
	cb.lastStateChangeTime = now
	cb.failCount = 0

	// Registrar a mudança de estado nas métricas
	if cb.metrics != nil {
		cb.metrics.CircuitBreakerStateChanged(cb.name, false)
	}

	cb.logger.Info("circuit breaker mudou para estado fechado", zap.String("name", cb.name))
}

// GetState retorna o estado atual do circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Reset reseta o circuit breaker para o estado fechado
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.toClose(time.Now())
}
