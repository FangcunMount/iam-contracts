package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/gin-gonic/gin"
)

const defaultAPILogTag = "http.access"

// APILoggerConfig 定义 API 日志中间件的可配置项
type APILoggerConfig struct {
	Tag                string
	SkipPaths          []string
	LogRequestHeaders  bool
	LogRequestBody     bool
	LogResponseHeaders bool
	LogResponseBody    bool
	MaskSensitiveData  bool
	MaxBodyBytes       int64
}

// DefaultAPILoggerConfig 返回默认配置
func DefaultAPILoggerConfig() APILoggerConfig {
	return APILoggerConfig{
		Tag:                defaultAPILogTag,
		SkipPaths:          []string{"/health", "/healthz", "/metrics", "/favicon.ico"},
		LogRequestHeaders:  true,
		LogRequestBody:     true,
		LogResponseHeaders: true,
		LogResponseBody:    true,
		MaskSensitiveData:  true,
		MaxBodyBytes:       16 * 1024, // 16KB
	}
}

// APILogger 详细 API 日志中间件
func APILogger() gin.HandlerFunc {
	return APILoggerWithConfig(DefaultAPILoggerConfig())
}

// APILoggerWithConfig 带配置的 API 日志中间件
func APILoggerWithConfig(config APILoggerConfig) gin.HandlerFunc {
	cfg := config.withDefaults()
	skipPaths := buildSkipMap(cfg.SkipPaths)

	return func(c *gin.Context) {
		if _, ok := skipPaths[c.Request.URL.Path]; ok {
			c.Next()
			return
		}

		start := time.Now()

		writer := newBodyCaptureWriter(c.Writer, cfg.LogResponseBody, cfg.MaxBodyBytes)
		c.Writer = writer

		c.Next()

		statusCode := writer.Status()
		latency := time.Since(start)

		// 使用简洁的格式化日志，而不是结构化日志
		logMsg := fmt.Sprintf("%-6s %s %d | %6.3fms | %s | %s",
			c.Request.Method,
			c.Request.URL.Path,
			statusCode,
			float64(latency.Microseconds())/1000.0,
			c.ClientIP(),
			c.Request.UserAgent(),
		)

		switch {
		case statusCode >= http.StatusInternalServerError:
			log.Errorf("[API] %s", logMsg)
			if len(c.Errors) > 0 {
				log.Errorf("[API] Errors: %s", c.Errors.String())
			}
		case statusCode >= http.StatusBadRequest:
			log.Warnf("[API] %s", logMsg)
		default:
			log.Infof("[API] %s", logMsg)
		}
	}
}

func (cfg APILoggerConfig) withDefaults() APILoggerConfig {
	result := cfg

	if result.Tag == "" {
		result.Tag = defaultAPILogTag
	}
	if result.MaxBodyBytes <= 0 {
		result.MaxBodyBytes = DefaultAPILoggerConfig().MaxBodyBytes
	}

	return result
}

func buildSkipMap(paths []string) map[string]struct{} {
	if len(paths) == 0 {
		return map[string]struct{}{}
	}

	skip := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		skip[path] = struct{}{}
	}

	return skip
}

type bodyCaptureWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	limitBytes int64
	capture    bool
}

func newBodyCaptureWriter(w gin.ResponseWriter, capture bool, limit int64) *bodyCaptureWriter {
	var buffer *bytes.Buffer
	if capture {
		buffer = &bytes.Buffer{}
	}

	return &bodyCaptureWriter{
		ResponseWriter: w,
		body:           buffer,
		statusCode:     w.Status(),
		limitBytes:     limit,
		capture:        capture,
	}
}

func (w *bodyCaptureWriter) Write(data []byte) (int, error) {
	if w.capture && w.body != nil && len(data) > 0 {
		if w.limitBytes <= 0 || int64(w.body.Len()) < w.limitBytes {
			remaining := len(data)
			if w.limitBytes > 0 {
				remaining = int(minInt64(w.limitBytes-int64(w.body.Len()), int64(len(data))))
			}
			if remaining > 0 {
				w.body.Write(data[:remaining])
			}
		}
	}

	return w.ResponseWriter.Write(data)
}

func (w *bodyCaptureWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *bodyCaptureWriter) Status() int {
	return w.statusCode
}

func (w *bodyCaptureWriter) Body() []byte {
	if !w.capture || w.body == nil {
		return nil
	}
	return w.body.Bytes()
}

func minInt64(a int64, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// 以下函数保留以备将来需要详细日志时使用

/*
func readRequestBody(c *gin.Context) []byte {
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.L(c.Request.Context()).Warnw("failed to read request body", "error", err)
		return nil
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(data))
	return data
}

func captureHeaders(headers http.Header, mask bool) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	result := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}
		value := strings.Join(values, ", ")
		if mask && isSensitiveHeader(key) {
			value = maskSensitiveValue(value)
		}
		result[key] = value
	}

	return result
}

func renderBodyForLog(data []byte, actualLen int, max int64, mask bool) string {
	if actualLen <= 0 {
		actualLen = len(data)
	}
	if actualLen == 0 {
		return ""
	}
	if max > 0 && int64(actualLen) > max {
		return fmt.Sprintf("[omitted body: %d bytes exceeds limit %d bytes]", actualLen, max)
	}
	if len(data) == 0 {
		return ""
	}

	var body string
	if isJSON(data) {
		body = formatJSON(data)
		if mask {
			body = maskSensitiveJSON(body)
		}
	} else {
		body = string(data)
		if mask {
			body = maskSensitiveWithRegex(body)
		}
	}

	return body
}

// isJSON 检查数据是否为 JSON 格式
func isJSON(data []byte) bool {
	var js json.RawMessage
	return json.Unmarshal(data, &js) == nil
}

// formatJSON 格式化 JSON 数据（移除不必要的空格和换行）
func formatJSON(data []byte) string {
	var compact bytes.Buffer
	if err := json.Compact(&compact, data); err != nil {
		return string(data)
	}
	result := compact.String()
	if len(result) > 500 {
		return result[:500] + "..."
	}
	return result
}
*/
