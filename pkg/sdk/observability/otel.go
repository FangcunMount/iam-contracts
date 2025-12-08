// Package observability 提供可观测性支持
package observability

import (
	"context"
)

// =============================================================================
// OpenTelemetry 具体实现（轻量级，不引入 otel 依赖）
// =============================================================================

// OTelTracingHook OpenTelemetry Tracing 钩子
// 这是一个桥接层，允许用户注入真正的 OTel tracer
type OTelTracingHook struct {
	tracer OTelTracer
}

// OTelTracer OpenTelemetry Tracer 接口
// 用户实现此接口来桥接真正的 OTel tracer
type OTelTracer interface {
	// Start 开始一个新的 span
	Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)
}

// Span 表示一个追踪 span
type Span interface {
	// End 结束 span
	End()
	// SetAttributes 设置属性
	SetAttributes(attrs map[string]string)
	// RecordError 记录错误
	RecordError(err error)
	// SetStatus 设置状态
	SetStatus(code SpanStatusCode, description string)
}

// SpanOption span 选项
type SpanOption func(*SpanConfig)

// SpanConfig span 配置
type SpanConfig struct {
	Kind SpanKind
}

// SpanKind span 类型
type SpanKind int

const (
	SpanKindClient SpanKind = iota
	SpanKindServer
	SpanKindProducer
	SpanKindConsumer
	SpanKindInternal
)

// SpanStatusCode span 状态码
type SpanStatusCode int

const (
	SpanStatusUnset SpanStatusCode = iota
	SpanStatusOK
	SpanStatusError
)

// WithSpanKind 设置 span 类型
func WithSpanKind(kind SpanKind) SpanOption {
	return func(c *SpanConfig) {
		c.Kind = kind
	}
}

// NewOTelTracingHook 创建 OTel Tracing 钩子
func NewOTelTracingHook(tracer OTelTracer) *OTelTracingHook {
	return &OTelTracingHook{tracer: tracer}
}

// StartSpan 实现 TracingHook 接口
func (h *OTelTracingHook) StartSpan(ctx context.Context, name string) (context.Context, func()) {
	if h.tracer == nil {
		return ctx, func() {}
	}

	ctx, span := h.tracer.Start(ctx, name, WithSpanKind(SpanKindClient))
	return ctx, func() { span.End() }
}

// SetAttributes 实现 TracingHook 接口
func (h *OTelTracingHook) SetAttributes(ctx context.Context, attrs map[string]string) {
	span := spanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attrs)
	}
}

// RecordError 实现 TracingHook 接口
func (h *OTelTracingHook) RecordError(ctx context.Context, err error) {
	span := spanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
		span.SetStatus(SpanStatusError, err.Error())
	}
}

// =============================================================================
// 真正的 OTel 适配器示例（需要引入 go.opentelemetry.io/otel）
// =============================================================================

/*
下面是一个真正的 OTel 适配器实现示例。
由于不想在 SDK 中引入 otel 依赖，这里只作为文档示例：

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type RealOTelTracer struct {
	tracer trace.Tracer
}

func NewRealOTelTracer(name string) *RealOTelTracer {
	return &RealOTelTracer{
		tracer: otel.Tracer(name),
	}
}

func (t *RealOTelTracer) Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	ctx, span := t.tracer.Start(ctx, name, trace.WithSpanKind(trace.SpanKindClient))
	return ctx, &realSpan{span: span}
}

type realSpan struct {
	span trace.Span
}

func (s *realSpan) End() { s.span.End() }

func (s *realSpan) SetAttributes(attrs map[string]string) {
	for k, v := range attrs {
		s.span.SetAttributes(attribute.String(k, v))
	}
}

func (s *realSpan) RecordError(err error) {
	s.span.RecordError(err)
}

func (s *realSpan) SetStatus(code SpanStatusCode, desc string) {
	if code == SpanStatusError {
		s.span.SetStatus(codes.Error, desc)
	} else {
		s.span.SetStatus(codes.Ok, desc)
	}
}

// 使用示例：
// tracer := NewRealOTelTracer("iam-sdk")
// hook := NewOTelTracingHook(tracer)
// client, _ := sdk.NewClient(ctx, cfg, config.WithUnaryInterceptors(
//     observability.TracingUnaryInterceptor(hook),
// ))
*/

// =============================================================================
// Span 上下文管理
// =============================================================================

type spanContextKey struct{}

// contextWithSpan 将 span 存入 context
func contextWithSpan(ctx context.Context, span Span) context.Context {
	return context.WithValue(ctx, spanContextKey{}, span)
}

// spanFromContext 从 context 获取 span
func spanFromContext(ctx context.Context) Span {
	if span, ok := ctx.Value(spanContextKey{}).(Span); ok {
		return span
	}
	return nil
}

// =============================================================================
// 基于 context 的 OTel Tracing Hook（增强版）
// =============================================================================

// ContextAwareOTelHook 支持 context 传递的 OTel 钩子
type ContextAwareOTelHook struct {
	tracer OTelTracer
}

// NewContextAwareOTelHook 创建支持 context 的 OTel 钩子
func NewContextAwareOTelHook(tracer OTelTracer) *ContextAwareOTelHook {
	return &ContextAwareOTelHook{tracer: tracer}
}

// StartSpan 开始 span 并存入 context
func (h *ContextAwareOTelHook) StartSpan(ctx context.Context, name string) (context.Context, func()) {
	if h.tracer == nil {
		return ctx, func() {}
	}

	ctx, span := h.tracer.Start(ctx, name, WithSpanKind(SpanKindClient))
	ctx = contextWithSpan(ctx, span)
	return ctx, func() { span.End() }
}

// SetAttributes 从 context 获取 span 并设置属性
func (h *ContextAwareOTelHook) SetAttributes(ctx context.Context, attrs map[string]string) {
	span := spanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attrs)
	}
}

// RecordError 从 context 获取 span 并记录错误
func (h *ContextAwareOTelHook) RecordError(ctx context.Context, err error) {
	span := spanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
		span.SetStatus(SpanStatusError, err.Error())
	}
}
