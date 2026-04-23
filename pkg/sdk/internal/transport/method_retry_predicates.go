package transport

import (
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetryPredicate 重试判断函数。
type RetryPredicate func(err error, attempt int) bool

// RetryPredicateConfig 带自定义判断的重试配置。
type RetryPredicateConfig struct {
	*MethodRetryConfig
	Predicate RetryPredicate
	OnRetry   func(attempt int, err error, backoff time.Duration)
}

// WithRetryPredicate 包装重试配置添加自定义判断。
func WithRetryPredicate(cfg *MethodRetryConfig, predicate RetryPredicate) *RetryPredicateConfig {
	return &RetryPredicateConfig{
		MethodRetryConfig: cfg,
		Predicate:         predicate,
	}
}

// RetryOnNetworkError 网络错误时重试。
func RetryOnNetworkError() RetryPredicate {
	return func(err error, attempt int) bool {
		st, ok := status.FromError(err)
		if !ok {
			return false
		}
		switch st.Code() {
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return true
		}
		return false
	}
}

// RetryOnServerError 服务端错误时重试。
func RetryOnServerError() RetryPredicate {
	return func(err error, attempt int) bool {
		st, ok := status.FromError(err)
		if !ok {
			return false
		}
		switch st.Code() {
		case codes.Internal, codes.Unknown, codes.DataLoss:
			return true
		}
		return false
	}
}

// RetryOnRateLimit 限流时重试。
func RetryOnRateLimit(maxAttempts int) RetryPredicate {
	return func(err error, attempt int) bool {
		if attempt >= maxAttempts {
			return false
		}
		st, ok := status.FromError(err)
		if !ok {
			return false
		}
		return st.Code() == codes.ResourceExhausted
	}
}

// CombinePredicates 组合多个重试判断。
func CombinePredicates(predicates ...RetryPredicate) RetryPredicate {
	return func(err error, attempt int) bool {
		for _, p := range predicates {
			if p(err, attempt) {
				return true
			}
		}
		return false
	}
}
