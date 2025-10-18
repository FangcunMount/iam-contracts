# AuthZ 模块目录树

## 完整目录结构

```text
internal/apiserver/modules/authz/
│
├── domain/                                      # 🏛️ 领域层（核心业务逻辑）
│   │
│   ├── role/                                    # 角色聚合根
│   │   ├── role.go                              # 角色实体 + 值对象
│   │   └── port/driven/
│   │       └── repo.go                          # 角色仓储接口
│   │
│   ├── assignment/                              # 赋权聚合根
│   │   ├── assignment.go                        # 赋权实体 + 值对象
│   │   └── port/driven/
│   │       └── repo.go                          # 赋权仓储接口
│   │
│   ├── resource/                                # 资源聚合根
│   │   ├── resource.go                          # 资源实体
│   │   ├── action.go                            # 动作值对象（预定义枚举）
│   │   └── port/driven/
│   │       └── repo.go                          # 资源仓储接口
│   │
│   └── policy/                                  # 策略聚合根
│       ├── policy_version.go                    # 策略版本实体
│       ├── rule.go                              # 策略规则值对象（p/g 规则）
│       └── port/driven/
│           ├── repo.go                          # 版本仓储接口
│           └── casbin.go                        # Casbin 操作接口
│
├── application/                                 # 🎯 应用层（用例编排 PAP）
│   ├── role/
│   │   └── service.go                           # 角色管理服务
│   ├── assignment/
│   │   └── service.go                           # 赋权管理服务（g 规则）
│   ├── policy/
│   │   └── service.go                           # 策略管理服务（p 规则）
│   ├── resource/
│   │   └── service.go                           # 资源目录管理服务
│   └── version/
│       └── service.go                           # 版本管理 + 广播服务
│
├── infra/                                       # 🔧 基础设施层（PRP + 技术实现）
│   │
│   ├── mysql/                                   # MySQL 持久化实现
│   │   ├── role/
│   │   │   ├── po.go                            # RolePO 持久化对象
│   │   │   ├── mapper.go                        # BO ↔ PO 转换器
│   │   │   └── repo.go                          # RoleRepository 实现
│   │   │
│   │   ├── assignment/
│   │   │   ├── po.go                            # AssignmentPO
│   │   │   ├── mapper.go
│   │   │   └── repo.go                          # AssignmentRepository 实现
│   │   │
│   │   ├── resource/
│   │   │   ├── po.go                            # ResourcePO
│   │   │   ├── mapper.go
│   │   │   └── repo.go                          # ResourceRepository 实现
│   │   │
│   │   └── policy/
│   │       ├── po.go                            # PolicyVersionPO
│   │       ├── mapper.go
│   │       └── repo.go                          # PolicyVersionRepository 实现
│   │
│   ├── casbin/                                  # Casbin 策略引擎封装（PDP）
│   │   ├── model.conf                           # RBAC 模型定义
│   │   ├── adapter.go                           # CasbinAdapter 实现（PRP 写入）
│   │   └── enforcer.go                          # CachedEnforcer 封装（PDP 决策）
│   │
│   └── redis/                                   # Redis 实现
│       └── version_pubsub.go                    # 策略版本发布/订阅
│
├── interface/                                   # 🌐 接口层（对外暴露）
│   │
│   ├── restful/                                 # REST API（PAP 管理接口）
│   │   ├── router.go                            # 路由注册
│   │   ├── handler_pap.go                       # PAP 管理 Handler
│   │   │                                        #  - 角色 CRUD
│   │   │                                        #  - 赋权管理
│   │   │                                        #  - 策略管理
│   │   │                                        #  - 资源目录查询
│   │   ├── handler_pdp.go                       # PDP 决策接口（可选）
│   │   │                                        #  - /decide 端点
│   │   └── dto/                                 # 数据传输对象
│   │       ├── role.go
│   │       ├── assignment.go
│   │       ├── policy.go
│   │       └── resource.go
│   │
│   └── sdk/                                     # SDK 封装
│       └── go/pep/                              # PEP 执行点 SDK
│           ├── guard.go                         # DomainGuard 核心封装
│           │                                    #  - Can().Read().All()
│           │                                    #  - Can().Read().Own(ownerID)
│           ├── context.go                       # 从上下文提取用户信息
│           └── middleware.go                    # Gin 中间件（可选）
│
└── docs/                                        # 📚 文档
    ├── README.md                                # 完整架构文档（本文件）
    ├── resources.seed.yaml                      # 资源目录 seed 数据
    └── policy_init.csv                          # 初始策略示例
```

