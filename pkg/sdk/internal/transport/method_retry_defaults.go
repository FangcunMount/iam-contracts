package transport

import (
	"time"

	"google.golang.org/grpc/codes"
)

// DefaultMethodRetryConfig 默认重试配置。
func DefaultMethodRetryConfig() *MethodRetryConfig {
	return &MethodRetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableCodes:    DefaultRetryableCodes(),
	}
}

// ReadOnlyMethodConfig 只读方法配置（可重试）。
var ReadOnlyMethodConfig = &MethodConfig{
	Timeout: 10 * time.Second,
	Retry: &MethodRetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    50 * time.Millisecond,
		MaxBackoff:        2 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableCodes: []codes.Code{
			codes.Unavailable,
			codes.ResourceExhausted,
			codes.Aborted,
			codes.DeadlineExceeded,
		},
	},
}

// WriteMethodConfig 写方法配置（不重试）。
var WriteMethodConfig = &MethodConfig{
	Timeout: 30 * time.Second,
	Retry:   nil,
}

// LongRunningMethodConfig 长时间运行方法配置。
var LongRunningMethodConfig = &MethodConfig{
	Timeout: 5 * time.Minute,
	Retry: &MethodRetryConfig{
		MaxAttempts:       2,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableCodes: []codes.Code{
			codes.Unavailable,
		},
	},
}

// IdempotentMethodConfig 幂等方法配置（激进重试）。
var IdempotentMethodConfig = &MethodConfig{
	Timeout: 30 * time.Second,
	Retry: &MethodRetryConfig{
		MaxAttempts:       4,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        5 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableCodes:    IdempotentRetryableCodes(),
	},
}

// IdempotentRetryableCodes 幂等操作可重试的状态码。
func IdempotentRetryableCodes() []codes.Code {
	return []codes.Code{
		codes.Unavailable,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.DeadlineExceeded,
		codes.Unknown,
		codes.Internal,
	}
}

// NonIdempotentRetryableCodes 非幂等操作可重试的状态码。
func NonIdempotentRetryableCodes() []codes.Code {
	return []codes.Code{codes.Unavailable}
}

// DefaultRetryableCodes 默认可重试的状态码。
func DefaultRetryableCodes() []codes.Code {
	return []codes.Code{
		codes.Unavailable,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.DeadlineExceeded,
	}
}
