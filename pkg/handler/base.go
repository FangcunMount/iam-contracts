// Package handler 提供 RESTful API 处理器的通用功能
package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/pkg/core"
)

// BaseHandler 提供统一的响应和参数绑定能力，避免各 handler 重复编写样板代码
type BaseHandler struct{}

// NewBaseHandler 创建基础 Handler
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{}
}

// Success 写出成功响应
func (h *BaseHandler) Success(c *gin.Context, data interface{}) {
	core.WriteResponse(c, nil, data)
}

// SuccessWithMessage 写出带消息的成功响应
func (h *BaseHandler) SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	if message == "" {
		h.Success(c, data)
		return
	}

	core.WriteResponse(c, nil, gin.H{
		"message": message,
		"data":    data,
	})
}

// Created 写出 201 响应
func (h *BaseHandler) Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// NoContent 写出 204 响应
func (h *BaseHandler) NoContent(c *gin.Context) {
	c.AbortWithStatus(http.StatusNoContent)
}

// Error 写出错误响应
func (h *BaseHandler) Error(c *gin.Context, err error) {
	if err == nil {
		h.Success(c, nil)
		return
	}
	core.WriteResponse(c, err, nil)
}

// ErrorWithCode 使用业务错误码构造错误并写出
func (h *BaseHandler) ErrorWithCode(c *gin.Context, errCode int, format string, args ...interface{}) {
	err := perrors.WithCode(errCode, format, args...)
	h.Error(c, err)
}

// BindJSON 绑定 JSON 请求体，返回统一封装的错误
func (h *BaseHandler) BindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return perrors.WithCode(code.ErrBind, "invalid JSON payload: %v", err)
	}
	return nil
}

// BindQuery 绑定查询参数，返回统一封装的错误
func (h *BaseHandler) BindQuery(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindQuery(obj); err != nil {
		return perrors.WithCode(code.ErrBind, "invalid query payload: %v", err)
	}
	return nil
}

// BindURI 绑定 URI 参数，返回统一封装的错误
func (h *BaseHandler) BindURI(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindUri(obj); err != nil {
		return perrors.WithCode(code.ErrBind, "invalid URI payload: %v", err)
	}
	return nil
}

// GetPathParam 获取路径参数
func (h *BaseHandler) GetPathParam(c *gin.Context, key string) string {
	return c.Param(key)
}

// GetQueryParam 获取查询参数
func (h *BaseHandler) GetQueryParam(c *gin.Context, key string) string {
	return c.Query(key)
}

// GetQueryParamInt 获取整数查询参数
func (h *BaseHandler) GetQueryParamInt(c *gin.Context, key string, defaultValue int) int {
	value := c.Query(key)
	if value == "" {
		return defaultValue
	}

	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	}

	return defaultValue
}

// GetUserID 从上下文获取当前用户ID
// 支持多种类型的用户ID，并统一转换为字符串
func (h *BaseHandler) GetUserID(c *gin.Context) (string, bool) {
	// 尝试从不同的上下文键获取用户ID
	for _, key := range []string{"user_id", "userID", "uid"} {
		userID, exists := c.Get(key)
		if !exists || userID == nil {
			continue
		}

		switch v := userID.(type) {
		case string:
			if v != "" {
				return v, true
			}
		case fmt.Stringer:
			return v.String(), true
		case uint64:
			return strconv.FormatUint(v, 10), true
		case uint32:
			return strconv.FormatUint(uint64(v), 10), true
		case uint:
			return strconv.FormatUint(uint64(v), 10), true
		case int64:
			if v >= 0 {
				return strconv.FormatInt(v, 10), true
			}
		case int32:
			if v >= 0 {
				return strconv.FormatInt(int64(v), 10), true
			}
		case int:
			if v >= 0 {
				return strconv.FormatInt(int64(v), 10), true
			}
		default:
			str := fmt.Sprintf("%v", v)
			if str != "" && str != "<nil>" {
				return str, true
			}
		}
	}

	return "", false
}

// GetTenantID 从上下文中获取租户 ID（多租户场景）
func (h *BaseHandler) GetTenantID(c *gin.Context) string {
	// 优先从上下文获取
	tenantID, exists := c.Get("tenant_id")
	if exists {
		if tid, ok := tenantID.(string); ok && tid != "" {
			return tid
		}
	}

	// 其次从 Header 获取
	if tid := c.GetHeader("X-Tenant-ID"); tid != "" {
		return tid
	}

	// 返回默认租户
	return "default"
}

// ParseUint 解析字符串为 uint64，带验证
func ParseUint(raw string, field string) (uint64, error) {
	if raw == "" {
		return 0, perrors.WithCode(code.ErrInvalidArgument, "%s cannot be empty", field)
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, perrors.WithCode(code.ErrInvalidArgument, "invalid %s: %s", field, raw)
	}
	return value, nil
}

// ParseInt 解析字符串为 int64，带验证
func ParseInt(raw string, field string) (int64, error) {
	if raw == "" {
		return 0, perrors.WithCode(code.ErrInvalidArgument, "%s cannot be empty", field)
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, perrors.WithCode(code.ErrInvalidArgument, "invalid %s: %s", field, raw)
	}
	return value, nil
}
