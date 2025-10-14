# 认证中间件使用指南

## 📌 概述

IAM 系统现在提供了基于新认证模块的 JWT 中间件，用于保护 API 端点。

## 🏗️ 架构

```
┌────────────────────────────────────────────────────────────────┐
│                      HTTP Request                               │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│               JWT 认证中间件 (jwt_middleware.go)                │
│  - 从 Header/Query/Cookie 提取令牌                              │
│  - 调用 TokenService.VerifyToken() 验证                         │
│  - 将用户信息存入 Gin Context                                   │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│             TokenService (Application Layer)                    │
│  - VerifyToken() 验证令牌并检查黑名单                           │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│             TokenService (Domain Layer)                         │
│  - VerifyAccessToken() 解析 JWT                                │
│  - 检查过期时间                                                  │
│  - 检查黑名单                                                    │
└────────────────────────────────────────────────────────────────┘
```

## 🔧 使用方法

### 1. 在 Container 中集成

首先需要更新容器初始化，将 Redis 传递给认证模块：

```go
// internal/apiserver/container/container.go

func (c *Container) Initialize(redisClient *redis.Client) error {
    // ...
    
    // 初始化认证模块（传入 db 和 redis）
    if err := c.initAuthModule(redisClient); err != nil {
        return fmt.Errorf("failed to initialize auth module: %w", err)
    }
    
    // ...
}

func (c *Container) initAuthModule(redisClient *redis.Client) error {
    authModule := assembler.NewAuthModule()
    if err := authModule.Initialize(c.mysqlDB, redisClient); err != nil {
        return fmt.Errorf("failed to initialize auth module: %w", err)
    }
    c.AuthModule = authModule
    return nil
}
```

### 2. 在路由中使用中间件

#### 方式 A：全局保护所有 API

```go
// internal/apiserver/routers.go

import (
    authnMiddleware "github.com/fangcun-mount/iam-contracts/internal/pkg/middleware/authn"
)

func (r *Router) RegisterRoutes(engine *gin.Engine) {
    // 注册公开路由（无需认证）
    r.registerPublicRoutes(engine)
    
    // 创建认证中间件
    authMiddleware := authnMiddleware.NewJWTAuthMiddleware(
        r.container.AuthModule.TokenService,
    )
    
    // 需要认证的 API 组
    apiV1 := engine.Group("/api/v1")
    apiV1.Use(authMiddleware.AuthRequired()) // 所有 v1 API 都需要认证
    {
        // 用户相关端点
        userhttp.Register(apiV1)
        
        // 其他受保护端点
        apiV1.GET("/profile", r.getProfile)
        apiV1.POST("/posts", r.createPost)
    }
}

func (r *Router) registerPublicRoutes(engine *gin.Engine) {
    // 注册认证端点（登录、刷新等）
    authnhttp.Register(engine)
    
    // 其他公开端点
    engine.GET("/health", r.healthCheck)
    engine.GET("/api/v1/public/info", r.publicInfo)
}
```

#### 方式 B：按需保护特定端点

```go
func (r *Router) RegisterRoutes(engine *gin.Engine) {
    authMiddleware := authnMiddleware.NewJWTAuthMiddleware(
        r.container.AuthModule.TokenService,
    )
    
    apiV1 := engine.Group("/api/v1")
    {
        // 公开端点（无认证）
        apiV1.GET("/posts", r.listPosts)
        apiV1.GET("/posts/:id", r.getPost)
        
        // 需要认证的端点
        apiV1.POST("/posts", 
            authMiddleware.AuthRequired(),  // 添加认证中间件
            r.createPost,
        )
        apiV1.PUT("/posts/:id",
            authMiddleware.AuthRequired(),
            r.updatePost,
        )
        apiV1.DELETE("/posts/:id",
            authMiddleware.AuthRequired(),
            r.deletePost,
        )
        
        // 需要特定角色的端点
        apiV1.GET("/admin/users",
            authMiddleware.AuthRequired(),
            authMiddleware.RequireRole("admin"),  // 需要管理员角色
            r.listUsers,
        )
    }
}
```

#### 方式 C：可选认证（登录用户显示更多信息）

```go
func (r *Router) RegisterRoutes(engine *gin.Engine) {
    authMiddleware := authnMiddleware.NewJWTAuthMiddleware(
        r.container.AuthModule.TokenService,
    )
    
    apiV1 := engine.Group("/api/v1")
    {
        // 可选认证：有令牌则验证，没有也能访问
        apiV1.GET("/posts",
            authMiddleware.AuthOptional(),  // 可选认证
            r.listPosts,  // 已登录用户可能看到更多内容
        )
    }
}

func (r *Router) listPosts(c *gin.Context) {
    // 检查是否已认证
    if userID, ok := authnMiddleware.GetCurrentUserID(c); ok {
        // 已登录用户：显示私密帖子
        log.Infof("User %s is viewing posts", userID)
    } else {
        // 未登录：只显示公开帖子
        log.Info("Anonymous user is viewing posts")
    }
    
    // ... 业务逻辑
}
```

