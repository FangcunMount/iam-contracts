// Package handler IDP 模块 REST API 处理器基础
package handler

import (
	"net/http"

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
		return perrors.WrapC(err, code.ErrBind, "failed to bind JSON request")
	}
	return nil
}

// BindQuery 绑定查询参数，返回统一封装的错误
func (h *BaseHandler) BindQuery(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindQuery(obj); err != nil {
		return perrors.WrapC(err, code.ErrBind, "failed to bind query parameters")
	}
	return nil
}

// BindURI 绑定 URI 参数，返回统一封装的错误
func (h *BaseHandler) BindURI(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindUri(obj); err != nil {
		return perrors.WrapC(err, code.ErrBind, "failed to bind URI parameters")
	}
	return nil
}

// GetUserID 从上下文中获取用户 ID（从 JWT token 中提取）
func (h *BaseHandler) GetUserID(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return ""
	}
	if uid, ok := userID.(string); ok {
		return uid
	}
	return ""
}

// GetTenantID 从上下文中获取租户 ID（多租户场景）
func (h *BaseHandler) GetTenantID(c *gin.Context) string {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return "default" // 默认租户
	}
	if tid, ok := tenantID.(string); ok {
		return tid
	}
	return "default"
}
