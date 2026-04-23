package observability

import "context"

// OTelTracingHook OpenTelemetry Tracing 钩子。
type OTelTracingHook struct {
	tracer OTelTracer
}

// NewOTelTracingHook 创建 OTel Tracing 钩子。
func NewOTelTracingHook(tracer OTelTracer) *OTelTracingHook {
	return &OTelTracingHook{tracer: tracer}
}

// StartSpan 实现 TracingHook 接口。
func (h *OTelTracingHook) StartSpan(ctx context.Context, name string) (context.Context, func()) {
	if h.tracer == nil {
		return ctx, func() {}
	}

	ctx, span := h.tracer.Start(ctx, name, WithSpanKind(SpanKindClient))
	return ctx, func() { span.End() }
}

// SetAttributes 实现 TracingHook 接口。
func (h *OTelTracingHook) SetAttributes(ctx context.Context, attrs map[string]string) {
	span := spanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attrs)
	}
}

// RecordError 实现 TracingHook 接口。
func (h *OTelTracingHook) RecordError(ctx context.Context, err error) {
	span := spanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
		span.SetStatus(SpanStatusError, err.Error())
	}
}

// ContextAwareOTelHook 支持 context 传递的 OTel 钩子。
type ContextAwareOTelHook struct {
	tracer OTelTracer
}

// NewContextAwareOTelHook 创建支持 context 的 OTel 钩子。
func NewContextAwareOTelHook(tracer OTelTracer) *ContextAwareOTelHook {
	return &ContextAwareOTelHook{tracer: tracer}
}

// StartSpan 开始 span 并存入 context。
func (h *ContextAwareOTelHook) StartSpan(ctx context.Context, name string) (context.Context, func()) {
	if h.tracer == nil {
		return ctx, func() {}
	}

	ctx, span := h.tracer.Start(ctx, name, WithSpanKind(SpanKindClient))
	ctx = contextWithSpan(ctx, span)
	return ctx, func() { span.End() }
}

// SetAttributes 从 context 获取 span 并设置属性。
func (h *ContextAwareOTelHook) SetAttributes(ctx context.Context, attrs map[string]string) {
	span := spanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attrs)
	}
}

// RecordError 从 context 获取 span 并记录错误。
func (h *ContextAwareOTelHook) RecordError(ctx context.Context, err error) {
	span := spanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
		span.SetStatus(SpanStatusError, err.Error())
	}
}