### 3. 在 Handler 中获取用户信息

```go
package handler

import (
    "github.com/gin-gonic/gin"
    authnMiddleware "github.com/fangcun-mount/iam-contracts/internal/pkg/middleware/authn"
    "github.com/fangcun-mount/iam-contracts/pkg/core"
)

type PostHandler struct {}

func (h *PostHandler) CreatePost(c *gin.Context) {
    // 方式 1：使用辅助函数
    userID, ok := authnMiddleware.GetCurrentUserID(c)
    if !ok {
        core.WriteResponse(c, errors.New("Not authenticated"), nil)
        return
    }
    
    accountID, _ := authnMiddleware.GetCurrentAccountID(c)
    sessionID, _ := authnMiddleware.GetCurrentSessionID(c)
    
    log.Infof("User %s (account=%s, session=%s) is creating a post", 
        userID, accountID, sessionID)
    
    // 方式 2：直接从 Context 获取
    if val, exists := c.Get("user_id"); exists {
        userID := val.(uint64)
        // ...
    }
    
    // ... 创建帖子逻辑
}

func (h *PostHandler) GetProfile(c *gin.Context) {
    userID, _ := authnMiddleware.GetCurrentUserID(c)
    
    // 查询用户信息
    // ...
}
```

## 🔐 令牌提取方式

中间件支持多种方式传递令牌（按优先级）：

### 1. Authorization Header (推荐)

```bash
# 标准 Bearer 格式
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
     http://localhost:8080/api/v1/profile

# 直接传递令牌（无 Bearer 前缀）
curl -H "Authorization: eyJhbGciOiJIUzI1NiIs..." \
     http://localhost:8080/api/v1/profile
```

### 2. Query Parameter

```bash
curl "http://localhost:8080/api/v1/profile?token=eyJhbGciOiJIUzI1NiIs..."
```

### 3. Cookie

```bash
curl -b "access_token=eyJhbGciOiJIUzI1NiIs..." \
     http://localhost:8080/api/v1/profile
```

## 📊 完整示例：集成到现有系统

### Step 1: 更新 server.go 添加 Redis

```go
// internal/apiserver/server.go

func (s *apiServer) PrepareRun() preparedAPIServer {
    // ... 现有代码 ...
    
    // 初始化 Redis 客户端
    redisClient := s.initRedis()
    
    // 创建容器并传入 MySQL 和 Redis
    s.container = container.NewContainer(mysqlDB, redisClient)
    
    // ... 其余代码 ...
}

func (s *apiServer) initRedis() *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:     viper.GetString("redis.addr"),
        Password: viper.GetString("redis.password"),
        DB:       viper.GetInt("redis.db"),
    })
}
```

### Step 2: 更新 container.go

```go
// internal/apiserver/container/container.go

type Container struct {
    mysqlDB     *gorm.DB
    redisClient *redis.Client  // 新增
    
    AuthModule *assembler.AuthModule
    UserModule *assembler.UserModule
    
    initialized bool
}

func NewContainer(mysqlDB *gorm.DB, redisClient *redis.Client) *Container {
    return &Container{
        mysqlDB:     mysqlDB,
        redisClient: redisClient,
    }
}

func (c *Container) Initialize() error {
    // ...
    
    // 初始化认证模块（传入 db 和 redis）
    if err := c.initAuthModule(); err != nil {
        return fmt.Errorf("failed to initialize auth module: %w", err)
    }
    
    // ...
}

func (c *Container) initAuthModule() error {
    authModule := assembler.NewAuthModule()
    if err := authModule.Initialize(c.mysqlDB, c.redisClient); err != nil {
        return fmt.Errorf("failed to initialize auth module: %w", err)
    }
    c.AuthModule = authModule
    return nil
}
```

### Step 3: 更新路由注册

```go
// internal/apiserver/routers.go

func (r *Router) RegisterRoutes(engine *gin.Engine) {
    // 1. 注册基础路由（health check 等）
    r.registerBaseRoutes(engine)
    
    // 2. 注册认证端点（公开，无需认证）
    authnhttp.Provide(authnhttp.Dependencies{
        AuthHandler:    r.container.AuthModule.AuthHandler,
        AccountHandler: r.container.AuthModule.AccountHandler,
    })
    authnhttp.Register(engine)
    
    // 3. 创建认证中间件
    authMiddleware := authnMiddleware.NewJWTAuthMiddleware(
        r.container.AuthModule.TokenService,
    )
    
    // 4. 注册受保护的 API
    apiV1 := engine.Group("/api/v1")
    apiV1.Use(authMiddleware.AuthRequired()) // 全局认证
    {
        // 用户模块
        userhttp.Provide(userhttp.Dependencies{
            Module: r.container.UserModule,
        })
        userhttp.Register(apiV1)
        
        // 管理员路由
        r.registerAdminRoutes(apiV1, authMiddleware)
    }
}

func (r *Router) registerAdminRoutes(group *gin.RouterGroup, authMiddleware *authnMiddleware.JWTAuthMiddleware) {
    admin := group.Group("/admin")
    admin.Use(authMiddleware.RequireRole("admin"))  // 需要管理员角色
    {
        admin.GET("/users", r.listAllUsers)
        admin.GET("/statistics", r.getStatistics)
    }
}
```

