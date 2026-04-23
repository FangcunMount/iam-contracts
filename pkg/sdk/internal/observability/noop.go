package observability

import (
	"context"
	"time"
)

// NoopMetricsCollector 空操作指标收集器。
type NoopMetricsCollector struct{}

func (NoopMetricsCollector) RecordRequest(method string, code string, duration time.Duration) {}

// NoopTracingHook 空操作追踪钩子。
type NoopTracingHook struct{}

func (NoopTracingHook) StartSpan(ctx context.Context, name string) (context.Context, func()) {
	return ctx, func() {}
}

func (NoopTracingHook) SetAttributes(ctx context.Context, attrs map[string]string) {}

func (NoopTracingHook) RecordError(ctx context.Context, err error) {}
