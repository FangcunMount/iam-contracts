# 故障排查: 所有 API 返回 404

## 问题描述

所有 RESTful API 端点都返回 404 错误:

```bash
curl -X POST https://iam.yangshujie.com/api/v1/accounts/operation
# 响应: 404 page not found

curl -X POST https://iam.yangshujie.com/api/v1/auth/login  
# 响应: 404 page not found
```

## 根本原因

**Container 初始化采用"全成功或全失败"策略,任何一个模块失败都会导致整个初始化失败:**

```go
// 旧代码 - 问题所在
func (c *Container) Initialize() error {
    // 如果 IDP 模块失败,这里就返回错误
    if err := c.initIDPModule(); err != nil {
        return fmt.Errorf("failed to initialize idp module: %w", err)
    }
    
    // 后续所有模块都不会初始化
    if err := c.initAuthModule(); err != nil { ... }
    if err := c.initUserModule(); err != nil { ... }
    if err := c.initAuthzModule(); err != nil { ... }
}
```

**导致的问题链:**

1. **IDP 模块初始化失败** (可能是 Redis 连接失败)
   ↓
2. **Container.Initialize() 返回错误**
   ↓
3. **所有后续模块 (Authn, User, Authz) 都没有初始化**
   ↓
4. **r.container.AuthnModule == nil**
   ↓
5. **路由注册时 handlers 都是 nil,路由不会被注册**
   ↓
6. **所有 API 请求返回 404**

## 具体原因

### 1. Container 初始化策略过于严格

```go
// server.go
if err := s.container.Initialize(); err != nil {
    log.Warnf("Failed to initialize hexagonal architecture container: %v", err)
    // ⚠️ 只是警告,不阻止启动,但 container 内的模块都是 nil
}
```

### 2. 路由注册依赖模块非空

```go
// routers.go (旧代码)
if r.container.AuthnModule != nil {
    authnhttp.Provide(authnhttp.Dependencies{...})
} else {
    authnhttp.Provide(authnhttp.Dependencies{}) // 空依赖
}

authnhttp.Register(engine) // ❌ handlers 都是 nil,不会注册任何路由
```

### 3. Register 函数的 nil 检查

```go
// router.go
func registerAuthEndpointsV2(group *gin.RouterGroup, handler *authhandler.AuthHandler) {
    if group == nil || handler == nil {
        return // ❌ handler 是 nil,直接返回,路由未注册
    }
    
    group.POST("/login", handler.Login)
    // ...
}
```

## 解决方案

### 1. 使 Container 初始化更加健壮

**允许部分模块失败,其他模块继续初始化:**

```go
// container.go - 修复后
func (c *Container) Initialize() error {
    var errors []error

    // 1. 尝试初始化 IDP 模块
    if err := c.initIDPModule(); err != nil {
        log.Warnf("Failed to initialize IDP module: %v", err)
        errors = append(errors, fmt.Errorf("idp module: %w", err))
        // ✅ 不直接返回,继续初始化其他模块
    }

    // 2. 尝试初始化 Authn 模块
    if err := c.initAuthModule(); err != nil {
        log.Warnf("Failed to initialize Authn module: %v", err)
        errors = append(errors, fmt.Errorf("authn module: %w", err))
    }

    // 3. 尝试初始化 User 模块
    if err := c.initUserModule(); err != nil {
        log.Warnf("Failed to initialize User module: %v", err)
        errors = append(errors, fmt.Errorf("user module: %w", err))
    }

    // 4. 尝试初始化 Authz 模块
    if err := c.initAuthzModule(); err != nil {
        log.Warnf("Failed to initialize Authz module: %v", err)
        errors = append(errors, fmt.Errorf("authz module: %w", err))
    }

    c.initialized = true
    
    // ✅ 打印每个模块的状态
    log.Infof("🏗️  Container initialization completed:")
    if c.IDPModule != nil {
        log.Info("   ✅ IDP module")
    } else {
        log.Warn("   ❌ IDP module failed")
    }
    // ... 其他模块状态
    
    if len(errors) > 0 {
        return fmt.Errorf("some modules failed (%d errors)", len(errors))
    }
    return nil
}
```

### 2. 优化路由注册逻辑

**只在模块非空时才注册路由:**

```go
// routers.go - 修复后
// Authn 模块
if r.container.AuthnModule != nil {
    authnhttp.Provide(authnhttp.Dependencies{
        AuthHandler:    r.container.AuthnModule.AuthHandler,
        AccountHandler: r.container.AuthnModule.AccountHandler,
        JWKSHandler:    r.container.AuthnModule.JWKSHandler,
    })
    authnhttp.Register(engine)  // ✅ 只在模块存在时注册
    log.Info("✅ Authn module routes registered")
} else {
    log.Warn("⚠️  Authn module not initialized, routes not registered")
    // ✅ 不调用 Provide 和 Register
}

// Authz 模块
if r.container.AuthzModule != nil {
    authzhttp.Provide(authzhttp.Dependencies{...})
    authzhttp.Register(engine)
    log.Info("✅ Authz module routes registered")
} else {
    log.Warn("⚠️  Authz module not initialized, routes not registered")
}

// IDP 模块
if r.container.IDPModule != nil {
    idphttp.Provide(idphttp.Dependencies{...})
    idphttp.Register(engine)
    log.Info("✅ IDP module routes registered")
} else {
    log.Warn("⚠️  IDP module not initialized, routes not registered")
}
```

