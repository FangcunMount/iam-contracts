package observability

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CircuitBreakerInterceptor 返回熔断器拦截器。
func CircuitBreakerInterceptor(cb *CircuitBreaker) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if !cb.Allow() {
			return status.Error(codes.Unavailable, "circuit breaker open")
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
