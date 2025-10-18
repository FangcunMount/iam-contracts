# 路由注册实现总结

## 概览

成功完成了 IAM 授权系统的路由注册工作，包括：
1. 创建 AuthzModule assembler
2. 更新 Container 集成 AuthzModule
3. 注册所有 REST API 路由

## 完成的工作

### 1. AuthzModule Assembler (authz.go)

创建了 `internal/apiserver/container/assembler/authz.go`，负责：

#### 模块结构
```go
type AuthzModule struct {
    // Application Services
    RoleService       *role.Service
    AssignmentService *assignment.Service
    PolicyService     *policy.Service
    ResourceService   *resource.Service
    
    // HTTP Handlers
    RoleHandler       *handler.RoleHandler
    AssignmentHandler *handler.AssignmentHandler
    PolicyHandler     *handler.PolicyHandler
    ResourceHandler   *handler.ResourceHandler
    
    // Infrastructure
    Enforcer *casbin.Enforcer
}
```

#### 初始化流程
1. **Casbin Enforcer**: 从配置文件加载 RBAC 模型
2. **仓储层**: Role, Assignment, Resource, PolicyVersion 仓储
3. **适配器层**: CasbinAdapter (策略引擎), VersionNotifier (Redis 通知)
4. **应用服务层**: 4 个应用服务 (Role, Assignment, Policy, Resource)
5. **HTTP 处理器**: 4 个 REST Handler

```go
func (m *AuthzModule) Initialize(db *gorm.DB, redisClient *redis.Client) error {
    // 1. Casbin 适配器
    casbinAdapter, err := casbin.NewCasbinAdapter(db, modelPath)
    
    // 2. 仓储层
    roleRepository := roleInfra.NewRoleRepository(db)
    assignmentRepository := assignmentInfra.NewAssignmentRepository(db)
    resourceRepository := resourceInfra.NewResourceRepository(db)
    policyVersionRepository := policyInfra.NewPolicyVersionRepository(db)
    
    // 3. 版本通知器
    versionNotifier := redis.NewVersionNotifier(nil, "authz:policy_changed")
    
    // 4. 应用服务
    m.RoleService = role.NewService(roleRepository)
    m.AssignmentService = assignment.NewService(assignmentRepository, roleRepository, casbinAdapter)
    m.PolicyService = policy.NewService(policyVersionRepository, roleRepository, resourceRepository, casbinAdapter, versionNotifier)
    m.ResourceService = resource.NewService(resourceRepository)
    
    // 5. HTTP 处理器
    m.RoleHandler = handler.NewRoleHandler(m.RoleService)
    m.AssignmentHandler = handler.NewAssignmentHandler(m.AssignmentService)
    m.PolicyHandler = handler.NewPolicyHandler(m.PolicyService)
    m.ResourceHandler = handler.NewResourceHandler(m.ResourceService)
    
    return nil
}
```

### 2. Container 集成

更新了 `internal/apiserver/container/container.go`:

#### 添加 AuthzModule 字段
```go
type Container struct {
    AuthnModule *assembler.AuthnModule
    UserModule  *assembler.UserModule
    AuthzModule *assembler.AuthzModule  // 新增
}
```

#### 初始化 AuthzModule
```go
func (c *Container) Initialize() error {
    // ... authn, user modules ...
    
    // 初始化授权模块
    if err := c.initAuthzModule(); err != nil {
        return fmt.Errorf("failed to initialize authz module: %w", err)
    }
    
    fmt.Printf("🏗️  Container initialized with modules: user, auth, authz\n")
    return nil
}

func (c *Container) initAuthzModule() error {
    authzModule := assembler.NewAuthzModule()
    if err := authzModule.Initialize(c.mysqlDB, c.redisClient); err != nil {
        return fmt.Errorf("failed to initialize authz module: %w", err)
    }
    c.AuthzModule = authzModule
    return nil
}
```

### 3. 路由注册

#### 更新 authz 路由文件 (`interface/restful/router.go`)

定义了依赖结构和注册函数：