### Step 4: 配置文件添加 Redis

```yaml
# configs/apiserver.yaml

# Redis 配置
redis:
  addr: "localhost:6379"
  password: ""
  db: 0

# JWT 配置
jwt:
  secret: "your-secret-key-change-in-production"
  access-token-ttl: 15m    # 访问令牌有效期
  refresh-token-ttl: 168h  # 刷新令牌有效期（7天）

# 微信配置
wechat:
  mini-programs:
    - app-id: "wx1234567890abcdef"
      app-secret: "your-app-secret"
```

## 🔄 认证流程

### 1. 用户登录

```bash
# 1. 用户登录（基础认证）
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "method": "basic",
    "credentials": {
      "username": "alice",
      "password": "password123"
    }
  }'

# 响应
{
  "code": 100001,
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIs...",
    "refreshToken": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "tokenType": "Bearer",
    "expiresIn": 900,
    "jti": "token-id-123"
  }
}
```

### 2. 访问受保护资源

```bash
# 使用访问令牌
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
     http://localhost:8080/api/v1/users/me

# 响应
{
  "code": 100001,
  "data": {
    "id": 1,
    "username": "alice",
    "email": "alice@example.com"
  }
}
```

### 3. 令牌过期后刷新

```bash
# 访问令牌过期（15分钟后）
curl -H "Authorization: Bearer expired-token" \
     http://localhost:8080/api/v1/users/me

# 响应（401）
{
  "code": 100005,
  "message": "Token invalid or expired"
}

# 使用刷新令牌获取新的访问令牌
curl -X POST http://localhost:8080/api/v1/auth/refresh_token \
  -H "Content-Type: application/json" \
  -d '{
    "refreshToken": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  }'

# 响应
{
  "code": 100001,
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIs...",  // 新的访问令牌
    "refreshToken": "x9y8z7w6-v5u4-3210-wxyz-9876543210ab",  // 新的刷新令牌（轮换）
    "tokenType": "Bearer",
    "expiresIn": 900
  }
}
```

### 4. 登出

```bash
# 单个令牌登出
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Content-Type: application/json" \
  -d '{
    "refreshToken": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  }'

# 所有令牌登出（所有设备）
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Content-Type: application/json" \
  -d '{
    "refreshToken": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "all": true
  }'
```

## 🛡️ 安全特性

1. **JWT 签名验证**：所有访问令牌都使用 HS256 签名
2. **令牌过期检查**：访问令牌 15 分钟过期
3. **黑名单机制**：登出后的令牌立即失效
4. **刷新令牌轮换**：每次刷新都生成新的刷新令牌
5. **多设备登出**：支持单设备或全部设备登出

## 🔨 待实现功能

- [ ] **角色系统**：`RequireRole()` 中间件目前暂时放行
- [ ] **权限系统**：`RequirePermission()` 中间件目前暂时放行
- [ ] **速率限制**：防止暴力破解
- [ ] **审计日志**：记录所有认证事件
- [ ] **令牌刷新限制**：防止刷新令牌滥用

## 📝 最佳实践

1. **使用 HTTPS**：生产环境必须使用 HTTPS 传输令牌
2. **短期访问令牌**：访问令牌设置较短的有效期（15分钟）
3. **刷新令牌轮换**：每次刷新都更换刷新令牌
4. **安全存储**：前端将刷新令牌存储在 HttpOnly Cookie 中
5. **及时登出**：用户登出后立即吊销令牌
6. **定期清理**：定期清理过期的刷新令牌（Redis TTL 自动处理）

## 🔍 调试技巧

### 查看令牌内容（仅开发环境）

```bash
# 解码 JWT（注意：不验证签名）
echo "eyJhbGciOiJIUzI1NiIs..." | cut -d. -f2 | base64 -d | jq
```

### 验证令牌

```bash
curl -X POST http://localhost:8080/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{
    "token": "eyJhbGciOiJIUzI1NiIs..."
  }'
```

## 📚 相关文档

- [认证模块设计文档](./authentication-design.md)
- [认证模块实现总结](./authentication-implementation-summary.md)
- [快速参考指南](./authentication-quick-reference.md)
