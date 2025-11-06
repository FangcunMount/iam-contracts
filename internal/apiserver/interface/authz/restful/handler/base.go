// Package handler REST API 处理器基础
package handler

import (
	"net/http"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/interface/authz/restful/dto"
	"github.com/gin-gonic/gin"
)

// getTenantID 从上下文中获取租户ID
// 实际项目中应该从 JWT token 或 header 中提取
func getTenantID(c *gin.Context) string {
	// TODO: 从认证中间件设置的上下文中获取
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = c.GetString("tenant_id")
	}
	if tenantID == "" {
		tenantID = "default" // 默认租户（开发环境）
	}
	return tenantID
}

// getUserID 从上下文中获取用户ID
// 实际项目中应该从 JWT token 中提取
func getUserID(c *gin.Context) string {
	// TODO: 从认证中间件设置的上下文中获取
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = c.GetString("user_id")
	}
	if userID == "" {
		userID = "system" // 默认用户（开发环境）
	}
	return userID
}

// handleError 统一错误处理
func handleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// 解析错误码
	coder := errors.ParseCoder(err)

	c.JSON(coder.HTTPStatus(), dto.ErrorResponse{
		Code:    coder.Code(),
		Message: coder.String(),
		Error:   err.Error(),
	})
}

// success 成功响应
func success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, dto.NewResponse(data))
}

// successList 分页列表成功响应
func successList(c *gin.Context, data interface{}, total int64, offset, limit int) {
	c.JSON(http.StatusOK, dto.NewListResponse(data, total, offset, limit))
}

// successNoContent 无内容成功响应
func successNoContent(c *gin.Context) {
	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
	})
}
