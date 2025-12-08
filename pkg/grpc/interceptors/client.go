// Package interceptors 提供客户端认证拦截器
package interceptors

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ===== 客户端凭证注入拦截器 =====

// ClientCredentialProvider 客户端凭证提供者接口
type ClientCredentialProvider interface {
	// GetMetadata 获取要附加到请求的 metadata
	GetMetadata(ctx context.Context) (map[string]string, error)
}

// BearerTokenProvider Bearer Token 提供者
type BearerTokenProvider struct {
	// 获取 token 的函数
	GetToken func() (string, error)
}

// GetMetadata 实现 ClientCredentialProvider 接口
func (p *BearerTokenProvider) GetMetadata(ctx context.Context) (map[string]string, error) {
	token, err := p.GetToken()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"authorization": "Bearer " + token,
	}, nil
}

// HMACCredentialProvider HMAC 凭证提供者
type HMACCredentialProvider struct {
	AccessKey string
	SecretKey string
}

// GetMetadata 实现 ClientCredentialProvider 接口
func (p *HMACCredentialProvider) GetMetadata(ctx context.Context) (map[string]string, error) {
	nonce := generateNonce()
	return GenerateHMACCredentials(p.AccessKey, p.SecretKey, nonce), nil
}

// APIKeyProvider API Key 提供者
type APIKeyProvider struct {
	APIKey string
}

// GetMetadata 实现 ClientCredentialProvider 接口
func (p *APIKeyProvider) GetMetadata(ctx context.Context) (map[string]string, error) {
	return map[string]string{
		"x-api-key": p.APIKey,
	}, nil
}

// generateNonce 生成随机 nonce
func generateNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // crypto/rand.Read 在 Linux/Unix 上不会失败
	return hex.EncodeToString(b)
}

// ===== 客户端拦截器 =====

// ClientCredentialInterceptor 客户端凭证注入拦截器
func ClientCredentialInterceptor(provider ClientCredentialProvider) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 获取凭证 metadata
		md, err := provider.GetMetadata(ctx)
		if err != nil {
			return err
		}

		// 附加到上下文
		pairs := make([]string, 0, len(md)*2)
		for k, v := range md {
			pairs = append(pairs, k, v)
		}
		ctx = metadata.AppendToOutgoingContext(ctx, pairs...)

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// ClientCredentialStreamInterceptor 流式客户端凭证注入拦截器
func ClientCredentialStreamInterceptor(provider ClientCredentialProvider) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		md, err := provider.GetMetadata(ctx)
		if err != nil {
			return nil, err
		}

		pairs := make([]string, 0, len(md)*2)
		for k, v := range md {
			pairs = append(pairs, k, v)
		}
		ctx = metadata.AppendToOutgoingContext(ctx, pairs...)

		return streamer(ctx, desc, cc, method, opts...)
	}
}

// ===== 客户端重试拦截器 =====

// RetryOption 重试选项
type RetryOption struct {
	MaxRetries  int
	InitialWait time.Duration
	MaxWait     time.Duration
	Multiplier  float64
	// 判断是否应该重试的函数
	ShouldRetry func(err error) bool
}

// DefaultRetryOption 默认重试选项
func DefaultRetryOption() *RetryOption {
	return &RetryOption{
		MaxRetries:  3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     5 * time.Second,
		Multiplier:  2.0,
	}
}

// ClientRetryInterceptor 客户端重试拦截器
func ClientRetryInterceptor(opt *RetryOption) grpc.UnaryClientInterceptor {
	if opt == nil {
		opt = DefaultRetryOption()
	}

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var lastErr error
		wait := opt.InitialWait

		for attempt := 0; attempt <= opt.MaxRetries; attempt++ {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err == nil {
				return nil
			}

			lastErr = err

			// 检查是否应该重试
			if opt.ShouldRetry != nil && !opt.ShouldRetry(err) {
				return err
			}

			// 最后一次尝试不需要等待
			if attempt == opt.MaxRetries {
				break
			}

			// 等待后重试
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}

			// 指数退避
			wait = time.Duration(float64(wait) * opt.Multiplier)
			if wait > opt.MaxWait {
				wait = opt.MaxWait
			}
		}

		return lastErr
	}
}

// ===== 客户端超时拦截器 =====

// ClientTimeoutInterceptor 客户端超时拦截器
func ClientTimeoutInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 如果上下文已有超时设置，不覆盖
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// ===== 客户端日志拦截器 =====

// ClientLogInterceptor 客户端日志拦截器
func ClientLogInterceptor(logger InterceptorLogger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		duration := time.Since(start)

		fields := map[string]interface{}{
			"method":   method,
			"duration": duration.String(),
		}

		if err != nil {
			fields["error"] = err.Error()
			if logger != nil {
				logger.LogError("gRPC client call failed", fields)
			}
		} else {
			if logger != nil {
				logger.LogInfo("gRPC client call succeeded", fields)
			}
		}

		return err
	}
}
