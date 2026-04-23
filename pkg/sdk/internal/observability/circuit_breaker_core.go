package observability

import (
	"sync"
	"time"

	"google.golang.org/grpc/codes"
)

// CircuitBreakerConfig 熔断器配置。
type CircuitBreakerConfig struct {
	FailureThreshold int
	OpenDuration     time.Duration
	HalfOpenRequests int
	FailureCodes     []codes.Code
	OnStateChange    func(from, to CircuitState)
	SuccessThreshold int
}

// DefaultCircuitBreakerConfig 默认熔断器配置。
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold: 5,
		OpenDuration:     30 * time.Second,
		HalfOpenRequests: 2,
		SuccessThreshold: 1,
		FailureCodes: []codes.Code{
			codes.Unavailable,
			codes.DeadlineExceeded,
			codes.ResourceExhausted,
		},
	}
}

// CircuitBreaker 熔断器。
type CircuitBreaker struct {
	config *CircuitBreakerConfig

	mu              sync.RWMutex
	state           CircuitState
	failures        int
	successes       int
	lastStateChange time.Time
	halfOpenCount   int
}

// NewCircuitBreaker 创建熔断器。
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}
	return &CircuitBreaker{
		config:          config,
		state:           CircuitClosed,
		lastStateChange: time.Now(),
	}
}

// State 获取当前状态。
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Allow 检查是否允许请求。
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastStateChange) > cb.config.OpenDuration {
			cb.transitionTo(CircuitHalfOpen)
			cb.halfOpenCount = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		if cb.halfOpenCount < cb.config.HalfOpenRequests {
			cb.halfOpenCount++
			return true
		}
		return false
	default:
		return false
	}
}

// RecordSuccess 记录成功。
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.successes++

	if cb.state == CircuitHalfOpen && cb.successes >= cb.successThreshold() {
		cb.transitionTo(CircuitClosed)
	}
}

// RecordFailure 记录失败。
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.successes = 0

	switch cb.state {
	case CircuitClosed:
		if cb.failures >= cb.config.FailureThreshold {
			cb.transitionTo(CircuitOpen)
		}
	case CircuitHalfOpen:
		cb.transitionTo(CircuitOpen)
	}
}

func (cb *CircuitBreaker) successThreshold() int {
	if cb.config.SuccessThreshold <= 0 {
		return 1
	}
	return cb.config.SuccessThreshold
}

func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	if cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(oldState, newState)
	}
}

func (cb *CircuitBreaker) isFailureCode(code codes.Code) bool {
	for _, fc := range cb.config.FailureCodes {
		if fc == code {
			return true
		}
	}
	return false
}