```go
type Dependencies struct {
    RoleHandler       *handler.RoleHandler
    AssignmentHandler *handler.AssignmentHandler
    PolicyHandler     *handler.PolicyHandler
    ResourceHandler   *handler.ResourceHandler
}

func Register(engine *gin.Engine) {
    authzGroup := engine.Group("/api/v1/authz")
    {
        // 健康检查
        authzGroup.GET("/health", healthHandler)
        
        // ============ 角色管理 ============
        roles := authzGroup.Group("/roles")
        {
            roles.POST("", deps.RoleHandler.CreateRole)
            roles.PUT("/:id", deps.RoleHandler.UpdateRole)
            roles.DELETE("/:id", deps.RoleHandler.DeleteRole)
            roles.GET("/:id", deps.RoleHandler.GetRole)
            roles.GET("", deps.RoleHandler.ListRoles)
            roles.GET("/:role_id/assignments", deps.AssignmentHandler.ListAssignmentsByRole)
            roles.GET("/:role_id/policies", deps.PolicyHandler.GetPoliciesByRole)
        }
        
        // ============ 角色分配 ============
        assignments := authzGroup.Group("/assignments")
        {
            assignments.POST("/grant", deps.AssignmentHandler.GrantRole)
            assignments.POST("/revoke", deps.AssignmentHandler.RevokeRole)
            assignments.DELETE("/:id", deps.AssignmentHandler.RevokeRoleByID)
            assignments.GET("/subject", deps.AssignmentHandler.ListAssignmentsBySubject)
        }
        
        // ============ 策略管理 ============
        policies := authzGroup.Group("/policies")
        {
            policies.POST("", deps.PolicyHandler.AddPolicyRule)
            policies.DELETE("", deps.PolicyHandler.RemovePolicyRule)
            policies.GET("/version", deps.PolicyHandler.GetCurrentVersion)
        }
        
        // ============ 资源管理 ============
        resources := authzGroup.Group("/resources")
        {
            resources.POST("", deps.ResourceHandler.CreateResource)
            resources.PUT("/:id", deps.ResourceHandler.UpdateResource)
            resources.DELETE("/:id", deps.ResourceHandler.DeleteResource)
            resources.GET("/:id", deps.ResourceHandler.GetResource)
            resources.GET("/key/:key", deps.ResourceHandler.GetResourceByKey)
            resources.GET("", deps.ResourceHandler.ListResources)
            resources.POST("/validate-action", deps.ResourceHandler.ValidateAction)
        }
    }
}
```

#### 更新主路由文件 (`routers.go`)

连接 Container 和路由注册：

```go
func (r *Router) RegisterRoutes(engine *gin.Engine) {
    // ... 其他模块 ...
    
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
    
    authzhttp.Register(engine)
    
    fmt.Printf("🔗 Registered routes for: base, user, authn, authz\n")
}
```

## 完整的 REST API 路由表

### 角色管理 (7 个端点)
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/authz/roles` | 创建角色 |
| PUT | `/api/v1/authz/roles/:id` | 更新角色 |
| DELETE | `/api/v1/authz/roles/:id` | 删除角色 |
| GET | `/api/v1/authz/roles/:id` | 获取角色详情 |
| GET | `/api/v1/authz/roles` | 列出角色 (分页) |
| GET | `/api/v1/authz/roles/:role_id/assignments` | 列出角色的分配记录 |
| GET | `/api/v1/authz/roles/:role_id/policies` | 获取角色的策略列表 |

### 角色分配 (4 个端点)
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/authz/assignments/grant` | 授予角色 |
| POST | `/api/v1/authz/assignments/revoke` | 撤销角色 |
| DELETE | `/api/v1/authz/assignments/:id` | 根据ID撤销角色 |
| GET | `/api/v1/authz/assignments/subject` | 列出主体的分配 (query: subject_type, subject_id) |

### 策略管理 (3 个端点)
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/authz/policies` | 添加策略规则 |
| DELETE | `/api/v1/authz/policies` | 移除策略规则 |
| GET | `/api/v1/authz/policies/version` | 获取当前策略版本 |

### 资源管理 (7 个端点)
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/authz/resources` | 创建资源 |
| PUT | `/api/v1/authz/resources/:id` | 更新资源 |
| DELETE | `/api/v1/authz/resources/:id` | 删除资源 |
| GET | `/api/v1/authz/resources/:id` | 获取资源详情 |
| GET | `/api/v1/authz/resources/key/:key` | 根据键获取资源 |
| GET | `/api/v1/authz/resources` | 列出资源 (分页) |
| POST | `/api/v1/authz/resources/validate-action` | 验证资源动作 |

