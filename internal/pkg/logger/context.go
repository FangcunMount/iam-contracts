// Package logger 提供请求范围的日志工具
// 支持通过 context 传递 Logger，确保整个请求链路的日志都带有统一的追踪信息
package logger

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/log"
	"go.uber.org/zap/zapcore"
)

// ctxLoggerKey 是用于在 context 中存储 Logger 的键
type ctxLoggerKey struct{}

// RequestLogger 请求范围的日志记录器
// 封装了 log.Logger，预设了追踪字段
type RequestLogger struct {
	fields []log.Field
}

// NewRequestLogger 创建请求范围的 Logger
// 自动从 context 中提取 trace_id, span_id, request_id 等追踪信息
func NewRequestLogger(ctx context.Context, fields ...log.Field) *RequestLogger {
	// 从 context 获取追踪字段
	baseFields := log.TraceFields(ctx)
	// 合并自定义字段
	allFields := make([]log.Field, 0, len(baseFields)+len(fields))
	allFields = append(allFields, baseFields...)
	allFields = append(allFields, fields...)

	return &RequestLogger{
		fields: allFields,
	}
}

// WithLogger 将 RequestLogger 放入 context
func WithLogger(ctx context.Context, logger *RequestLogger) context.Context {
	return context.WithValue(ctx, ctxLoggerKey{}, logger)
}

// L 从 context 获取 RequestLogger
// 如果 context 中没有 Logger，返回一个带有追踪信息的默认 Logger
func L(ctx context.Context) *RequestLogger {
	if logger, ok := ctx.Value(ctxLoggerKey{}).(*RequestLogger); ok {
		return logger
	}
	// 返回一个新的 Logger，尝试从 ctx 获取追踪信息
	return NewRequestLogger(ctx)
}

// WithFields 创建一个带有额外字段的新 Logger
func (l *RequestLogger) WithFields(fields ...log.Field) *RequestLogger {
	newFields := make([]log.Field, 0, len(l.fields)+len(fields))
	newFields = append(newFields, l.fields...)
	newFields = append(newFields, fields...)
	return &RequestLogger{fields: newFields}
}

// WithField 创建一个带有单个额外字段的新 Logger
func (l *RequestLogger) WithField(key string, value interface{}) *RequestLogger {
	return l.WithFields(log.Any(key, value))
}

// Debug 记录 Debug 级别日志
func (l *RequestLogger) Debug(msg string, fields ...log.Field) {
	allFields := l.mergeFields(fields)
	log.Debugw(msg, fieldsToKV(allFields)...)
}

// Debugw 记录 Debug 级别日志（key-value 格式）
func (l *RequestLogger) Debugw(msg string, keysAndValues ...interface{}) {
	kvs := l.prependFields(keysAndValues)
	log.Debugw(msg, kvs...)
}

// Info 记录 Info 级别日志
func (l *RequestLogger) Info(msg string, fields ...log.Field) {
	allFields := l.mergeFields(fields)
	log.Infow(msg, fieldsToKV(allFields)...)
}

// Infow 记录 Info 级别日志（key-value 格式）
func (l *RequestLogger) Infow(msg string, keysAndValues ...interface{}) {
	kvs := l.prependFields(keysAndValues)
	log.Infow(msg, kvs...)
}

// Warn 记录 Warn 级别日志
func (l *RequestLogger) Warn(msg string, fields ...log.Field) {
	allFields := l.mergeFields(fields)
	log.Warnw(msg, fieldsToKV(allFields)...)
}

// Warnw 记录 Warn 级别日志（key-value 格式）
func (l *RequestLogger) Warnw(msg string, keysAndValues ...interface{}) {
	kvs := l.prependFields(keysAndValues)
	log.Warnw(msg, kvs...)
}

// Error 记录 Error 级别日志
func (l *RequestLogger) Error(msg string, fields ...log.Field) {
	allFields := l.mergeFields(fields)
	log.Errorw(msg, fieldsToKV(allFields)...)
}

// Errorw 记录 Error 级别日志（key-value 格式）
func (l *RequestLogger) Errorw(msg string, keysAndValues ...interface{}) {
	kvs := l.prependFields(keysAndValues)
	log.Errorw(msg, kvs...)
}

// mergeFields 合并基础字段和额外字段
func (l *RequestLogger) mergeFields(fields []log.Field) []log.Field {
	if len(fields) == 0 {
		return l.fields
	}
	allFields := make([]log.Field, 0, len(l.fields)+len(fields))
	allFields = append(allFields, l.fields...)
	allFields = append(allFields, fields...)
	return allFields
}

// prependFields 将基础字段转换为 key-value 并前置到参数列表
func (l *RequestLogger) prependFields(keysAndValues []interface{}) []interface{} {
	baseKV := fieldsToKV(l.fields)
	result := make([]interface{}, 0, len(baseKV)+len(keysAndValues))
	result = append(result, baseKV...)
	result = append(result, keysAndValues...)
	return result
}

// fieldsToKV 将 log.Field 切片转换为 key-value 切片
func fieldsToKV(fields []log.Field) []interface{} {
	if len(fields) == 0 {
		return nil
	}
	kv := make([]interface{}, 0, len(fields)*2)
	for _, f := range fields {
		kv = append(kv, f.Key, fieldValue(f))
	}
	return kv
}

// fieldValue 从 zapcore.Field 中提取值
func fieldValue(f log.Field) interface{} {
	switch f.Type {
	case zapcore.StringType:
		return f.String
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		return f.Integer
	case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
		return uint64(f.Integer)
	case zapcore.Float64Type:
		return float64(f.Integer)
	case zapcore.Float32Type:
		return float32(f.Integer)
	case zapcore.BoolType:
		return f.Integer == 1
	case zapcore.DurationType:
		return f.Integer // 纳秒
	case zapcore.TimeType, zapcore.TimeFullType:
		return f.Interface
	default:
		// 对于其他类型（如 ObjectMarshalerType、ArrayMarshalerType 等），返回 Interface
		if f.Interface != nil {
			return f.Interface
		}
		return f.String
	}
}
