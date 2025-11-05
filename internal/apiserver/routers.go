package apiserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/container"
	authnhttp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/interface/restful"
	authzhttp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/interface/restful"
	idphttp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/interface/restful"
	userhttp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/interface/restful"
	authnMiddleware "github.com/FangcunMount/iam-contracts/internal/pkg/middleware/authn"
)

// Router 集中的路由管理器
type Router struct {
	container *container.Container
	engine    *gin.Engine // 保存 engine 引用用于调试
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

	r.engine = engine // 保存引用用于调试

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
		authnhttp.Register(engine)
		log.Info("✅ Authn module routes registered")
	} else {
		log.Warn("⚠️  Authn module not initialized, routes not registered")
	}

	// Authz 模块（授权管理）
	if r.container.AuthzModule != nil {
		authzhttp.Provide(authzhttp.Dependencies{
			RoleHandler:       r.container.AuthzModule.RoleHandler,
			AssignmentHandler: r.container.AuthzModule.AssignmentHandler,
			PolicyHandler:     r.container.AuthzModule.PolicyHandler,
			ResourceHandler:   r.container.AuthzModule.ResourceHandler,
		})
		authzhttp.Register(engine)
		log.Info("✅ Authz module routes registered")
	} else {
		log.Warn("⚠️  Authz module not initialized, routes not registered")
	}

	// IDP 模块（身份提供者）
	if r.container.IDPModule != nil {
		idphttp.Provide(idphttp.Dependencies{
			WechatAppHandler: r.container.IDPModule.WechatAppHandler,
			// WechatAuthHandler 已移除 - 认证由 authn 模块统一提供
		})
		idphttp.Register(engine)
		log.Info("✅ IDP module routes registered")
	} else {
		log.Warn("⚠️  IDP module not initialized, routes not registered")
	}

	// User 模块路由始终注册
	userhttp.Register(engine)
	log.Info("✅ User module routes registered")

	r.registerAdminRoutes(engine, authMiddleware)

	log.Info("🔗 All routes registration completed")
}

func (r *Router) registerBaseRoutes(engine *gin.Engine) {
	engine.GET("/health", r.healthCheck)
	engine.GET("/ping", r.ping)
	engine.GET("/debug/routes", r.debugRoutes)   // 调试端点：列出所有注册的路由
	engine.GET("/debug/modules", r.debugModules) // 调试端点：查看模块状态

	// Swagger UI 路由（默认在开发环境可用）
	// 生产环境建议通过配置控制是否启用
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("/swagger/doc.json"),
	))

	publicAPI := engine.Group("/api/v1/public")
	{
		publicAPI.GET("/info", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service":     "iam-apiserver",
				"version":     "1.0.0",
				"description": "IAM Contracts API Server",
				"swagger":     "/swagger/index.html",
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

// debugRoutes 调试端点：列出所有注册的路由
func (r *Router) debugRoutes(c *gin.Context) {
	if r.engine == nil {
		c.JSON(500, gin.H{"error": "engine not initialized"})
		return
	}

	routes := r.engine.Routes()
	routeList := make([]gin.H, 0, len(routes))
	for _, route := range routes {
		routeList = append(routeList, gin.H{
			"method": route.Method,
			"path":   route.Path,
		})
	}
	c.JSON(200, gin.H{
		"total":  len(routes),
		"routes": routeList,
	})
}

// debugModules 调试端点：查看模块初始化状态
func (r *Router) debugModules(c *gin.Context) {
	response := gin.H{
		"container_initialized": r.container != nil,
	}

	if r.container != nil {
		response["modules"] = gin.H{
			"authn": r.container.AuthnModule != nil,
			"authz": r.container.AuthzModule != nil,
			"user":  r.container.UserModule != nil,
			"idp":   r.container.IDPModule != nil,
		}
		response["container_status"] = "initialized"
	} else {
		response["container_status"] = "not_initialized"
	}

	c.JSON(200, response)
}
