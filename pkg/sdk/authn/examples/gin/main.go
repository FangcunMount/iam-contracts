package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	authnsdk "github.com/FangcunMount/iam-contracts/pkg/sdk/authn"
	"github.com/gin-gonic/gin"
)

// 全局验证器实例
var verifier *authnsdk.Verifier

// 初始化验证器
func initVerifier() error {
	cfg := authnsdk.Config{
		JWKSURL:         "https://iam.example.com/.well-known/jwks.json",
		AllowedAudience: []string{"my-app", "admin-panel"},
		AllowedIssuer:   "https://iam.example.com",
	}

	var err error
	verifier, err = authnsdk.NewVerifier(cfg, nil)
	if err != nil {
		return fmt.Errorf("初始化验证器失败: %w", err)
	}

	log.Println("✅ 验证器初始化成功")
	return nil
}

// AuthMiddleware JWT 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 提取 Authorization header
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			return
		}

		// 2. 检查 Bearer 前缀
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			return
		}

		// 3. 提取 token
		token := strings.TrimPrefix(auth, "Bearer ")

		// 4. 验证 token
		resp, err := verifier.Verify(c.Request.Context(), token, nil)
		if err != nil {
			log.Printf("Token 验证失败: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			return
		}

		// 5. 将用户信息存入上下文
		c.Set("user_id", resp.Claims.UserId)
		c.Set("tenant_id", resp.Claims.TenantId)
		c.Set("account_id", resp.Claims.AccountId)
		c.Set("token_id", resp.Claims.TokenId)

		// 6. 继续处理请求
		c.Next()
	}
}

// OptionalAuthMiddleware 可选的认证中间件
// 如果有 token 则验证，没有 token 也允许通过
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			// 没有 token，继续处理（匿名用户）
			c.Next()
			return
		}

		if !strings.HasPrefix(auth, "Bearer ") {
			c.Next()
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		resp, err := verifier.Verify(c.Request.Context(), token, nil)
		if err != nil {
			// token 无效，继续处理（匿名用户）
			log.Printf("Token 验证失败（可选认证）: %v", err)
			c.Next()
			return
		}

		// token 有效，存储用户信息
		c.Set("user_id", resp.Claims.UserId)
		c.Set("tenant_id", resp.Claims.TenantId)
		c.Set("authenticated", true)

		c.Next()
	}
}

// TenantMiddleware 租户验证中间件
// 必须在 AuthMiddleware 之后使用
func TenantMiddleware(allowedTenants []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "tenant information missing",
			})
			return
		}

		tenantIDStr := tenantID.(string)

		// 检查租户是否在允许列表中
		allowed := false
		for _, t := range allowedTenants {
			if t == tenantIDStr {
				allowed = true
				break
			}
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "tenant not allowed",
			})
			return
		}

		c.Next()
	}
}

// getUserID 从上下文获取用户 ID
func getUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	return userID.(string), true
}

// getTenantID 从上下文获取租户 ID
func getTenantID(c *gin.Context) (string, bool) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return "", false
	}
	return tenantID.(string), true
}

// 路由处理函数

// 公开端点（无需认证）
func publicHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "这是公开端点，无需认证",
	})
}

// 受保护端点（需要认证）
func protectedHandler(c *gin.Context) {
	userID, _ := getUserID(c)
	tenantID, _ := getTenantID(c)

	c.JSON(http.StatusOK, gin.H{
		"message":   "这是受保护端点，需要认证",
		"user_id":   userID,
		"tenant_id": tenantID,
	})
}

// 用户信息端点
func userInfoHandler(c *gin.Context) {
	userID, _ := getUserID(c)
	tenantID, _ := getTenantID(c)
	accountID, _ := c.Get("account_id")
	tokenID, _ := c.Get("token_id")

	c.JSON(http.StatusOK, gin.H{
		"user_id":    userID,
		"tenant_id":  tenantID,
		"account_id": accountID,
		"token_id":   tokenID,
	})
}

// 可选认证端点
func optionalAuthHandler(c *gin.Context) {
	authenticated, exists := c.Get("authenticated")
	if exists && authenticated.(bool) {
		userID, _ := getUserID(c)
		c.JSON(http.StatusOK, gin.H{
			"message": "已认证用户",
			"user_id": userID,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message": "匿名用户",
		})
	}
}

// 租户专属端点
func tenantOnlyHandler(c *gin.Context) {
	userID, _ := getUserID(c)
	tenantID, _ := getTenantID(c)

	c.JSON(http.StatusOK, gin.H{
		"message":   "这是租户专属端点",
		"user_id":   userID,
		"tenant_id": tenantID,
	})
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	// 公开路由
	r.GET("/public", publicHandler)

	// 可选认证路由
	r.GET("/optional", OptionalAuthMiddleware(), optionalAuthHandler)

	// 需要认证的路由
	authenticated := r.Group("/api")
	authenticated.Use(AuthMiddleware())
	{
		authenticated.GET("/protected", protectedHandler)
		authenticated.GET("/user/info", userInfoHandler)

		// 需要特定租户权限的路由
		tenantRoutes := authenticated.Group("/tenant")
		tenantRoutes.Use(TenantMiddleware([]string{"tenant-123", "tenant-456"}))
		{
			tenantRoutes.GET("/dashboard", tenantOnlyHandler)
			tenantRoutes.GET("/settings", tenantOnlyHandler)
		}
	}

	return r
}

func main() {
	// 1. 初始化验证器
	if err := initVerifier(); err != nil {
		log.Fatal(err)
	}

	// 2. 设置路由
	router := setupRouter()

	// 3. 启动服务器
	log.Println("🚀 服务器启动在 http://localhost:8080")
	log.Println("测试端点:")
	log.Println("  - GET /public        (无需认证)")
	log.Println("  - GET /optional      (可选认证)")
	log.Println("  - GET /api/protected (需要认证)")
	log.Println("  - GET /api/user/info (需要认证)")
	log.Println("  - GET /api/tenant/dashboard (需要认证 + 租户权限)")

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
