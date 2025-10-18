# Authz 模块架构重构计划 V2（正确的 DDD + CQRS 方案）

## 问题分析

当前 authz 模块存在的架构问题：

1. ❌ **缺少 driving 端口定义**：没有在 domain/port/driving 中定义用例接口
2. ❌ **未遵循 CQRS**：Command 和 Query 没有分离
3. ❌ **应用服务职责不清**：应用服务直接包含业务逻辑，而不是实现 driving 接口
4. ❌ **领域服务暴露不当**：领域服务不应该直接被应用层调用，而是通过端口

## 正确的架构模式（参考 authn 模块）

```
┌─────────────────────────────────────────────────────────────┐
│                     Interface Layer                          │
│  (REST Handlers) - 调用 driving 接口                         │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│                  Application Layer                           │
│  - 实现 domain/port/driving 接口                             │
│  - 编排领域服务和领域对象                                    │
│  - 管理事务边界                                              │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│                    Domain Layer                              │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  port/driving (驱动端口 - 用例接口)                   │  │
│  │  - ResourceCommander (创建、更新、删除)               │  │
│  │  - ResourceQueryer (查询)                             │  │
│  │  - RoleCommander / RoleQueryer (CQRS 分离)           │  │
│  │  - PolicyCommander / PolicyQueryer                    │  │
│  │  - AssignmentCommander / AssignmentQueryer            │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  service (领域服务 - 封装业务规则)                    │  │
│  │  - ResourceManager (唯一性检查、验证)                 │  │
│  │  - RoleManager (角色业务规则)                         │  │
│  │  - PolicyManager (策略规则、版本管理)                 │  │
│  │  - AssignmentManager (Casbin 规则、事务)             │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  port/driven (被驱动端口 - 仓储接口)                  │  │
│  │  - ResourceRepo, RoleRepo, PolicyRepo, AssignmentRepo│  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  entity (领域实体)                                     │  │
│  │  - Resource, Role, Policy, Assignment                 │  │
│  └──────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│                Infrastructure Layer                          │
│  - 实现 port/driven 接口                                     │
│  - MySQL, Redis, Casbin                                     │
└─────────────────────────────────────────────────────────────┘
```

## 核心原则

### 1. 端口与适配器（Hexagonal Architecture）

- **Driving Ports（驱动端口）**：在 `domain/port/driving` 定义用例接口
- **Driven Ports（被驱动端口）**：在 `domain/port/driven` 定义仓储接口
- **应用层实现 Driving Ports**
- **基础设施层实现 Driven Ports**

### 2. CQRS（命令查询职责分离）

- **Commander 接口**：处理写操作（Create, Update, Delete）
- **Queryer 接口**：处理读操作（Get, List, Validate）
- **好处**：
  - 读写分离，可以独立优化
  - 接口职责单一，易于理解和维护
  - 便于实现不同的优化策略（读缓存、写事务）

### 3. 领域服务的正确使用

- **领域服务不对外暴露**：不被接口层直接调用
- **被应用服务编排**：应用服务调用领域服务完成业务逻辑
- **封装业务规则**：唯一性检查、验证、复杂计算

## 重构步骤（正确顺序）

### 阶段 1：清理现有错误架构（1小时）

**任务：删除错误的 ResourceManager 和相关代码**

```bash
# 删除文件
rm internal/apiserver/modules/authz/domain/resource/service/manager.go

# 恢复 application/resource/service.go 到原始状态
git checkout internal/apiserver/modules/authz/application/resource/service.go
```

**理由**：

- 当前的 ResourceManager 设计是错误的
- 领域服务不应该直接被注入到应用服务
- 需要先定义 driving 端口

---

### 阶段 2：定义 Resource Driving 端口（CQRS）（1.5小时）

**2.1 创建 ResourceCommander 接口**

```go
// domain/resource/port/driving/commander.go
package driving

type ResourceCommander interface {
    CreateResource(ctx context.Context, cmd CreateResourceCommand) (*resource.Resource, error)
    UpdateResource(ctx context.Context, cmd UpdateResourceCommand) (*resource.Resource, error)
    DeleteResource(ctx context.Context, resourceID resource.ResourceID) error
}

type CreateResourceCommand struct {
    Key         string
    DisplayName string
    AppName     string
    Domain      string
    Type        string
    Actions     []string
    Description string
}

type UpdateResourceCommand struct {
    ID          resource.ResourceID
    DisplayName *string
    Actions     []string
    Description *string
}
```

