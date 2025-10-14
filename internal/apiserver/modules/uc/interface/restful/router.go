package restful

import (
	"github.com/gin-gonic/gin"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/container/assembler"
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
	if deps.AuthMiddleware != nil {
		api.Use(deps.AuthMiddleware)
	}

	registerUserRoutes(api, deps.Module)
	registerChildRoutes(api, deps.Module)
	registerGuardianshipRoutes(api, deps.Module)
}

func registerUserRoutes(api *gin.RouterGroup, module *assembler.UserModule) {
	if module.UserHandler == nil {
		return
	}

	users := api.Group("/users")
	{
		users.POST("", module.UserHandler.CreateUser)
		users.GET("/profile", module.UserHandler.GetUserProfile)
		users.GET("/:userId", module.UserHandler.GetUser)
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

	api.POST("/children:register", module.ChildHandler.RegisterChild)
	api.GET("/children:search", module.ChildHandler.SearchChildren)

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

	api.POST("/guardians:grant", module.GuardianshipHandler.Grant)
	api.POST("/guardians:revoke", module.GuardianshipHandler.Revoke)
	api.GET("/guardians", module.GuardianshipHandler.List)
}
