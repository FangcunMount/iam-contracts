package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/pkg/core"
)

// BaseHandler 提供统一的响应与绑定能力。
type BaseHandler struct{}

// NewBaseHandler 构造基础处理器。
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{}
}

// Success 输出标准成功响应。
func (h *BaseHandler) Success(c *gin.Context, data interface{}) {
	core.WriteResponse(c, nil, data)
}

// Created 输出 201 响应。
func (h *BaseHandler) Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// NoContent 输出 204 响应。
func (h *BaseHandler) NoContent(c *gin.Context) {
	c.AbortWithStatus(http.StatusNoContent)
}

// Error 输出错误响应。
func (h *BaseHandler) Error(c *gin.Context, err error) {
	if err == nil {
		h.Success(c, nil)
		return
	}
	core.WriteResponse(c, err, nil)
}

// ErrorWithCode 使用业务错误码写出错误。
func (h *BaseHandler) ErrorWithCode(c *gin.Context, code int, format string, args ...interface{}) {
	h.Error(c, perrors.WithCode(code, format, args...))
}

// BindJSON 绑定 JSON 请求体。
func (h *BaseHandler) BindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return perrors.WithCode(code.ErrBind, "invalid JSON payload: %v", err)
	}
	return nil
}

// ParseUint parses a string based uint64 with validation.
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