**2.2 创建 ResourceQueryer 接口**

```go
// domain/resource/port/driving/queryer.go
package driving

type ResourceQueryer interface {
    GetResourceByID(ctx context.Context, resourceID resource.ResourceID) (*resource.Resource, error)
    GetResourceByKey(ctx context.Context, key string) (*resource.Resource, error)
    ListResources(ctx context.Context, query ListResourcesQuery) (*ListResourcesResult, error)
    ValidateAction(ctx context.Context, resourceKey, action string) (bool, error)
}

type ListResourcesQuery struct {
    Offset int
    Limit  int
}

type ListResourcesResult struct {
    Resources []*resource.Resource
    Total     int64
}
```

---

### 阶段 3：创建 Resource 领域服务（内部使用）（1小时）

**3.1 创建 ResourceManager（不对外暴露）**

```go
// domain/resource/service/manager.go
package service

// ResourceManager 资源管理领域服务（内部使用，不对外暴露）
// 职责：封装资源相关的业务规则
type ResourceManager struct {
    resourceRepo driven.ResourceRepo
}

// 业务规则方法（被应用服务调用）
func (m *ResourceManager) checkKeyUniqueness(ctx context.Context, key string) error
func (m *ResourceManager) validateActions(actions []string) error
func (m *ResourceManager) validateResourceParameters(opts CreateResourceOptions) error
```

**关键点**：

- ✅ 领域服务是内部实现细节
- ✅ 不被接口层直接调用
- ✅ 被应用服务编排使用

---

### 阶段 4：应用服务实现 Driving 接口（1.5小时）

**4.1 ResourceCommandService 实现 ResourceCommander**

```go
// application/resource/command_service.go
package resource

type ResourceCommandService struct {
    resourceManager *service.ResourceManager  // 调用领域服务
    resourceRepo    driven.ResourceRepo
}

// 实现 driving.ResourceCommander 接口
func (s *ResourceCommandService) CreateResource(
    ctx context.Context, 
    cmd driving.CreateResourceCommand,
) (*resource.Resource, error) {
    // 1. 调用领域服务进行业务验证
    if err := s.resourceManager.ValidateCreateParameters(cmd); err != nil {
        return nil, err
    }
    
    // 2. 调用领域服务检查唯一性
    if err := s.resourceManager.CheckKeyUniqueness(ctx, cmd.Key); err != nil {
        return nil, err
    }
    
    // 3. 创建领域对象
    newResource := resource.NewResource(cmd.Key, cmd.Actions, ...)
    
    // 4. 持久化
    if err := s.resourceRepo.Create(ctx, &newResource); err != nil {
        return nil, err
    }
    
    return &newResource, nil
}
```

**4.2 ResourceQueryService 实现 ResourceQueryer**

```go
// application/resource/query_service.go
package resource

type ResourceQueryService struct {
    resourceRepo driven.ResourceRepo
}

// 实现 driving.ResourceQueryer 接口
func (s *ResourceQueryService) GetResourceByID(
    ctx context.Context, 
    resourceID resource.ResourceID,
) (*resource.Resource, error) {
    return s.resourceRepo.FindByID(ctx, resourceID)
}

func (s *ResourceQueryService) ListResources(
    ctx context.Context, 
    query driving.ListResourcesQuery,
) (*driving.ListResourcesResult, error) {
    // 查询逻辑（可以加缓存优化）
    resources, total, err := s.resourceRepo.List(ctx, query.Offset, query.Limit)
    if err != nil {
        return nil, err
    }
    return &driving.ListResourcesResult{
        Resources: resources,
        Total:     total,
    }, nil
}
```

**关键点**：

- ✅ 应用服务实现 driving 接口
- ✅ 应用服务编排领域服务
- ✅ Command 和 Query 分离

---

### 阶段 5：更新接口层和容器（1小时）

**5.1 更新 ResourceHandler**

```go
// interface/restful/handler/resource.go
type ResourceHandler struct {
    commander driving.ResourceCommander  // 依赖接口，不是具体实现
    queryer   driving.ResourceQueryer
}

func (h *ResourceHandler) CreateResource(c *gin.Context) {
    var req CreateResourceRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        // 错误处理
        return
    }
    
    // 转换为 Command
    cmd := driving.CreateResourceCommand{
        Key:         req.Key,
        DisplayName: req.DisplayName,
        // ...
    }
    
    // 调用 driving 接口
    resource, err := h.commander.CreateResource(c.Request.Context(), cmd)
    if err != nil {
        // 错误处理
        return
    }
    
    // 返回响应
    c.JSON(200, toResourceResponse(resource))
}
```

