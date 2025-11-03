package restful

import (
	"github.com/gin-gonic/gin"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/container/assembler"
)

// Dependencies bundles the runtime collaborators required by the UC HTTP adapters.
type Dependencies struct {
	Module         *assembler.UserModule
	AuthMiddleware gin.HandlerFunc
}

var deps Dependencies

// Provide wires the dependencies to be used when Register is invoked.
func Provide(d Dependencies) {
	deps = d
}

// Register exposes the UC module REST endpoints on the supplied engine.
func Register(engine *gin.Engine) {
	if engine == nil || deps.Module == nil {
		return
	}

	api := engine.Group("/api/v1")

	// 公开端点（无需认证）
	registerPublicUserRoutes(api, deps.Module)

	// 受保护的端点（需要认证）
	if deps.AuthMiddleware != nil {
		api.Use(deps.AuthMiddleware)
	}

	registerProtectedUserRoutes(api, deps.Module)
	registerChildRoutes(api, deps.Module)
	registerGuardianshipRoutes(api, deps.Module)
}

// registerPublicUserRoutes 注册公开的用户端点（无需认证）
func registerPublicUserRoutes(api *gin.RouterGroup, module *assembler.UserModule) {
	if module.UserHandler == nil {
		return
	}

	users := api.Group("/users")
	{
		// 用户注册是公开端点，任何人都可以访问
		users.POST("", module.UserHandler.CreateUser)
	}
}

// registerProtectedUserRoutes 注册受保护的用户端点（需要认证）
func registerProtectedUserRoutes(api *gin.RouterGroup, module *assembler.UserModule) {
	if module.UserHandler == nil {
		return
	}

	users := api.Group("/users")
	{
		// 获取当前用户资料（需要认证）
		users.GET("/profile", module.UserHandler.GetUserProfile)
		// 获取指定用户信息（需要认证）
		users.GET("/:userId", module.UserHandler.GetUser)
		// 更新用户信息（需要认证）
		users.PATCH("/:userId", module.UserHandler.PatchUser)
	}
}

func registerChildRoutes(api *gin.RouterGroup, module *assembler.UserModule) {
	if module.ChildHandler == nil {
		return
	}

	me := api.Group("/me")
	{
		me.GET("/children", module.ChildHandler.ListMyChildren)
	}

	api.POST("/children/register", module.ChildHandler.RegisterChild)
	api.GET("/children/search", module.ChildHandler.SearchChildren)

	children := api.Group("/children")
	{
		children.POST("", module.ChildHandler.CreateChild)
		children.GET("/:childId", module.ChildHandler.GetChild)
		children.PATCH("/:childId", module.ChildHandler.PatchChild)
	}
}

func registerGuardianshipRoutes(api *gin.RouterGroup, module *assembler.UserModule) {
	if module.GuardianshipHandler == nil {
		return
	}

	api.POST("/guardians/grant", module.GuardianshipHandler.Grant)
	api.POST("/guardians/revoke", module.GuardianshipHandler.Revoke)
	api.GET("/guardians", module.GuardianshipHandler.List)
}
