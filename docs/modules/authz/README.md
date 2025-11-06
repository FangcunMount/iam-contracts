# AuthZ 模块架构文档 (V1)

## 概述

AuthZ 模块基于 RBAC 模型实现域对象级权限控制，采用 Casbin 作为策略引擎，遵循 PAP-PRP-PDP-PEP 架构模式。

## 整体架构

### 架构图

```text
┌─────────────────────────────────────────────────────────────────┐
│                    Client & Business Services                    │
│  ┌──────────────┐         ┌──────────────────────────────────┐  │
│  │   前端/调用方  │ ──────▶ │ 业务服务 (UseCase + PEP Guard)  │  │
│  └──────────────┘         └──────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    │                 │                 │
                    ▼                 ▼                 ▼
           ┌────────────────┐ ┌──────────────┐ ┌──────────────┐
           │  Authn (JWKS)  │ │ Casbin Cache │ │  Redis Pub   │
           │   JWT 验签     │ │  Enforcer    │ │     Sub      │
           └────────────────┘ └──────────────┘ └──────────────┘
                                      │                 ▲
                                      ▼                 │
           ┌────────────────────────────────────────────┼────┐
           │             AuthZ Module (PAP/PRP)         │    │
           │  ┌──────────────────────────────────────┐  │    │
           │  │  PAP 管理 API                        │  │    │
           │  │  - 角色管理                          │  │    │
           │  │  - 赋权管理 (g 规则)                 │  │    │
           │  │  - 策略管理 (p 规则)                 │  │    │
           │  │  - 资源目录管理                      │  │    │
           │  └──────────────────────────────────────┘  │    │
           │                     │                       │    │
           │                     ▼                       │    │
           │  ┌──────────────────────────────────────┐  │    │
           │  │  PRP (Policy Repository)            │  │    │
           │  │  - casbin_rule (权威策略表)         │  │    │
           │  │  - authz_roles (角色表)             │  │    │
           │  │  - authz_assignments (赋权表)       │  │    │
           │  │  - authz_resources (资源目录)       │  │    │
           │  │  - authz_policy_versions (版本)     │  │    │
           │  └──────────────────────────────────────┘  │    │
           │                     │                       │    │
           │                     ▼                       │    │
           │  ┌──────────────────────────────────────┐  │    │
           │  │  Version Management                  │  ├────┘
           │  │  - 策略变更 → version++              │  │
           │  │  - 广播通知 (Redis Pub/Sub)         │  │
           │  └──────────────────────────────────────┘  │
           └─────────────────────────────────────────────┘
```

### 组件职责

#### PAP (Policy Administration Point) - 策略管理面

- **职责**: 维护角色、赋权、资源目录，生成和校验策略
- **接口**:
  - `POST /authz/roles` - 创建/更新角色
  - `POST /authz/assignments` - 用户/组 ↔ 角色赋权
  - `POST /authz/policies` - 添加策略规则（p 规则）
  - `GET /authz/resources` - 获取资源目录
- **约束**: 所有 obj/act 必须来自资源目录

#### PRP (Policy Retrieval Point) - 策略存储

- **Casbin GORM Adapter**: 存储在 `casbin_rule` 表
- **领域表** (便于审计与管理):
  - `authz_roles`: 角色定义
  - `authz_assignments`: 赋权记录
  - `authz_resources`: 域对象资源目录
  - `authz_policy_versions`: 策略版本

#### PDP (Policy Decision Point) - 决策点

- **默认方案**: 嵌入式 `CachedEnforcer` 在业务服务内（最低延迟）
- **可选方案**: 提供轻量 `/v1/decide` REST API 给异构客户端
- **行为**: 布尔判定 + 返回 policy_version

#### PEP (Policy Enforcement Point) - 执行点

- **SDK 形式**: `DomainGuard` 封装
- **两段式判定**:
  1. 先判 `*_all` 权限
  2. 不通过则判 `*_own` 并校验 Owner
- **使用示例**:

  ```go
  if guard.Can(ctx).Read("scale:form:*").All() {
      return repo.Get(id)
  }
  if guard.Can(ctx).Read("scale:form:*").Own(ownerID) {
      return repo.GetIfOwner(id, uid)
  }
  return ErrForbidden
  ```

### 版本管理与缓存刷新

