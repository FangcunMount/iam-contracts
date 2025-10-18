# DomainGuard - PEP SDK

DomainGuard 是一个轻量级的权限检查客户端（PEP - Policy Enforcement Point），为业务服务提供简单易用的权限检查 API。

## 特性

- ✅ **简单易用**: 提供直观的 API，快速集成到业务服务
- ✅ **高性能**: 内置版本缓存，减少不必要的查询
- ✅ **实时更新**: 通过 Redis 订阅策略变更，自动刷新缓存
- ✅ **批量检查**: 支持批量权限检查，提升性能
- ✅ **Gin 中间件**: 开箱即用的 Gin 中间件，快速保护路由
- ✅ **灵活配置**: 支持自定义错误处理、路径跳过等

## 快速开始

### 1. 安装

```bash
go get github.com/fangcun-mount/iam-contracts/pkg/dominguard
```

### 2. 创建 DomainGuard 实例

```go
import (
    "github.com/fangcun-mount/iam-contracts/pkg/dominguard"
    casbin "github.com/casbin/casbin/v2"
)

// 创建 Casbin Enforcer
enforcer, _ := casbin.NewEnforcer("model.conf", "policy.csv")

// 创建 DomainGuard
guard, err := dominguard.NewDomainGuard(dominguard.Config{
    Enforcer:     enforcer,
    RedisClient:  redisClient, // 可选，用于监听策略变更
    CacheTTL:     5 * time.Minute,
    VersionTopic: "authz:policy_changed",
})
```

### 3. 基本权限检查

```go
allowed, err := guard.CheckPermission(
    ctx,
    "user123",    // 用户ID
    "tenant1",    // 租户ID
    "order",      // 资源
    "read",       // 操作
)

if allowed {
    // 执行业务逻辑
} else {
    // 拒绝访问
}
```

### 4. 批量权限检查

```go
permissions := []dominguard.Permission{
    {Resource: "order", Action: "read"},
    {Resource: "order", Action: "write"},
    {Resource: "product", Action: "read"},
}

results, err := guard.BatchCheckPermissions(ctx, userID, tenantID, permissions)
// results: {"order:read": true, "order:write": false, "product:read": true}
```

### 5. 服务权限检查

```go
// 检查服务是否有权限
allowed, err := guard.CheckServicePermission(
    ctx,
    "order-service", // 服务ID
    "tenant1",
    "inventory",
    "update",
)
```

## Gin 中间件

### 基本使用

```go
// 创建中间件
authMiddleware := dominguard.NewAuthMiddleware(dominguard.MiddlewareConfig{
    Guard: guard,
    GetUserID: func(c *gin.Context) string {
        return c.GetString("user_id") // 从 JWT 或 Session 中提取
    },
    GetTenantID: func(c *gin.Context) string {
        return c.GetHeader("X-Tenant-ID")
    },
    SkipPaths: []string{"/health", "/login"},
})

// 保护路由
router.GET("/orders", 
    authMiddleware.RequirePermission("order", "read"),
    orderHandler,
)
```

### 需要任意一个权限

```go
router.GET("/orders/:id", 
    authMiddleware.RequireAnyPermission([]dominguard.Permission{
        {Resource: "order", Action: "read"},
        {Resource: "order", Action: "write"},
    }),
    orderDetailHandler,
)
```

### 需要所有权限

```go
router.POST("/orders/:id/ship", 
    authMiddleware.RequireAllPermissions([]dominguard.Permission{
        {Resource: "order", Action: "write"},
        {Resource: "inventory", Action: "update"},
    }),
    shipOrderHandler,
)
```

## 资源显示名称

```go
// 注册资源的友好显示名称
guard.RegisterResource("order", "订单")
guard.RegisterResource("product", "产品")
guard.RegisterResource("inventory", "库存")

// 在错误提示中会显示友好名称
// "没有权限访问 订单" 而不是 "没有权限访问 order"
```

## 自定义错误处理

```go
authMiddleware := dominguard.NewAuthMiddleware(dominguard.MiddlewareConfig{
    Guard: guard,
    GetUserID: getUserID,
    GetTenantID: getTenantID,
    ErrorHandler: func(c *gin.Context, err error) {
        // 自定义错误响应
        if permErr, ok := err.(*dominguard.PermissionError); ok {
            c.JSON(http.StatusForbidden, gin.H{
                "code": permErr.Code,
                "message": permErr.Message,
                "timestamp": time.Now().Unix(),
            })
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": err.Error(),
        })
    },
})
```

## 缓存管理

```go
// 获取缓存的版本号
version, exists := guard.GetCachedVersion(tenantID)

// 手动设置缓存版本
guard.SetCachedVersion(tenantID, newVersion)

// 缓存会自动刷新（通过 Redis 订阅）
// 当策略变更时，缓存会自动清空
```

## 架构说明

```
业务服务
    ↓
DomainGuard (PEP SDK)
    ↓
Casbin Enforcer
    ↓
策略规则 (存储在数据库)
```

DomainGuard 作为 PEP (Policy Enforcement Point)，负责：
1. 提供简单的权限检查 API
2. 缓存策略版本，减少查询
3. 监听策略变更，自动刷新
4. 提供中间件，保护路由

## 最佳实践

### 1. 统一提取用户信息

```go
// 创建通用的用户信息提取函数
func GetUserID(c *gin.Context) string {
    // 从 JWT Token 中提取
    claims := c.MustGet("claims").(*jwt.Claims)
    return claims.UserID
}

func GetTenantID(c *gin.Context) string {
    // 从请求头中提取
    return c.GetHeader("X-Tenant-ID")
}

// 在所有中间件中复用
authMiddleware := dominguard.NewAuthMiddleware(dominguard.MiddlewareConfig{
    Guard: guard,
    GetUserID: GetUserID,
    GetTenantID: GetTenantID,
})
```

### 2. 预先注册资源

```go
// 在应用启动时注册所有资源
func registerResources(guard *dominguard.DomainGuard) {
    guard.RegisterResource("order", "订单")
    guard.RegisterResource("product", "产品")
    guard.RegisterResource("user", "用户")
    guard.RegisterResource("inventory", "库存")
}
```

### 3. 分层权限检查

```go
// 在 Controller 层使用中间件进行粗粒度检查
router.GET("/orders", authMiddleware.RequirePermission("order", "read"), ...)

// 在 Service 层进行细粒度检查
func (s *OrderService) GetOrder(ctx context.Context, userID, orderID string) {
    // 检查是否有权限访问特定订单
    order := s.repo.GetOrder(orderID)
    if order.UserID != userID {
        // 检查是否有管理员权限
        allowed, _ := s.guard.CheckPermission(ctx, userID, tenantID, "order", "admin")
        if !allowed {
            return ErrPermissionDenied
        }
    }
    // ...
}
```

## 性能建议

1. **启用缓存**: 配置合适的 `CacheTTL`，减少重复查询
2. **使用批量检查**: 对于多个权限检查，使用 `BatchCheckPermissions`
3. **启用 Redis 订阅**: 实时刷新缓存，保持策略最新
4. **合理设置跳过路径**: 对于公开路径（如健康检查），跳过权限检查

## 许可证

MIT
