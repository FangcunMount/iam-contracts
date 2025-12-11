// Package handler REST API 处理器基础
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/interface/authz/restful/dto"
	"github.com/FangcunMount/iam-contracts/pkg/core"
)

// BaseHandler 继承公共的 BaseHandler，并添加 authz 模块特定的方法
type BaseHandler struct {
	*core.BaseHandler
}

// NewBaseHandler 创建基础 Handler
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{
		BaseHandler: core.NewBaseHandler(),
	}
}

// getTenantID 从上下文中获取租户ID
// 实际项目中应该从 JWT token 或 header 中提取
func getTenantID(c *gin.Context) string {
	// 使用公共方法
	return core.NewBaseHandler().GetTenantID(c)
}

// getUserID 从上下文中获取用户ID
// 实际项目中应该从 JWT token 中提取
func getUserID(c *gin.Context) string {
	userID, _ := core.NewBaseHandler().GetUserID(c)
	if userID == "" {
		return "system" // 默认用户（开发环境）
	}
	return userID
}

// handleError 统一错误处理 (authz 模块特定的错误格式)
func handleError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	// 委托给 BaseHandler 的 Error 方法
	// 但是使用 authz 特定的错误响应格式
	core.NewBaseHandler().Error(c, err)
}

// success 成功响应 (authz 模块特定的响应格式)
func success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, dto.NewResponse(data))
}

// successList 分页列表成功响应 (authz 模块特定的响应格式)
func successList(c *gin.Context, data interface{}, total int64, offset, limit int) {
	c.JSON(http.StatusOK, dto.NewListResponse(data, total, offset, limit))
}

// successNoContent 无内容成功响应 (authz 模块特定的响应格式)
func successNoContent(c *gin.Context) {
	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
	})
}