1. **策略变更**: PAP 操作后 → `policy_version++`
2. **广播通知**: 通过 Redis Pub/Sub 发送 `authz:policy_changed` 消息
3. **缓存失效**: 各业务服务的 Enforcer 收到消息后调用 `InvalidateCache()`
4. **本地缓存**: Enforcer 使用 LRU + 短 TTL 决策缓存（可选）

## 目录结构

```text
internal/apiserver/
├── domain/authz/                          # 领域层
│   ├── role/                              # 角色聚合
│   │   ├── role.go                        # 角色实体
│   │   └── port/
│   │       └── driven/
│   │           └── repo.go                # 角色仓储接口
│   │
│   ├── assignment/                        # 赋权聚合
│   │   ├── assignment.go                  # 赋权实体
│   │   └── port/
│   │       └── driven/
│   │           └── repo.go                # 赋权仓储接口
│   │
│   ├── resource/                          # 资源聚合
│   │   ├── resource.go                    # 资源实体
│   │   ├── action.go                      # 动作值对象
│   │   └── port/
│   │       └── driven/
│   │           └── repo.go                # 资源仓储接口
│   │
│   └── policy/                            # 策略聚合
│       ├── policy_version.go              # 策略版本实体
│       ├── rule.go                        # 策略规则值对象
│       └── port/
│           └── driven/
│               ├── repo.go                # 版本仓储接口
│               └── casbin.go              # Casbin 操作接口
│
├── application/authz/                     # 应用层（用例编排）
│   ├── role/
│   │   └── service.go                     # 角色管理服务
│   ├── assignment/
│   │   └── service.go                     # 赋权管理服务
│   ├── policy/
│   │   └── service.go                     # 策略管理服务
│   ├── resource/
│   │   └── service.go                     # 资源管理服务
│   └── version/
│       └── service.go                     # 版本管理服务
│
├── infra/                                 # 基础设施层（按技术栈划分）
│   ├── mysql/                             # MySQL 仓储实现
│   │   ├── role/
│   │   ├── assignment/
│   │   ├── resource/
│   │   └── policy/
│   ├── casbin/                            # Casbin 适配器与模型
│   │   ├── model.conf
│   │   └── adapter.go
│   └── redis/                             # Redis 发布/订阅
│       └── version_notifier.go
│
├── interface/                             # 接口层
│   ├── restful/                           # REST API
│   │   ├── router.go                      # 路由注册
│   │   ├── handler_pap.go                 # PAP 管理接口
│   │   ├── handler_pdp.go                 # PDP 决策接口（可选）
│   │   └── dto/                           # 数据传输对象
│   │       ├── role.go
│   │       ├── assignment.go
│   │       ├── policy.go
│   │       └── resource.go
│   │
│   └── sdk/                               # SDK
│       └── go/
│           └── pep/                       # PEP 执行点
│               ├── guard.go               # DomainGuard 封装
│               ├── context.go             # 上下文提取
│               └── middleware.go          # 中间件（可选）
│
└── docs/                                  # 文档
    ├── README.md                          # 使用说明
    ├── architecture.md                    # 架构文档（本文件）
    ├── resources.seed.yaml                # 资源目录 seed 数据
    └── policy_init.csv                    # 初始策略示例
```

## 关键设计决策

### 1. 领域模型设计

参照 `authn` 模块的实践：

- 每个聚合独立目录（role, assignment, resource, policy）
- 领域对象与持久化对象分离
- 使用 Port/Adapter 模式定义仓储接口
- 基础设施层通过 Mapper 实现 BO ↔ PO 转换

### 2. Casbin 模型 (V1)

