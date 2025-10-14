package apiserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/container"
	authnhttp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/interface/restful"
	authzhttp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/interface/restful"
	userhttp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/interface/restful"
)

// Router 集中的路由管理器
type Router struct {
	container *container.Container
	auth      *Auth
}

// NewRouter 创建路由管理器
func NewRouter(c *container.Container) *Router {
	return &Router{
		container: c,
		auth:      NewAuth(c), // 初始化认证配置
	}
}

// RegisterRoutes 注册所有路由
func (r *Router) RegisterRoutes(engine *gin.Engine) {
	if engine == nil {
		return
	}

	r.registerBaseRoutes(engine)

	if r.container == nil {
		fmt.Printf("⚠️  container not initialized, skipped module route registration\n")
		return
	}

	autoAuth := r.auth.CreateAuthMiddleware("auto")
	jwtStrategy := r.auth.NewJWTAuth()

	userhttp.Provide(userhttp.Dependencies{
		Module:         r.container.UserModule,
		AuthMiddleware: autoAuth,
	})
	authnhttp.Provide(authnhttp.Dependencies{
		JWTStrategy: &jwtStrategy,
	})
	authzhttp.Provide(authzhttp.Dependencies{})

	userhttp.Register(engine)
	authnhttp.Register(engine)
	authzhttp.Register(engine)

	r.registerAdminRoutes(engine, autoAuth)

	fmt.Printf("🔗 Registered routes for: base, user, authn, authz\n")
}

func (r *Router) registerBaseRoutes(engine *gin.Engine) {
	engine.GET("/health", r.healthCheck)
	engine.GET("/ping", r.ping)

	publicAPI := engine.Group("/api/v1/public")
	{
		publicAPI.GET("/info", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service":     "web-framework",
				"version":     "1.0.0",
				"description": "Web框架系统",
			})
		})
	}
	// admin.Use(r.requireAdminRole()) // 需要实现管理员权限检查中间件
}

func (r *Router) registerAdminRoutes(engine *gin.Engine, authMiddleware gin.HandlerFunc) {
	if engine == nil {
		return
	}

	apiV1 := engine.Group("/api/v1")
	if authMiddleware != nil {
		apiV1.Use(authMiddleware)
	}

	admin := apiV1.Group("/admin")
	{
		admin.GET("/users", r.placeholder)      // 管理员获取所有用户
		admin.GET("/statistics", r.placeholder) // 系统统计信息
		admin.GET("/logs", r.placeholder)       // 系统日志
	}
}

// placeholder 占位符处理器（用于未实现的功能）
func (r *Router) placeholder(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"code":    501,
		"message": "功能尚未实现",
		"path":    c.Request.URL.Path,
		"method":  c.Request.Method,
	})
}

// healthCheck 健康检查处理函数
func (r *Router) healthCheck(c *gin.Context) {
	response := gin.H{
		"status":       "healthy",
		"version":      "1.0.0",
		"discovery":    "auto",
		"architecture": "hexagonal",
		"router":       "centralized",
		"auth":         "enabled", // 新增认证状态
		"components": gin.H{
			"domain":      "user",
			"ports":       "storage",
			"adapters":    "mysql, http",
			"application": "user_service",
		},
		"jwt_config": gin.H{
			"realm":       viper.GetString("jwt.realm"),
			"timeout":     viper.GetDuration("jwt.timeout").String(),
			"max_refresh": viper.GetDuration("jwt.max-refresh").String(),
			"key_loaded":  viper.GetString("jwt.key") != "", // 不显示实际密钥，只显示是否加载
		},
	}

	c.JSON(200, response)
}

// ping 简单的连通性测试
func (r *Router) ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
		"status":  "ok",
		"router":  "centralized",
		"auth":    "enabled",
	})
}
