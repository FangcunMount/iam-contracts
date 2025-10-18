// Package dominguard 使用示例
package dominguard_test

import (
	"context"
	"fmt"
	"time"

	"github.com/fangcun-mount/iam-contracts/pkg/dominguard"
	"github.com/gin-gonic/gin"
)

// 示例：基本的权限检查
func ExampleBasicPermissionCheck() {
	// 1. 创建 DomainGuard 实例
	guard, err := dominguard.NewDomainGuard(dominguard.Config{
		Enforcer: nil, // 在实际使用中，传入 Casbin Enforcer
		CacheTTL: 5 * time.Minute,
	})
	if err != nil {
		panic(err)
	}

	// 2. 检查用户权限
	ctx := context.Background()
	userID := "user123"
	tenantID := "tenant1"
	resource := "order"
	action := "read"

	allowed, err := guard.CheckPermission(ctx, userID, tenantID, resource, action)
	if err != nil {
		fmt.Printf("权限检查失败: %v\n", err)
		return
	}

	if allowed {
		fmt.Println("用户有权限执行该操作")
	} else {
		fmt.Println("用户没有权限执行该操作")
	}
}

// 示例：批量权限检查
func ExampleBatchPermissionCheck() {
	guard, _ := dominguard.NewDomainGuard(dominguard.Config{
		Enforcer: nil,
	})

	ctx := context.Background()
	userID := "user123"
	tenantID := "tenant1"

	// 批量检查多个权限
	permissions := []dominguard.Permission{
		{Resource: "order", Action: "read"},
		{Resource: "order", Action: "write"},
		{Resource: "product", Action: "read"},
	}

	results, err := guard.BatchCheckPermissions(ctx, userID, tenantID, permissions)
	if err != nil {
		fmt.Printf("批量权限检查失败: %v\n", err)
		return
	}

	for key, allowed := range results {
		fmt.Printf("%s: %v\n", key, allowed)
	}
}

// 示例：Gin 中间件使用
func ExampleGinMiddleware() {
	// 创建 DomainGuard
	guard, _ := dominguard.NewDomainGuard(dominguard.Config{
		Enforcer: nil,
	})

	// 注册资源显示名称
	guard.RegisterResource("order", "订单")
	guard.RegisterResource("product", "产品")

	// 创建认证中间件
	authMiddleware := dominguard.NewAuthMiddleware(dominguard.MiddlewareConfig{
		Guard: guard,
		GetUserID: func(c *gin.Context) string {
			// 从 JWT 或 Session 中提取用户ID
			return c.GetString("user_id")
		},
		GetTenantID: func(c *gin.Context) string {
			// 从请求头或上下文中提取租户ID
			return c.GetHeader("X-Tenant-ID")
		},
		SkipPaths: []string{
			"/health",
			"/login",
		},
	})

	// 设置路由
	router := gin.Default()

	// 需要 order:read 权限
	router.GET("/orders", authMiddleware.RequirePermission("order", "read"), func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "订单列表"})
	})

	// 需要 order:write 权限
	router.POST("/orders", authMiddleware.RequirePermission("order", "write"), func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "创建订单成功"})
	})

	// 需要任意一个权限：order:read 或 order:write
	router.GET("/orders/:id", authMiddleware.RequireAnyPermission([]dominguard.Permission{
		{Resource: "order", Action: "read"},
		{Resource: "order", Action: "write"},
	}), func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "订单详情"})
	})

	// 需要所有权限：order:read 和 product:read
	router.GET("/orders/:id/details", authMiddleware.RequireAllPermissions([]dominguard.Permission{
		{Resource: "order", Action: "read"},
		{Resource: "product", Action: "read"},
	}), func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "订单详细信息"})
	})

	router.Run(":8080")
}

// 示例：服务权限检查
func ExampleServicePermissionCheck() {
	guard, _ := dominguard.NewDomainGuard(dominguard.Config{
		Enforcer: nil,
	})

	ctx := context.Background()
	serviceID := "order-service"
	tenantID := "tenant1"
	resource := "inventory"
	action := "update"

	// 检查服务权限
	allowed, err := guard.CheckServicePermission(ctx, serviceID, tenantID, resource, action)
	if err != nil {
		fmt.Printf("服务权限检查失败: %v\n", err)
		return
	}

	if allowed {
		fmt.Println("服务有权限执行该操作")
		// 执行库存更新
	} else {
		fmt.Println("服务没有权限执行该操作")
	}
}
