# 🔐 IAM 系统认证集成方案

## 问题

当前 IAM 系统该如何进行认证？

## 解决方案

### 现状分析

目前系统有**两套认证机制**：

1. **旧系统** (`internal/apiserver/auth.go`)
   - 使用 `github.com/appleboy/gin-jwt/v2`
   - 硬编码认证逻辑（admin/admin123）
   - 不与数据库集成

2. **新系统** (`internal/apiserver/modules/authn/`)
   - 完整的 DDD 4 层架构
   - 支持多种认证方式（Basic、WeChat）
   - 与数据库和 Redis 集成
   - 完整的令牌管理

### 推荐方案：使用新认证模块 + JWT 中间件

我已经创建了一个新的认证中间件：`internal/pkg/middleware/authn/jwt_middleware.go`

## 快速开始

### 1. 更新容器初始化

```go
// internal/apiserver/server.go

func (s *apiServer) PrepareRun() preparedAPIServer {
    // 初始化数据库
    mysqlDB, _ := s.dbManager.GetMySQLDB()
    
    // 初始化 Redis（新增）
    redisClient := redis.NewClient(&redis.Options{
        Addr:     viper.GetString("redis.addr"),
        Password: viper.GetString("redis.password"),
        DB:       viper.GetInt("redis.db"),
    })
    
    // 创建容器（传入 MySQL 和 Redis）
    s.container = container.NewContainer(mysqlDB, redisClient)
    s.container.Initialize()
    
    // ... 其余代码
}
```

### 2. 更新 Container

```go
// internal/apiserver/container/container.go

type Container struct {
    mysqlDB     *gorm.DB
    redisClient *redis.Client  // 新增
    
    AuthModule *assembler.AuthModule
    UserModule *assembler.UserModule
}

func NewContainer(mysqlDB *gorm.DB, redisClient *redis.Client) *Container {
    return &Container{
        mysqlDB:     mysqlDB,
        redisClient: redisClient,
    }
}

func (c *Container) initAuthModule() error {
    authModule := assembler.NewAuthModule()
    // 传入 MySQL 和 Redis
    if err := authModule.Initialize(c.mysqlDB, c.redisClient); err != nil {
        return err
    }
    c.AuthModule = authModule
    return nil
}
```

### 3. 更新路由注册

```go
// internal/apiserver/routers.go

import (
    authnMiddleware "github.com/fangcun-mount/iam-contracts/internal/pkg/middleware/authn"
)

func (r *Router) RegisterRoutes(engine *gin.Engine) {
    // 1. 基础路由
    r.registerBaseRoutes(engine)
    
    // 2. 公开的认证端点（登录、刷新等）
    authnhttp.Provide(authnhttp.Dependencies{
        AuthHandler:    r.container.AuthModule.AuthHandler,
        AccountHandler: r.container.AuthModule.AccountHandler,
    })
    authnhttp.Register(engine)
    
    // 3. 创建认证中间件
    authMiddleware := authnMiddleware.NewJWTAuthMiddleware(
        r.container.AuthModule.TokenService,
    )
    
    // 4. 受保护的 API（需要认证）
    apiV1 := engine.Group("/api/v1")
    apiV1.Use(authMiddleware.AuthRequired())  // 全局认证
    {
        userhttp.Register(apiV1)
        // ... 其他需要认证的端点
    }
}
```

### 4. 配置文件

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
  access-token-ttl: 15m
  refresh-token-ttl: 168h
```

## 使用示例

### 1. 用户登录

```bash
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
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "uuid-string",
  "tokenType": "Bearer",
  "expiresIn": 900
}
```

### 2. 访问受保护资源

```bash
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
     http://localhost:8080/api/v1/users/me
```

### 3. 刷新令牌

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh_token \
  -H "Content-Type: application/json" \
  -d '{"refreshToken": "uuid-string"}'
```

### 4. 在 Handler 中获取用户信息

```go
import authnMiddleware "github.com/fangcun-mount/iam-contracts/internal/pkg/middleware/authn"

func (h *Handler) GetProfile(c *gin.Context) {
    userID, ok := authnMiddleware.GetCurrentUserID(c)
    if !ok {
        core.WriteResponse(c, errors.New("Not authenticated"), nil)
        return
    }
    
    accountID, _ := authnMiddleware.GetCurrentAccountID(c)
    
    // 使用 userID 和 accountID 查询数据
}
```

## 中间件选项

### 1. 必需认证

```go
apiV1.Use(authMiddleware.AuthRequired())  // 所有端点都需要认证
```

### 2. 可选认证

```go
apiV1.Use(authMiddleware.AuthOptional())  // 有令牌则验证，没有也允许通过
```

### 3. 按端点认证

```go
apiV1.GET("/public", r.publicHandler)  // 无需认证
apiV1.GET("/private", 
    authMiddleware.AuthRequired(),  // 单个端点需要认证
    r.privateHandler,
)
```

### 4. 角色/权限控制（待实现）

```go
admin := apiV1.Group("/admin")
admin.Use(authMiddleware.AuthRequired())
admin.Use(authMiddleware.RequireRole("admin"))  // 需要管理员角色
{
    admin.GET("/users", r.listUsers)
}
```

## 认证端点

新认证模块提供的端点：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/auth/login` | POST | 统一登录（支持 basic、wx:minip） |
| `/api/v1/auth/refresh_token` | POST | 刷新令牌 |
| `/api/v1/auth/logout` | POST | 登出 |
| `/api/v1/auth/verify` | POST | 验证令牌 |
| `/.well-known/jwks.json` | GET | JWKS 公钥 |

## 迁移步骤

### 阶段 1：并行运行（当前）

- ✅ 新认证模块已实现
- ✅ JWT 中间件已创建
- ⏳ 旧的 gin-jwt 仍在使用

### 阶段 2：切换到新系统（推荐）

1. 更新 Container 支持 Redis
2. 更新路由使用新的 JWT 中间件
3. 配置 Redis 和 JWT 参数
4. 测试认证流程

### 阶段 3：移除旧系统

1. 删除 `internal/apiserver/auth.go`
2. 删除旧的 gin-jwt 依赖
3. 清理未使用的代码

## 优势

使用新认证系统的优势：

1. ✅ **数据库集成**：真实的用户认证，不再硬编码
2. ✅ **多种认证方式**：支持用户名密码、微信小程序等
3. ✅ **完整令牌管理**：访问令牌 + 刷新令牌 + 黑名单
4. ✅ **安全性**：刷新令牌轮换、令牌过期、黑名单机制
5. ✅ **可扩展**：DDD 架构，易于添加新的认证方式
6. ✅ **生产就绪**：完整的错误处理和日志记录

## 待办事项

- [ ] 实现角色系统和 RequireRole 中间件
- [ ] 实现权限系统和 RequirePermission 中间件
- [ ] 添加速率限制防止暴力破解
- [ ] 添加审计日志记录认证事件
- [ ] 从配置文件加载 JWT 密钥和 TTL

## 相关文档

- [认证中间件使用指南](./authentication-middleware-guide.md) - 详细的使用文档
- [认证模块实现总结](./authentication-implementation-summary.md) - 实现细节
- [快速参考](./authentication-quick-reference.md) - API 速查表
