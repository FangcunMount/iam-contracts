package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	// 这里可以注入用户相关的服务
}

// NewUserHandler 创建用户处理器
func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// GetUserProfile 获取用户资料
func (h *UserHandler) GetUserProfile(c *gin.Context) {
	// 从JWT token中获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权访问",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"user_id": userID,
			"profile": gin.H{
				"id":       userID,
				"username": "demo_user",
				"email":    "demo@example.com",
				"status":   "active",
			},
		},
		"message": "获取用户资料成功",
	})
}