**5.2 更新 AuthzModule Assembler**

```go
// container/assembler/authz.go
func (m *AuthzModule) Initialize(db *gorm.DB, redisClient *redis.Client) error {
    // 1. 初始化仓储层
    resourceRepo := resourceInfra.NewResourceRepository(db)
    
    // 2. 初始化领域服务（内部使用）
    resourceManager := resourceService.NewResourceManager(resourceRepo)
    
    // 3. 初始化应用服务（实现 driving 接口）
    resourceCommandService := resourceApp.NewResourceCommandService(resourceManager, resourceRepo)
    resourceQueryService := resourceApp.NewResourceQueryService(resourceRepo)
    
    // 4. 初始化 Handler（依赖 driving 接口）
    m.ResourceHandler = handler.NewResourceHandler(resourceCommandService, resourceQueryService)
    
    return nil
}
```

---

### 阶段 6-9：重复相同模式重构其他模块

#### 阶段 6：Role 模块（3小时）

- 定义 RoleCommander / RoleQueryer
- 创建 RoleManager 领域服务
- RoleCommandService / RoleQueryService 实现接口
- 更新 Handler 和 Assembler

#### 阶段 7：Policy 模块（4小时）

- 定义 PolicyCommander / PolicyQueryer
- 创建 PolicyManager 领域服务（版本管理、Casbin同步）
- PolicyCommandService / PolicyQueryService 实现接口
- 更新 Handler 和 Assembler

#### 阶段 8：Assignment 模块（4小时）

- 定义 AssignmentCommander / AssignmentQueryer
- 创建 AssignmentManager 领域服务（事务、Casbin分组）
- AssignmentCommandService / AssignmentQueryService 实现接口
- 更新 Handler 和 Assembler

#### 阶段 9：验证和测试（3小时）

- 编译验证
- 单元测试
- 集成测试

---

## 架构对比

### ❌ 错误的架构（之前的设计）

```
Handler -> Application Service -> Domain Service (直接注入) -> Repository
                                  ↑ 错误：领域服务不应该直接暴露
```

### ✅ 正确的架构

```
Handler -> Driving Interface (port/driving)
              ↓ (实现)
           Application Service (编排)
              ↓ (调用)
           Domain Service (封装业务规则)
              ↓ (调用)
           Driven Interface (port/driven)
              ↓ (实现)
           Repository (Infrastructure)
```

## 关键差异

| 维度 | 错误设计（V1） | 正确设计（V2） |
|------|---------------|---------------|
| **端口定义** | ❌ 缺少 driving 端口 | ✅ domain/port/driving 定义用例接口 |
| **CQRS** | ❌ 未分离 | ✅ Commander / Queryer 分离 |
| **领域服务** | ❌ 直接暴露给应用层 | ✅ 内部实现，被应用服务编排 |
| **应用服务** | ❌ 直接包含业务逻辑 | ✅ 实现 driving 接口，编排领域服务 |
| **Handler依赖** | ❌ 依赖具体应用服务 | ✅ 依赖 driving 接口 |
| **职责分离** | ❌ 不清晰 | ✅ 清晰的分层和职责 |

## 重构时间估算

| 阶段 | 模块 | 时间 |
|------|------|------|
| 1 | 清理错误架构 | 1h |
| 2 | Resource - Driving 端口 | 1.5h |
| 3 | Resource - 领域服务 | 1h |
| 4 | Resource - 应用服务 | 1.5h |
| 5 | Resource - 接口层和容器 | 1h |
| 6 | Role 模块 | 3h |
| 7 | Policy 模块 | 4h |
| 8 | Assignment 模块 | 4h |
| 9 | 验证和测试 | 3h |
| **总计** | | **20小时** |

## 参考代码（authn 模块）

参考文件：

- `domain/account/port/driving/service.go` - Driving 接口定义
- `domain/account/service/manager.go` - 领域服务实现
- `application/account/account_app_service.go` - 应用服务实现接口

## 下一步行动

1. ✅ **确认架构方案** - 等待确认
2. ⏸️ **开始阶段1** - 清理错误代码
3. ⏸️ **执行阶段2-9** - 按正确架构重构

---

**备注**：这是正确的 DDD + CQRS + Hexagonal Architecture 实现方式，完全遵循 authn 模块的成熟模式。