## 数据库表

```text
authz_roles              # 角色定义表
authz_assignments        # 用户/组 ↔ 角色赋权表
authz_resources          # 域对象资源目录表
authz_policy_versions    # 策略版本表
casbin_rule              # Casbin 权威策略表（自动管理）
```

## 设计模式对照

| 层次 | 模式 | 说明 |
|-----|------|------|
| **领域层** | DDD 聚合根 | role, assignment, resource, policy 四个聚合 |
| **端口层** | 六边形架构 | port/driven 定义仓储和 Casbin 接口 |
| **基础设施** | Adapter 模式 | MySQL 和 Casbin 实现具体适配器 |
| **应用层** | 用例编排 | PAP 策略管理服务 |
| **接口层** | API + SDK | REST API (PAP) + Go SDK (PEP) |

## 架构分层对照 (XACML 模式)

```text
┌─────────────────────────────────────────────┐
│  PEP (Policy Enforcement Point)            │  ← interface/sdk/go/pep/
│  执行点：DomainGuard 两段式权限判定        │
└─────────────────────────────────────────────┘
                    ↓ Enforce
┌─────────────────────────────────────────────┐
│  PDP (Policy Decision Point)                │  ← infra/casbin/enforcer.go
│  决策点：CachedEnforcer 布尔判定           │
└─────────────────────────────────────────────┘
                    ↓ Query
┌─────────────────────────────────────────────┐
│  PRP (Policy Retrieval Point)               │  ← infra/mysql/ + casbin_rule
│  存储点：MySQL 领域表 + Casbin 策略表      │
└─────────────────────────────────────────────┘
                    ↑ Write
┌─────────────────────────────────────────────┐
│  PAP (Policy Administration Point)          │  ← application/ + interface/restful/
│  管理点：角色/赋权/策略/资源管理 API       │
└─────────────────────────────────────────────┘
```

## 关键文件说明

### 领域层（Domain）

- **role/role.go**: 角色实体，包含 `RoleID` 值对象和 `Key()` 方法生成 Casbin 角色标识
- **assignment/assignment.go**: 赋权实体，包含 `SubjectKey()` 和 `RoleKey()` 生成 g 规则标识
- **resource/resource.go**: 资源实体，定义域对象类型（如 `scale:form:*`）
- **resource/action.go**: 动作值对象，预定义标准动作枚举（create, read_all, read_own 等）
- **policy/rule.go**: 策略规则值对象（PolicyRule 和 GroupingRule）
- **policy/policy_version.go**: 策略版本实体，用于缓存失效通知

### 基础设施层（Infra）