```text
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

**特点**:

- 纯等值匹配，无自定义函数
- 支持租户隔离（domain）
- 支持角色继承（g 规则）

### 3. 资源定义 (V1)

**格式**: `<app>:<domain>:<type>:*`

**示例**:

- `scale:form:*` - 量表表单
- `scale:report:*` - 量表报告
- `ops:user:*` - 运营用户管理

**动作枚举**:

- `create` - 创建（own）
- `read_all` / `read_own` - 读取
- `update_all` / `update_own` - 更新
- `delete_all` / `delete_own` - 删除
- `approve` - 审批
- `export` - 导出
- `disable_all` - 禁用

### 4. 两段式权限判定

```go
// 伪代码
func (uc *FormUseCase) GetForm(ctx context.Context, id uint64) (*Form, error) {
    // 第一段：判断全局权限
    if guard.Can(ctx).Read("scale:form:*").All() {
        return uc.repo.FindByID(ctx, id)
    }
    
    // 第二段：判断所有者权限
    userID := auth.GetUserID(ctx)
    if guard.Can(ctx).Read("scale:form:*").Own(userID) {
        form, err := uc.repo.FindByID(ctx, id)
        if err != nil {
            return nil, err
        }
        if form.OwnerID != userID {
            return nil, ErrForbidden
        }
        return form, nil
    }
    
    return nil, ErrForbidden
}
```

## 数据库 Schema

### authz_roles

```sql
CREATE TABLE authz_roles (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL,
  display_name VARCHAR(128),
  tenant_id VARCHAR(64) NOT NULL,
  description VARCHAR(512),
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  created_by BIGINT UNSIGNED,
  updated_by BIGINT UNSIGNED,
  deleted_by BIGINT UNSIGNED,
  version INT UNSIGNED NOT NULL DEFAULT 1,
  UNIQUE KEY uk_tenant_name (tenant_id, name),
  KEY idx_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### authz_assignments

```sql
CREATE TABLE authz_assignments (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  subject_type VARCHAR(16) NOT NULL,  -- user/group
  subject_id VARCHAR(64) NOT NULL,
  role_id BIGINT UNSIGNED NOT NULL,
  tenant_id VARCHAR(64) NOT NULL,
  granted_by VARCHAR(64),
  granted_at DATETIME,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  created_by BIGINT UNSIGNED,
  updated_by BIGINT UNSIGNED,
  deleted_by BIGINT UNSIGNED,
  version INT UNSIGNED NOT NULL DEFAULT 1,
  KEY idx_subject (subject_type, subject_id),
  KEY idx_role (role_id),
  KEY idx_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### authz_resources

```sql
CREATE TABLE authz_resources (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  `key` VARCHAR(128) NOT NULL UNIQUE,  -- scale:form:*
  display_name VARCHAR(128),
  app_name VARCHAR(32),
  domain VARCHAR(32),
  type VARCHAR(32),
  actions TEXT,  -- JSON array: ["create","read_all","read_own",...]
  description VARCHAR(512),
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  created_by BIGINT UNSIGNED,
  updated_by BIGINT UNSIGNED,
  deleted_by BIGINT UNSIGNED,
  version INT UNSIGNED NOT NULL DEFAULT 1,
  KEY idx_app (app_name),
  KEY idx_domain (domain),
  KEY idx_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### authz_policy_versions

```sql
CREATE TABLE authz_policy_versions (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tenant_id VARCHAR(64) NOT NULL UNIQUE,
  version BIGINT NOT NULL,
  changed_by VARCHAR(64),
  reason VARCHAR(512),
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  created_by BIGINT UNSIGNED,
  updated_by BIGINT UNSIGNED,
  deleted_by BIGINT UNSIGNED
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### casbin_rule

由 Casbin GORM Adapter 自动管理，存储 p 和 g 规则。

## 初始化数据

### 资源目录 (docs/resources.seed.yaml)

```yaml
resources:
  - key: "scale:form:*"
    display_name: "量表表单"
    app_name: "scale"
    domain: "form"
    type: "*"
    actions: 
      - create
      - read_all
      - read_own
      - update_all
      - update_own
      - delete_all
      - delete_own
      - approve
      - export
    description: "量表表单资源"
    
  - key: "scale:report:*"
    display_name: "量表报告"
    app_name: "scale"
    domain: "report"
    type: "*"
    actions:
      - read_all
      - export
    description: "量表报告资源"
    
  - key: "ops:user:*"
    display_name: "用户管理"
    app_name: "ops"
    domain: "user"
    type: "*"
    actions:
      - read_all
      - update_all
      - disable_all
    description: "运营用户管理资源"
```

### 策略示例 (docs/policy_init.csv)

```csv
# 角色 → 域对象 → 动作（租户 t1）
p,role:scale-editor,t1,scale:form:*,create
p,role:scale-editor,t1,scale:form:*,read_own
p,role:scale-editor,t1,scale:form:*,update_own
p,role:scale-reviewer,t1,scale:form:*,read_all
p,role:scale-reviewer,t1,scale:form:*,approve

# 用户授权
g,user:1001,role:scale-editor,t1
g,user:2002,role:scale-reviewer,t1
```

## API 示例

### 创建角色

```http
POST /authz/roles
Content-Type: application/json

{
  "name": "scale-editor",
  "display_name": "量表编辑员",
  "tenant_id": "t1",
  "description": "可创建和编辑自己的量表"
}
```

### 创建赋权

```http
POST /authz/assignments
Content-Type: application/json

{
  "subject_type": "user",
  "subject_id": "1001",
  "role_id": 1,
  "tenant_id": "t1",
  "granted_by": "admin"
}
```

### 添加策略

```http
POST /authz/policies
Content-Type: application/json

{
  "role": "role:scale-editor",
  "tenant_id": "t1",
  "policies": [
    {
      "object": "scale:form:*",
      "action": "create"
    },
    {
      "object": "scale:form:*",
      "action": "read_own"
    }
  ]
}
```

### 获取资源目录

```http
GET /authz/resources?app_name=scale
```

### 决策接口（可选）

```http
POST /authz/decide
Content-Type: application/json

{
  "subject": "user:1001",
  "domain": "t1",
  "object": "scale:form:*",
  "action": "read_own"
}

# Response
{
  "allowed": true,
  "policy_version": 5
}
```

## 使用指南

### 1. 初始化

```go
// 初始化 Casbin
casbinAdapter, err := casbin.NewCasbinAdapter(
    db, 
    "internal/apiserver/infra/casbin/model.conf",
)

// 初始化仓储
roleRepo := role.NewRoleRepository(db)
assignmentRepo := assignment.NewAssignmentRepository(db)
resourceRepo := resource.NewResourceRepository(db)
policyVersionRepo := policy.NewPolicyVersionRepository(db)

// 初始化应用服务
roleService := role.NewRoleService(roleRepo, casbinAdapter)
assignmentService := assignment.NewAssignmentService(assignmentRepo, casbinAdapter)
policyService := policy.NewPolicyService(casbinAdapter, resourceRepo, policyVersionRepo)
```

### 2. 在业务服务中使用 Guard

```go
import (
    "github.com/FangcunMount/iam-contracts/internal/apiserver/interface/authz/sdk/go/pep"
)

type FormUseCase struct {
    repo  FormRepository
    guard *pep.DomainGuard
}

func (uc *FormUseCase) GetForm(ctx context.Context, id uint64) (*Form, error) {
    // 两段式权限判定
    if uc.guard.Can(ctx).Read("scale:form:*").All() {
        return uc.repo.FindByID(ctx, id)
    }
    
    userID := auth.GetUserID(ctx)
    if uc.guard.Can(ctx).Read("scale:form:*").Own(userID) {
        form, err := uc.repo.FindByID(ctx, id)
        if err != nil {
            return nil, err
        }
        if form.OwnerID != userID {
            return nil, ErrForbidden
        }
        return form, nil
    }
    
    return nil, ErrForbidden
}
```

### 3. 监听策略变更

```go
// Redis Pub/Sub 监听
subscriber := redis.NewPolicyVersionSubscriber(redisClient)
subscriber.Subscribe(ctx, func(tenantID string, version int64) {
    // 清除本地 Enforcer 缓存
    casbinAdapter.InvalidateCache()
    log.Info("Policy cache invalidated", "tenant", tenantID, "version", version)
})
```

## V2 规划

V1 之后可以考虑的增强功能：

1. **菜单/前端路由管理**: 维护前端可见的菜单树
2. **API 路由注册与扫描**: 自动扫描后端 API 并关联权限
3. **ABAC 增强**: 支持属性条件判断（如时间、IP 等）
4. **审计增强**: 完整的操作审计流水和访问日志
5. **策略模拟**: 提供策略生效前的模拟测试工具
6. **策略冲突检测**: 自动检测冲突或冗余策略
7. **数据权限**: 支持字段级、行级数据过滤

## 注意事项

1. **性能优化**:
   - 使用 CachedEnforcer 减少数据库查询
   - 合理设置缓存 TTL 和 LRU 大小
   - 监控决策延迟

2. **安全考虑**:
   - 所有策略变更需记录审计日志
   - 敏感操作需要二次确认
   - 定期审查权限分配

3. **测试策略**:
   - 单元测试覆盖领域逻辑
   - 集成测试验证 Casbin 规则
   - E2E 测试验证完整权限流程

4. **运维建议**:
   - 监控 policy_version 变更频率
   - 定期备份 casbin_rule 表
   - 监控 Redis Pub/Sub 健康状态

## 参考资料

- [Casbin 官方文档](https://casbin.org/)
- [RBAC 权限模型](https://en.wikipedia.org/wiki/Role-based_access_control)
- [XACML 架构](https://en.wikipedia.org/wiki/XACML)
- [项目 authn 模块实现](/internal/apiserver/application/authn/)
