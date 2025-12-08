package observability

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ========== 熔断器 ==========

// CircuitState 熔断器状态
type CircuitState int

const (
	// CircuitClosed 关闭状态（正常）
	CircuitClosed CircuitState = iota
	// CircuitOpen 打开状态（熔断中）
	CircuitOpen
	// CircuitHalfOpen 半开状态（探测中）
	CircuitHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	// FailureThreshold 触发熔断的连续失败次数
	FailureThreshold int

	// OpenDuration 熔断器打开持续时间
	OpenDuration time.Duration

	// HalfOpenRequests 半开状态允许的请求数
	HalfOpenRequests int

	// FailureCodes 被视为失败的状态码
	FailureCodes []codes.Code

	// OnStateChange 状态变化回调
	OnStateChange func(from, to CircuitState)
}

// DefaultCircuitBreakerConfig 默认熔断器配置
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold: 5,
		OpenDuration:     30 * time.Second,
		HalfOpenRequests: 2,
		FailureCodes: []codes.Code{
			codes.Unavailable,
			codes.DeadlineExceeded,
			codes.ResourceExhausted,
		},
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	config *CircuitBreakerConfig

	mu              sync.RWMutex
	state           CircuitState
	failures        int
	successes       int
	lastStateChange time.Time
	halfOpenCount   int
}

// NewCircuitBreaker 创建熔断器
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

// State 获取当前状态
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Allow 检查是否允许请求
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// 检查是否可以转换到半开状态
		if time.Since(cb.lastStateChange) > cb.config.OpenDuration {
			cb.transitionTo(CircuitHalfOpen)
			cb.halfOpenCount = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		// 限制半开状态的请求数
		if cb.halfOpenCount < cb.config.HalfOpenRequests {
			cb.halfOpenCount++
			return true
		}
		return false
	}
	return false
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.successes++

	if cb.state == CircuitHalfOpen {
		// 半开状态下成功，转换到关闭状态
		cb.transitionTo(CircuitClosed)
	}
}

// RecordFailure 记录失败
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
		// 半开状态下失败，重新打开
		cb.transitionTo(CircuitOpen)
	}
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

// ========== 熔断器拦截器 ==========

// CircuitBreakerInterceptor 返回熔断器拦截器
func CircuitBreakerInterceptor(cb *CircuitBreaker) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		if !cb.Allow() {
			return status.Error(codes.Unavailable, "circuit breaker open")
		}

		err := invoker(ctx, method, req, reply, cc, opts...)

		if err != nil {
			if st, ok := status.FromError(err); ok {
				if cb.isFailureCode(st.Code()) {
					cb.RecordFailure()
					return err
				}
			}
		}

		cb.RecordSuccess()
		return err
	}
}

// PerMethodCircuitBreaker 按方法的熔断器管理器
type PerMethodCircuitBreaker struct {
	config   *CircuitBreakerConfig
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

// NewPerMethodCircuitBreaker 创建按方法的熔断器管理器
func NewPerMethodCircuitBreaker(config *CircuitBreakerConfig) *PerMethodCircuitBreaker {
	return &PerMethodCircuitBreaker{
		config:   config,
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetBreaker 获取方法对应的熔断器
func (p *PerMethodCircuitBreaker) GetBreaker(method string) *CircuitBreaker {
	p.mu.RLock()
	cb, ok := p.breakers[method]
	p.mu.RUnlock()

	if ok {
		return cb
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// 双重检查
	if cb, ok := p.breakers[method]; ok {
		return cb
	}

	cb = NewCircuitBreaker(p.config)
	p.breakers[method] = cb
	return cb
}

// PerMethodCircuitBreakerInterceptor 按方法熔断的拦截器
func PerMethodCircuitBreakerInterceptor(pm *PerMethodCircuitBreaker) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		cb := pm.GetBreaker(method)

		if !cb.Allow() {
			return status.Error(codes.Unavailable, "circuit breaker open for method: "+method)
		}

		err := invoker(ctx, method, req, reply, cc, opts...)

		if err != nil {
			if st, ok := status.FromError(err); ok {
				if cb.isFailureCode(st.Code()) {
					cb.RecordFailure()
					return err
				}
			}
		}

		cb.RecordSuccess()
		return err
	}
}
