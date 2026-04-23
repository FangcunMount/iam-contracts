package observability

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PerMethodCircuitBreaker 按方法的熔断器管理器。
type PerMethodCircuitBreaker struct {
	config   *CircuitBreakerConfig
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

// NewPerMethodCircuitBreaker 创建按方法的熔断器管理器。
func NewPerMethodCircuitBreaker(config *CircuitBreakerConfig) *PerMethodCircuitBreaker {
	return &PerMethodCircuitBreaker{
		config:   config,
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetBreaker 获取方法对应的熔断器。
func (p *PerMethodCircuitBreaker) GetBreaker(method string) *CircuitBreaker {
	p.mu.RLock()
	cb, ok := p.breakers[method]
	p.mu.RUnlock()
	if ok {
		return cb
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if cb, ok := p.breakers[method]; ok {
		return cb
	}

	cb = NewCircuitBreaker(p.config)
	p.breakers[method] = cb
	return cb
}

// PerMethodCircuitBreakerInterceptor 按方法熔断的拦截器。
func PerMethodCircuitBreakerInterceptor(pm *PerMethodCircuitBreaker) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		cb := pm.GetBreaker(method)
		if !cb.Allow() {
			return status.Error(codes.Unavailable, "circuit breaker open for method: "+method)
		}

		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			if st, ok := status.FromError(err); ok && cb.isFailureCode(st.Code()) {
				cb.RecordFailure()
				return err
			}
		}

		cb.RecordSuccess()
		return err
	}
}