## 诊断步骤

### 1. 检查服务健康状态

```bash
curl https://iam.yangshujie.com/health
```

如果返回健康状态,说明服务在运行,但路由可能没有注册。

### 2. 查看启动日志

查找以下关键日志:

```
Failed to initialize IDP module: ...
Failed to initialize Authn module: ...
Failed to initialize User module: ...
Failed to initialize Authz module: ...
```

或优化后的日志:

```
🏗️  Container initialization completed:
   ✅ IDP module
   ✅ Authn module
   ✅ User module
   ✅ Authz module
```

### 3. 查找路由注册日志

```
✅ Authn module routes registered
✅ Authz module routes registered
✅ IDP module routes registered
✅ User module routes registered
```

或警告:

```
⚠️  Authn module not initialized, routes not registered
⚠️  Authz module not initialized, routes not registered
```

### 4. 测试特定端点

```bash
# 测试认证端点
curl -X POST https://iam.yangshujie.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test"}'

# 测试账户端点
curl -X POST https://iam.yangshujie.com/api/v1/accounts/operation \
  -H "Content-Type: application/json" \
  -d '{"userId":"test","username":"test","password":"test"}'

# 测试用户端点
curl https://iam.yangshujie.com/api/v1/users
```

## 常见失败原因

### 1. Redis 连接失败

```
Failed to connect to Redis cache (127.0.0.1:6379): connection refused
Failed to connect to Redis store (127.0.0.1:6379): connection refused
```

**解决方案:**
- 检查 Redis 是否运行
- 检查 Redis 配置 (host, port, password)
- 检查网络连接

### 2. MySQL 连接失败

```
Failed to get MySQL connection: ...
```

**解决方案:**
- 检查 MySQL 是否运行
- 检查数据库配置
- 检查数据库权限

### 3. 模块依赖问题

IDP 模块失败会导致 Authn 模块失败(因为 Authn 依赖 IDP)。

**解决方案:**
- 修复上游模块(IDP)
- 或修改 Authn 模块使其可以在没有 IDP 的情况下工作

## 修复验证

修复后,重新部署并检查:

### 1. 查看启动日志

应该看到:

```
🏗️  Container initialization completed:
   ✅ IDP module
   ✅ Authn module
   ✅ User module
   ✅ Authz module

✅ Authn module routes registered
✅ Authz module routes registered
✅ IDP module routes registered
✅ User module routes registered
🔗 All routes registration completed
```

### 2. 测试 API

```bash
# 应该返回具体错误,而不是 404
curl -X POST https://iam.yangshujie.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{}'

# 预期: 400 Bad Request (参数错误)
# 而不是: 404 Not Found
```

## 预防措施

### 1. 改进模块初始化

让每个模块的初始化更加健壮,提供降级选项:

```go
func (m *AuthnModule) Initialize(params ...interface{}) error {
    db := params[0].(*gorm.DB)
    redisClient := params[1].(*redis.Client)
    idpModule := params[2].(*IDPModule)
    
    // ✅ 允许在没有 Redis 的情况下运行
    if redisClient == nil {
        log.Warn("Redis not available, token features will be limited")
    }
    
    // ✅ 允许在没有 IDP 的情况下运行
    if idpModule == nil {
        log.Warn("IDP module not available, WeChat auth disabled")
    }
    
    // 继续初始化其他组件...
}
```

### 2. 健康检查改进

在健康检查中包含模块状态:

```go
func (r *Router) healthCheck(c *gin.Context) {
    c.JSON(200, gin.H{
        "status": "healthy",
        "modules": gin.H{
            "authn": r.container.AuthnModule != nil,
            "authz": r.container.AuthzModule != nil,
            "user":  r.container.UserModule != nil,
            "idp":   r.container.IDPModule != nil,
        },
    })
}
```

### 3. 启动失败策略

对于关键模块,可以选择启动失败:

```go
// 如果 Authn 模块初始化失败,整个服务启动失败
if r.container.AuthnModule == nil {
    log.Fatal("Authn module is required but failed to initialize")
}
```

## 总结

- ✅ Container 初始化改为"尽力而为",允许部分失败
- ✅ 路由注册时检查模块是否为空
- ✅ 添加详细的初始化状态日志
- ✅ 每个模块独立失败,不影响其他模块

现在即使某个模块失败,其他模块仍然可以正常工作,相关 API 可以正常访问。
