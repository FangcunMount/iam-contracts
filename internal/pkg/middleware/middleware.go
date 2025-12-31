package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	gindump "github.com/tpkeeper/gin-dump"
)

// Middlewares store registered middlewares.
// 存储注册的中间件
var Middlewares = defaultMiddlewares()

// NoCache 是一个中间件函数，用于添加头信息，防止客户端缓存 HTTP 响应
func NoCache(c *gin.Context) {
	c.Header("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate, value")
	c.Header("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
	c.Header("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	c.Next()
}

// Options 是一个中间件函数，用于添加头信息，处理 OPTIONS 请求，并中止中间件链和结束请求
func Options(c *gin.Context) {
	if c.Request.Method != "OPTIONS" {
		c.Next()
	} else {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "authorization, origin, content-type, accept")
		c.Header("Allow", "HEAD,GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Content-Type", "application/json")
		c.AbortWithStatus(http.StatusOK)
	}
}

// Secure 是一个中间件函数，用于添加安全头信息和资源访问头信息
func Secure(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Frame-Options", "DENY")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-XSS-Protection", "1; mode=block")

	if c.Request.TLS != nil {
		c.Header("Strict-Transport-Security", "max-age=31536000")
	}
}

// defaultMiddlewares 返回默认的中间件
func defaultMiddlewares() map[string]gin.HandlerFunc {
	return map[string]gin.HandlerFunc{
		"recovery": gin.Recovery(),
		"secure":   Secure,
		"options":  Options,
		"nocache":  NoCache,
		"tracing":  Tracing(), // 链路追踪中间件（整合了 request_id 生成）
		"dump":     gindump.Dump(),
		// 注意: 以下中间件已被更优的方案替代，建议使用 api_logger
		// "requestid":       RequestID(),        // 已被 tracing 中间件整合
		// "logger":          Logger(),           // 已被 api_logger 替代（Deprecated）
		// "enhanced_logger": EnhancedLogger(),   // 已被 api_logger 替代（Deprecated）
	}
}
