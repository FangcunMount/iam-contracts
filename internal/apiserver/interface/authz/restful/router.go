package restful

import (
	"net/http"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/interface/authz/restful/handler"
	"github.com/gin-gonic/gin"
)

// Dependencies 授权模块的依赖
type Dependencies struct {
	RoleHandler       *handler.RoleHandler
	AssignmentHandler *handler.AssignmentHandler
	PolicyHandler     *handler.PolicyHandler
	ResourceHandler   *handler.ResourceHandler
}

var deps Dependencies

// Provide 存储依赖供 Register 使用
func Provide(d Dependencies) {
	deps = d
}

// Register 注册授权模块的所有路由
func Register(engine *gin.Engine) {
	if engine == nil {
		return
	}

	authzGroup := engine.Group("/api/v1/authz")
	{
		// 健康检查
		authzGroup.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
				"module": "authz",
			})
		})

		// 如果依赖未初始化，只注册健康检查
		if deps.RoleHandler == nil {
			return
		}

		// ============ 角色管理 ============
		roles := authzGroup.Group("/roles")
		{
			roles.POST("", deps.RoleHandler.CreateRole)                                 // 创建角色
			roles.PUT("/:id", deps.RoleHandler.UpdateRole)                              // 更新角色
			roles.DELETE("/:id", deps.RoleHandler.DeleteRole)                           // 删除角色
			roles.GET("/:id", deps.RoleHandler.GetRole)                                 // 获取角色详情
			roles.GET("", deps.RoleHandler.ListRoles)                                   // 列出角色
			roles.GET("/:id/assignments", deps.AssignmentHandler.ListAssignmentsByRole) // 列出角色的分配记录
			roles.GET("/:id/policies", deps.PolicyHandler.GetPoliciesByRole)            // 获取角色的策略列表
		}

		// ============ 角色分配 ============
		assignments := authzGroup.Group("/assignments")
		{
			assignments.POST("/grant", deps.AssignmentHandler.GrantRole)                 // 授予角色
			assignments.POST("/revoke", deps.AssignmentHandler.RevokeRole)               // 撤销角色
			assignments.DELETE("/:id", deps.AssignmentHandler.RevokeRoleByID)            // 根据ID撤销
			assignments.GET("/subject", deps.AssignmentHandler.ListAssignmentsBySubject) // 列出主体的分配
		}

		// ============ 策略管理 ============
		policies := authzGroup.Group("/policies")
		{
			policies.POST("", deps.PolicyHandler.AddPolicyRule)            // 添加策略规则
			policies.DELETE("", deps.PolicyHandler.RemovePolicyRule)       // 移除策略规则
			policies.GET("/version", deps.PolicyHandler.GetCurrentVersion) // 获取当前策略版本
		}

		// ============ 资源管理 ============
		resources := authzGroup.Group("/resources")
		{
			resources.POST("", deps.ResourceHandler.CreateResource)                 // 创建资源
			resources.PUT("/:id", deps.ResourceHandler.UpdateResource)              // 更新资源
			resources.DELETE("/:id", deps.ResourceHandler.DeleteResource)           // 删除资源
			resources.GET("/:id", deps.ResourceHandler.GetResource)                 // 获取资源详情
			resources.GET("/key/:key", deps.ResourceHandler.GetResourceByKey)       // 根据键获取资源
			resources.GET("", deps.ResourceHandler.ListResources)                   // 列出资源
			resources.POST("/validate-action", deps.ResourceHandler.ValidateAction) // 验证资源动作
		}
	}
}
