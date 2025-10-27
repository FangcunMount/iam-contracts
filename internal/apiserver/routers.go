package apiserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/container"
	authnhttp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/interface/restful"
	authzhttp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/interface/restful"
	userhttp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/interface/restful"
	authnMiddleware "github.com/FangcunMount/iam-contracts/internal/pkg/middleware/authn"
	"github.com/FangcunMount/iam-contracts/pkg/log"
)

// Router 集中的路由管理器
type Router struct {
	container *container.Container
}

// NewRouter 创建路由管理器
func NewRouter(c *container.Container) *Router {
	return &Router{
		container: c,
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

	// 创建新的认证中间件
	var authMiddleware *authnMiddleware.JWTAuthMiddleware
	if r.container.AuthnModule != nil && r.container.AuthnModule.TokenService != nil {
		authMiddleware = authnMiddleware.NewJWTAuthMiddleware(
			r.container.AuthnModule.TokenService,
		)
	} else {
		log.Warn("Authn module unavailable; routes will be exposed without JWT middleware")
	}

	// User 模块使用新中间件
	userhttp.Provide(userhttp.Dependencies{
		Module: r.container.UserModule,
		AuthMiddleware: func() gin.HandlerFunc {
			if authMiddleware != nil {
				return authMiddleware.AuthRequired()
			}
			// 如果认证中间件未初始化，返回一个空的中间件
			return func(c *gin.Context) {
				c.Next()
			}
		}(),
	})

	// Authn 模块（公开端点）
	if r.container.AuthnModule != nil {
		authnhttp.Provide(authnhttp.Dependencies{
			AuthHandler:    r.container.AuthnModule.AuthHandler,
			AccountHandler: r.container.AuthnModule.AccountHandler,
			JWKSHandler:    r.container.AuthnModule.JWKSHandler,
		})
	} else {
		authnhttp.Provide(authnhttp.Dependencies{})
	}

	// Authz 模块（授权管理）
	if r.container.AuthzModule != nil {
		authzhttp.Provide(authzhttp.Dependencies{
			RoleHandler:       r.container.AuthzModule.RoleHandler,
			AssignmentHandler: r.container.AuthzModule.AssignmentHandler,
			PolicyHandler:     r.container.AuthzModule.PolicyHandler,
			ResourceHandler:   r.container.AuthzModule.ResourceHandler,
		})
	} else {
		authzhttp.Provide(authzhttp.Dependencies{})
	}

	userhttp.Register(engine)
	if r.container.AuthnModule != nil {
		authnhttp.Register(engine)
	} else {
		log.Warn("Authn endpoints disabled because module failed to initialize")
	}
	if r.container.AuthzModule != nil {
		authzhttp.Register(engine)
	} else {
		log.Warn("Authz endpoints disabled because module failed to initialize")
	}

	r.registerAdminRoutes(engine, authMiddleware)

	fmt.Printf("🔗 Registered routes for: base, user, authn, authz\n")
}

func (r *Router) registerBaseRoutes(engine *gin.Engine) {
	engine.GET("/health", r.healthCheck)
	engine.GET("/ping", r.ping)

	publicAPI := engine.Group("/api/v1/public")
	{
		publicAPI.GET("/info", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service":     "iam-apiserver",
				"version":     "1.0.0",
				"description": "IAM Contracts API Server",
			})
		})
	}
	// admin.Use(r.requireAdminRole()) // 需要实现管理员权限检查中间件
}

func (r *Router) registerAdminRoutes(engine *gin.Engine, authMiddleware *authnMiddleware.JWTAuthMiddleware) {
	if engine == nil {
		return
	}

	apiV1 := engine.Group("/api/v1")
	if authMiddleware != nil {
		apiV1.Use(authMiddleware.AuthRequired())
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
		"auth_system": gin.H{
			"type":    "jwt",
			"enabled": true,
			"module":  "authn (DDD 4-layer)",
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