**总计**: 21 个 REST API 端点

## 技术亮点

### 1. 依赖注入分层
```
Container
  ↓ (初始化)
AuthzModule
  ↓ (组装)
Services + Handlers
  ↓ (注入)
Router Dependencies
  ↓ (注册)
Gin Routes
```

### 2. 模块化设计
- **Assembler 模式**: 集中管理模块依赖组装
- **Container 模式**: 统一管理所有模块生命周期
- **依赖分离**: Handler 只依赖 Service 接口，不依赖具体实现

### 3. 路由分组
使用 Gin 的路由组功能，清晰划分：
- `/api/v1/authz/roles/*` - 角色管理
- `/api/v1/authz/assignments/*` - 角色分配
- `/api/v1/authz/policies/*` - 策略管理
- `/api/v1/authz/resources/*` - 资源管理

### 4. 健壮性
- **空安全**: 检查 `AuthzModule != nil` 再注册路由
- **错误传播**: 所有初始化错误向上传播
- **日志输出**: 清晰的模块初始化和路由注册日志

### 5. RESTful 设计
- **资源命名**: 使用复数名词 (roles, assignments, policies, resources)
- **HTTP 方法**: GET (查询), POST (创建), PUT (更新), DELETE (删除)
- **幂等性**: PUT/DELETE 操作保持幂等性
- **嵌套路由**: `/roles/:role_id/assignments` 表达资源关系

## 编译验证

```bash
$ go build ./cmd/apiserver/...
# 成功编译，无错误
```

## 待完成的工作

### 1. Casbin 模型文件
需要创建 `configs/casbin_model.conf`:
```ini
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act
```

### 2. Redis 版本兼容
当前 VersionNotifier 使用 `github.com/redis/go-redis/v9`，而 Container 使用 `github.com/go-redis/redis/v7`。
需要：
- 升级 Container 到 redis v9，或
- 创建适配器桥接两个版本

### 3. 数据库迁移
需要执行数据库迁移，创建必要的表：
- `authz_roles` - 角色表
- `authz_assignments` - 角色分配表
- `authz_resources` - 资源表
- `authz_policy_versions` - 策略版本表
- `casbin_rule` - Casbin 策略表

### 4. 中间件
建议添加：
- **认证中间件**: 验证 JWT token
- **租户提取中间件**: 自动提取 `X-Tenant-ID`
- **审计日志中间件**: 记录所有 PAP 操作

### 5. PEP SDK
创建策略执行点 SDK，供业务服务使用

## 架构图

```
HTTP Request
    ↓
Gin Router (routers.go)
    ↓
authzhttp.Register()
    ↓
Handler (interface/restful/handler/)
    ├─ getTenantID()
    ├─ getUserID()
    └─ handleError()
    ↓
Application Service (application/)
    ├─ RoleService
    ├─ AssignmentService
    ├─ PolicyService
    └─ ResourceService
    ↓
Domain Layer (domain/)
    ├─ Role Aggregate
    ├─ Assignment Aggregate
    ├─ Resource Aggregate
    └─ Policy Aggregate
    ↓
Infrastructure (infra/)
    ├─ MySQL Repositories
    ├─ Casbin Adapter
    └─ Redis Notifier
    ↓
External Systems
    ├─ MySQL Database
    ├─ Redis Cache
    └─ Casbin Enforcer
```

## 下一步

根据 todolist，接下来是：
1. **创建 PEP SDK (DomainGuard)** - 提供流畅的权限检查 API
2. **集成测试** - 测试完整的端到端流程

---

**文件更新清单**:
- ✅ `internal/apiserver/container/assembler/authz.go` (新建)
- ✅ `internal/apiserver/container/container.go` (更新)
- ✅ `internal/apiserver/modules/authz/interface/restful/router.go` (重写)
- ✅ `internal/apiserver/routers.go` (更新)

**编译状态**: ✅ 通过