- **mysql/*/po.go**: 持久化对象（PO），对应数据库表结构
- **mysql/*/mapper.go**: BO ↔ PO 转换器，实现领域对象与持久化对象互转
- **mysql/*/repo.go**: Repository 实现，继承 `BaseRepository` 提供 CRUD 操作
- **casbin/model.conf**: Casbin RBAC 模型定义，纯等值匹配，支持域隔离
- **casbin/adapter.go**: Casbin 适配器封装，实现 `CasbinPort` 接口

### 应用层（Application）

- **role/service.go**: 角色管理服务（创建、更新、删除、查询角色）
- **assignment/service.go**: 赋权管理服务（添加/撤销 g 规则）
- **policy/service.go**: 策略管理服务（添加/删除 p 规则，校验资源和动作）
- **resource/service.go**: 资源目录服务（维护域对象资源清单）
- **version/service.go**: 版本管理服务（递增版本号并广播通知）

### 接口层（Interface）

- **restful/handler_pap.go**: PAP 管理 API Handler（角色/赋权/策略/资源的 CRUD 接口）
- **sdk/go/pep/guard.go**: DomainGuard 核心实现，提供流式 API 进行权限判定

  ```go
  guard.Can(ctx).Read("scale:form:*").All()      // 判断全局读权限
  guard.Can(ctx).Read("scale:form:*").Own(uid)   // 判断所有者读权限
  ```

### 文档（Docs）

- **README.md**: 完整架构文档（架构图、设计决策、API 示例、使用指南）
- **resources.seed.yaml**: 资源目录初始化数据（定义所有域对象及其允许的动作）
- **policy_init.csv**: 策略示例数据（p 规则和 g 规则的初始配置）

## 工作流程

### 策略管理流程（PAP）

1. **创建角色**: `POST /authz/roles` → `application/role/service.go` → `infra/mysql/role/repo.go`
2. **添加策略**: `POST /authz/policies` → `application/policy/service.go` → `infra/casbin/adapter.go` → `casbin_rule` 表
3. **用户赋权**: `POST /authz/assignments` → `application/assignment/service.go` → `infra/casbin/adapter.go` (g 规则)
4. **版本递增**: 策略变更 → `application/version/service.go` → `policy_version++` → Redis Pub/Sub 广播

### 权限判定流程（PEP → PDP）

```go
// 业务服务用例中
func (uc *FormUseCase) GetForm(ctx context.Context, id uint64) (*Form, error) {
    // 1. PEP 发起判定
    if uc.guard.Can(ctx).Read("scale:form:*").All() {
        // 2. PDP (Enforcer) 查询 PRP (casbin_rule)
        // 3. 命中缓存或从数据库加载
        // 4. 返回判定结果：允许
        return uc.repo.FindByID(ctx, id)
    }
    
    // 5. 全局权限不通过，尝试所有者权限
    userID := auth.GetUserID(ctx)
    if uc.guard.Can(ctx).Read("scale:form:*").Own(userID) {
        form, _ := uc.repo.FindByID(ctx, id)
        if form.OwnerID == userID {
            return form, nil
        }
    }
    
    return nil, ErrForbidden
}
```

### 缓存刷新流程

1. **策略变更**: PAP 修改策略 → `policy_version` 表递增
2. **广播通知**: `version/service.go` → Redis Pub `authz:policy_changed` 消息
3. **订阅接收**: 各业务服务 Redis Sub 收到通知
4. **清除缓存**: 调用 `CasbinAdapter.InvalidateCache()` 清除 Enforcer 缓存
5. **下次查询**: 重新从 `casbin_rule` 加载最新策略

## 依赖关系

```text
interface/restful/    →  application/         →  domain/
                         infra/mysql/         →  domain/port/driven/
interface/sdk/pep/    →  infra/casbin/        →  domain/policy/port/driven/
                         infra/redis/
```

- **interface 层**依赖 **application 层**和 **infra 层**
- **application 层**依赖 **domain 层端口**
- **infra 层**实现 **domain 层端口**接口
- **domain 层**零依赖（纯业务逻辑）

## V1 关键约束

1. ✅ 资源仅限"域对象类型"（如 `scale:form:*`），无实例级权限
2. ✅ 动作范围编码进动作名（`read_all` / `read_own`），避免 ABAC 复杂性
3. ✅ 所有者判断在业务服务内完成（两段式判定）
4. ✅ 不包含菜单/前端路由管理
5. ✅ 不包含 API 路由自动扫描与注册

## 后续演进路径

- **V1.1**: 添加审计日志（策略变更记录、判定失败采样）
- **V1.2**: 提供轻量 PDP 决策服务（`/v1/decide` REST API）
- **V2.0**: 支持 ABAC（属性条件判断）、菜单管理、API 扫描

---

**参考**: 本目录树遵循 `authn` 模块的六边形架构风格，领域对象与持久化对象分离，使用 Port/Adapter 模式实现依赖倒置。
