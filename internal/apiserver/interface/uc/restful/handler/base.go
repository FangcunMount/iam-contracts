package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/middleware/authn"
	"github.com/FangcunMount/iam-contracts/pkg/core"
)

// BaseHandler 提供统一的响应和参数绑定能力，避免各 handler 重复编写样板代码。
type BaseHandler struct{}

// NewBaseHandler 创建基础 Handler。
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{}
}

// Success 写出成功响应。
func (h *BaseHandler) Success(c *gin.Context, data interface{}) {
	core.WriteResponse(c, nil, data)
}

// SuccessWithMessage 写出带消息的成功响应。
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

// Created 写出 201 响应。
func (h *BaseHandler) Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// Error 写出错误响应。
func (h *BaseHandler) Error(c *gin.Context, err error) {
	if err == nil {
		h.Success(c, nil)
		return
	}

	core.WriteResponse(c, err, nil)
}

// ErrorWithCode 使用业务错误码构造错误并写出。
func (h *BaseHandler) ErrorWithCode(c *gin.Context, code int, format string, args ...interface{}) {
	h.Error(c, perrors.WithCode(code, format, args...))
}

// BindJSON 绑定 JSON 请求体，返回统一封装的错误。
func (h *BaseHandler) BindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return perrors.WithCode(code.ErrBind, "invalid JSON payload: %v", err)
	}
	return nil
}

// BindQuery 绑定查询参数，返回统一封装的错误。
func (h *BaseHandler) BindQuery(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindQuery(obj); err != nil {
		return perrors.WithCode(code.ErrBind, "invalid query payload: %v", err)
	}
	return nil
}

// BindURI 绑定 URI 参数，返回统一封装的错误。
func (h *BaseHandler) BindURI(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindUri(obj); err != nil {
		return perrors.WithCode(code.ErrBind, "invalid uri payload: %v", err)
	}
	return nil
}

// GetPathParam 获取路径参数。
func (h *BaseHandler) GetPathParam(c *gin.Context, key string) string {
	return c.Param(key)
}

// GetQueryParam 获取查询参数。
func (h *BaseHandler) GetQueryParam(c *gin.Context, key string) string {
	return c.Query(key)
}

// GetQueryParamInt 获取整数查询参数。
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

// GetUserID 从上下文获取当前用户ID。
// 使用常量 ContextKeyUserID 避免字符串拼写错误，增强安全性
func (h *BaseHandler) GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(authn.ContextKeyUserID)
	if !exists || userID == nil {
		return "", false
	}

	switch v := userID.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	case uint32:
		return strconv.FormatUint(uint64(v), 10), true
	case uint:
		return strconv.FormatUint(uint64(v), 10), true
	case int64:
		if v < 0 {
			return "", false
		}
		return strconv.FormatInt(v, 10), true
	case int32:
		if v < 0 {
			return "", false
		}
		return strconv.FormatInt(int64(v), 10), true
	case int:
		if v < 0 {
			return "", false
		}
		return strconv.FormatInt(int64(v), 10), true
	default:
		return fmt.Sprintf("%v", v), true
	}
}
