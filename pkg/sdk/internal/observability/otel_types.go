package observability

import "context"

// OTelTracer OpenTelemetry Tracer 接口。
type OTelTracer interface {
	Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)
}

// Span 表示一个追踪 span。
type Span interface {
	End()
	SetAttributes(attrs map[string]string)
	RecordError(err error)
	SetStatus(code SpanStatusCode, description string)
}

// SpanOption span 选项。
type SpanOption func(*SpanConfig)

// SpanConfig span 配置。
type SpanConfig struct {
	Kind SpanKind
}

// SpanKind span 类型。
type SpanKind int

const (
	SpanKindClient SpanKind = iota
	SpanKindServer
	SpanKindProducer
	SpanKindConsumer
	SpanKindInternal
)

// SpanStatusCode span 状态码。
type SpanStatusCode int

const (
	SpanStatusUnset SpanStatusCode = iota
	SpanStatusOK
	SpanStatusError
)

// WithSpanKind 设置 span 类型。
func WithSpanKind(kind SpanKind) SpanOption {
	return func(c *SpanConfig) {
		c.Kind = kind
	}
}
