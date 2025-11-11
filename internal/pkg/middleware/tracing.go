package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/component-base/pkg/util/idutil"
)

const (
	traceHeaderKey = "X-Trace-Id"
	// XRequestIDKey 定义 X-Request-ID 键字符串
	XRequestIDKey = "X-Request-ID"
)

// Tracing 注入 trace_id/span_id/request_id 到请求上下文，便于链路追踪
func Tracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(traceHeaderKey)
		if traceID == "" {
			traceID = idutil.NewTraceID()
		}

		spanID := idutil.NewSpanID()

		requestID := c.GetString(XRequestIDKey)
		if requestID == "" {
			requestID = c.GetHeader(XRequestIDKey)
			if requestID == "" {
				requestID = idutil.NewRequestID()
			}
			c.Set(XRequestIDKey, requestID)
			c.Request.Header.Set(XRequestIDKey, requestID)
		}

		ctx := log.WithTraceContext(c.Request.Context(), traceID, spanID, requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Set(string(log.TraceIDKey), traceID)
		c.Set(string(log.SpanIDKey), spanID)
		c.Set(string(log.RequestIDKey), requestID)

		c.Writer.Header().Set(traceHeaderKey, traceID)
		c.Writer.Header().Set(XRequestIDKey, requestID)

		c.Next()
	}
}
