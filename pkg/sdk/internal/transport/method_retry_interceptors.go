package transport

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// TimeoutUnaryInterceptor 按方法超时拦截器。
func TimeoutUnaryInterceptor(configs *MethodConfigs) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		cfg := configs.GetConfig(method)
		if cfg == nil || cfg.Timeout <= 0 {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
			defer cancel()
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// RetryUnaryInterceptor 按方法重试拦截器。
func RetryUnaryInterceptor(configs *MethodConfigs) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		cfg := configs.GetConfig(method)
		if cfg == nil || cfg.Retry == nil || cfg.Retry.MaxAttempts <= 1 {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		retryCfg := cfg.Retry
		var lastErr error
		for attempt := 0; attempt < retryCfg.MaxAttempts; attempt++ {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err == nil {
				return nil
			}

			lastErr = err
			if !isRetryable(err, retryCfg.RetryableCodes) {
				return err
			}
			if attempt == retryCfg.MaxAttempts-1 {
				break
			}

			backoff := calculateBackoff(attempt, retryCfg)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		return lastErr
	}
}

// RetryPredicateUnaryInterceptor 带自定义判断的重试拦截器。
func RetryPredicateUnaryInterceptor(configs map[string]*RetryPredicateConfig, defaultCfg *RetryPredicateConfig) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		cfg := configs[method]
		if cfg == nil {
			cfg = defaultCfg
		}
		if cfg == nil || cfg.MaxAttempts <= 1 {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		var lastErr error
		for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err == nil {
				return nil
			}

			lastErr = err

			shouldRetry := false
			if cfg.Predicate != nil {
				shouldRetry = cfg.Predicate(err, attempt)
			} else {
				shouldRetry = isRetryable(err, cfg.RetryableCodes)
			}
			if !shouldRetry {
				return err
			}
			if attempt == cfg.MaxAttempts-1 {
				break
			}

			backoff := calculateBackoff(attempt, cfg.MethodRetryConfig)
			if cfg.OnRetry != nil {
				cfg.OnRetry(attempt+1, err, backoff)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		return lastErr
	}
}
