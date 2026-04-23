package transport

import (
	"context"

	"google.golang.org/grpc/metadata"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	traceIDKey   contextKey = "trace_id"
)

// WithRequestID 设置请求 ID。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	ctx = context.WithValue(ctx, requestIDKey, requestID)

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	md = md.Copy()
	md.Set(MetadataKeyRequestID, requestID)

	return metadata.NewOutgoingContext(ctx, md)
}

// WithTraceID 设置追踪 ID。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	ctx = context.WithValue(ctx, traceIDKey, traceID)

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	md = md.Copy()
	md.Set(MetadataKeyTraceID, traceID)

	return metadata.NewOutgoingContext(ctx, md)
}

// GetRequestID 获取请求 ID。
func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if vals := md.Get(MetadataKeyRequestID); len(vals) > 0 {
			return vals[0]
		}
	}

	return ""
}

// GetTraceID 获取追踪 ID。
func GetTraceID(ctx context.Context) string {
	if v := ctx.Value(traceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if vals := md.Get(MetadataKeyTraceID); len(vals) > 0 {
			return vals[0]
		}
	}

	return ""
}
