# 授权中心 - 领域模型设计

> [返回授权中心文档](./README.md)

本文档详细介绍授权中心的领域模型设计，包括聚合根、实体、值对象和领域服务，深入阐述每个模型的职责和实现的领域知识。

---

## 目录

1. [领域概述](#1-领域概述)
2. [领域模型总览](#2-领域模型总览)
3. [Role 聚合根](#3-role-聚合根)
4. [Assignment 聚合根](#4-assignment-聚合根)
5. [Resource 聚合根](#5-resource-聚合根)
6. [Policy 聚合根](#6-policy-聚合根)
7. [领域服务](#7-领域服务)
8. [值对象](#8-值对象)
9. [仓储接口](#9-仓储接口)
10. [Casbin 集成](#10-casbin-集成)

---

## 1. 领域概述

授权中心（AuthZ）领域负责基于 RBAC 的权限控制，采用 Casbin 作为策略引擎，遵循 PAP-PRP-PDP-PEP 架构模式，实现域对象级的细粒度权限管理。

### 1.1 核心领域概念

- **Role（角色）**：权限的集合，是授权的基本单位
- **Assignment（赋权）**：用户/组与角色的绑定关系
- **Resource（资源）**：可被授权的域对象（如量表、表单、档案）
- **Policy（策略）**：角色对资源的操作权限规则
- **Version（版本）**：策略变更的版本号，用于缓存失效

### 1.2 领域边界

**本领域负责**：

- ✅ 角色定义和生命周期管理
- ✅ 用户/组到角色的赋权管理
- ✅ 资源目录维护和动作定义
- ✅ 策略规则的创建、删除和查询
- ✅ 策略版本管理和缓存失效通知
- ✅ 权限决策（PDP）

**本领域不负责**：

- ❌ 用户身份验证（由 Authn 模块负责）
- ❌ 业务对象的 CRUD（由业务模块负责）
- ❌ 审计日志记录（由审计模块负责）
- ❌ 资源所有权判断（由业务模块提供 Owner 信息）

### 1.3 RBAC 模型

本系统采用 **RBAC with Domain** 模型（Casbin RBAC 模型）：

```text
┌──────────────────────────────────────────────────────┐
│                  RBAC with Domain                     │
├──────────────────────────────────────────────────────┤
│                                                       │
│  Subject (主体)                                       │
│    user:1234567890  (用户)                           │
│    group:admin      (组)                             │
│                                                       │
│         │ g (grouping)                                │
│         ▼                                             │
│                                                       │
│  Role (角色)                                          │
│    role:admin       (管理员)                          │
│    role:therapist   (治疗师)                          │
│                                                       │
│         │ p (policy)                                  │
│         ▼                                             │
│                                                       │
│  Object (对象)        Action (动作)                   │
│    scale:form:*       read_all   (读取全部)           │
│    scale:form:*       read_own   (读取自己)           │
│    scale:form:*       update_own (更新自己)           │
│                                                       │
│  Domain (域)                                          │
│    tenant:org001    (租户/组织)                       │
│                                                       │
└──────────────────────────────────────────────────────┘
```

---

## 2. 领域模型总览

### 2.1 聚合根设计

```text
┌─────────────────────────────────────────────────────────────┐
│                  AuthZ Domain Model                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────┐         ┌──────────────────┐         │
│  │  Role (聚合根)    │         │ Assignment (聚合根)│         │
│  ├──────────────────┤         ├──────────────────┤         │
│  │ + ID             │   N:M   │ + ID             │         │
│  │ + Name           │◄────────┤ + SubjectType    │         │
│  │ + DisplayName    │         │ + SubjectID      │         │
│  │ + TenantID       │         │ + RoleID         │         │
│  │ + IsSystem       │         │ + TenantID       │         │
│  └──────────────────┘         │ + GrantedBy      │         │
│           │                    └──────────────────┘         │
│           │ 1:N                                             │
│           ▼                                                  │
│  ┌──────────────────────────────────────────────┐          │
│  │       Policy (领域服务)                       │          │
│  │  ┌──────────────────────────────────────┐   │          │
│  │  │ PolicyRule (值对象)                  │   │          │
│  │  │  - Sub (角色)                        │   │          │
│  │  │  - Dom (租户)                        │   │          │
│  │  │  - Obj (资源)                        │   │          │
│  │  │  - Act (动作)                        │   │          │
│  │  └──────────────────────────────────────┘   │          │
│  │                                              │          │
│  │  ┌──────────────────────────────────────┐   │          │
│  │  │ GroupingRule (值对象)                │   │          │
│  │  │  - Sub (用户/组)                     │   │          │
│  │  │  - Dom (租户)                        │   │          │
│  │  │  - Role (角色)                       │   │          │
│  │  └──────────────────────────────────────┘   │          │
│  └──────────────────────────────────────────────┘          │
│                                                              │
│  ┌──────────────────┐         ┌──────────────────┐         │
│  │ Resource (聚合根) │         │PolicyVersion(聚合根)│        │
│  ├──────────────────┤         ├──────────────────┤         │
│  │ + ID             │         │ + ID             │         │
│  │ + Key            │         │ + TenantID       │         │
│  │ + AppName        │         │ + Version        │         │
│  │ + Domain         │         │ + ChangedBy      │         │
│  │ + Type           │         │ + Reason         │         │
│  │ + Actions[]      │         └──────────────────┘         │
│  └──────────────────┘                                       │
│           │                                                  │
│           │ 验证                                             │
│           ▼                                                  │
│  ┌──────────────────────────────────────────────┐          │
│  │       ResourceValidator (领域服务)            │          │
│  │  - CheckKeyUnique                             │          │
│  │  - ValidateAction                             │          │
│  └──────────────────────────────────────────────┘          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 聚合根职责划分

| 聚合根/领域服务 | 核心职责 | 不变性维护 |
|----------------|---------|-----------|
| **Role** | 角色定义、权限集合管理 | 租户内角色名唯一、系统角色不可删除 |
| **Assignment** | 赋权关系管理、主体与角色绑定 | 同一主体在同租户下不能重复绑定同一角色 |
| **Resource** | 资源目录、动作定义 | 资源键全局唯一、动作必须在标准集合内 |
| **PolicyVersion** | 策略版本管理、缓存失效通知 | 版本号单调递增 |
| **Policy（领域服务）** | 策略规则生成、Casbin 规则管理 | 策略规则的有效性 |

### 2.3 PAP-PRP-PDP-PEP 架构

```text
┌─────────────────────────────────────────────────────┐
│                   Architecture                       │
└─────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────┐
│  PAP (Policy Administration Point)                   │
│  策略管理接口                                          │
│  - POST /authz/roles                                 │
│  - POST /authz/assignments/grant                     │
│  - POST /authz/policies                              │
└──────────────────────────────────────────────────────┘
           │
           │ 写入
           ▼
┌──────────────────────────────────────────────────────┐
│  PRP (Policy Retrieval Point)                        │
│  策略存储                                             │
│  - iam_authz_roles (角色表)                          │
│  - iam_authz_assignments (赋权表)                    │
│  - iam_authz_resources (资源目录)                    │
│  - iam_casbin_rule (Casbin 规则表)                   │
│  - iam_authz_policy_versions (版本表)                │
└──────────────────────────────────────────────────────┘
           │
           │ 读取
           ▼
┌──────────────────────────────────────────────────────┐
│  PDP (Policy Decision Point)                         │
│  决策引擎                                             │
│  - Casbin Enforcer                                   │
│  - Enforce(sub, dom, obj, act) → bool               │
└──────────────────────────────────────────────────────┘
           │
           │ 判决
           ▼
┌──────────────────────────────────────────────────────┐
│  PEP (Policy Enforcement Point)                      │
│  执行点 (业务服务)                                     │
│  - DomainGuard (Guard SDK)                           │
│  - 两段式判定: All → Own                              │
└──────────────────────────────────────────────────────┘
```

---

## 3. Role 聚合根

### 3.1 领域概念

**Role（角色）** 是权限的集合，是 RBAC 模型的核心概念。角色定义了一组权限，用户/组通过赋权（Assignment）获得角色，从而拥有相应的权限。

### 3.2 聚合根定义

```go
// internal/apiserver/domain/authz/role/role.go
package role

type Role struct {
    ID          meta.ID  // 角色ID
    Name        string   // 角色名称（标识符）
    DisplayName string   // 显示名称
    TenantID    string   // 租户ID（域）
    Description string   // 描述
}

// NewRole 创建新角色
func NewRole(name, displayName, tenantID string, 
    opts ...RoleOption) Role

// Key 返回 Casbin 中的角色标识
// 格式：role:<name>
func (r *Role) Key() string {
    return "role:" + r.Name
}
```

### 3.3 领域职责

| 职责类别 | 具体职责 | 实现的领域知识 |
|---------|---------|---------------|
| **标识生成** | 生成 Casbin 角色标识 | 格式 `role:<name>`，用于策略规则 |
| **租户隔离** | 角色绑定到租户 | 不同租户可以有同名角色但权限不同 |
| **显示名称** | 提供用户友好的名称 | Name 是标识符，DisplayName 是展示名称 |
| **系统角色** | 标记系统内置角色 | 系统角色不能删除 |

### 3.4 业务不变性

1. **唯一性约束**
   - ✅ 租户内角色名（Name）必须唯一
   - ✅ 角色 ID 全局唯一

2. **命名约束**
   - ✅ Name 不能为空，推荐使用 snake_case（如 `therapist`, `admin`）
   - ✅ DisplayName 不能为空（如 "治疗师", "管理员"）

3. **系统角色保护**
   - ✅ 系统角色（is_system=1）不能被删除
   - ✅ 系统角色不能修改名称

### 3.5 角色命名规范

```text
角色标识格式：role:<name>

示例：
- role:admin           (管理员)
- role:therapist       (治疗师)
- role:data_analyst    (数据分析师)
- role:guardian        (监护人)
```

---

## 4. Assignment 聚合根

### 4.1 领域概念

**Assignment（赋权）** 表示用户或组与角色的绑定关系。通过赋权，主体（Subject）获得角色的所有权限。

### 4.2 聚合根定义

```go
// internal/apiserver/domain/authz/assignment/assignment.go
package assignment

type Assignment struct {
    ID          AssignmentID  // 赋权ID
    SubjectType SubjectType   // 主体类型（user/group）
    SubjectID   string        // 主体ID
    RoleID      uint64        // 角色ID
    TenantID    string        // 租户ID（域）
    GrantedBy   string        // 授权操作人
}

// NewAssignment 创建新赋权
func NewAssignment(subjectType SubjectType, subjectID string, 
    roleID uint64, tenantID string, opts ...AssignmentOption) Assignment

// SubjectKey 返回 Casbin 中的主体标识
// 格式：<type>:<id>
func (a *Assignment) SubjectKey() string {
    return string(a.SubjectType) + ":" + a.SubjectID
}

// RoleKey 返回 Casbin 中的角色标识
// 格式：role:<role_id>
func (a *Assignment) RoleKey() string {
    id := meta.FromUint64(a.RoleID)
    return "role:" + id.String()
}

// SubjectType 主体类型
type SubjectType string

const (
    SubjectTypeUser  SubjectType = "user"   // 用户
    SubjectTypeGroup SubjectType = "group"  // 组
)
```

### 4.3 领域职责

| 职责类别 | 具体职责 | 实现的领域知识 |
|---------|---------|---------------|
| **绑定关系** | 主体与角色的关联 | 主体通过赋权获得角色权限 |
| **主体标识** | 生成 Casbin 主体标识 | 格式 `user:<id>` 或 `group:<id>` |
| **授权追溯** | 记录授权人 | 审计需求，记录谁授予的权限 |
| **租户隔离** | 赋权在租户内有效 | 跨租户需要重新赋权 |

### 4.4 业务不变性

1. **唯一性约束**
   - ✅ 同一主体在同一租户下不能重复绑定同一角色
   - ✅ 赋权 ID 全局唯一

2. **数据完整性**
   - ✅ SubjectType、SubjectID、RoleID、TenantID 都不能为空
   - ✅ RoleID 必须对应一个存在的角色

3. **删除级联**
   - ✅ 角色删除时，相关赋权记录应该被清理
   - ✅ 用户删除时，相关赋权记录应该被清理

### 4.5 赋权流程

```text
┌──────────────────────────────────────────────────────┐
│                  赋权流程                             │
└──────────────────────────────────────────────────────┘

    [管理员发起赋权]
         │
         ▼
    [创建 Assignment]
    - SubjectType: user
    - SubjectID: 1234567890
    - RoleID: 5678901234
    - TenantID: org001
         │
         ▼
    [生成 Casbin g 规则]
    g, user:1234567890, role:5678901234, org001
         │
         ▼
    [写入 casbin_rule 表]
    ptype=g, v0=user:1234567890, 
    v1=role:5678901234, v2=org001
         │
         ▼
    [策略版本++]
         │
         ▼
    [广播缓存失效通知]
    Redis Pub/Sub: authz:policy_changed
```

---

## 5. Resource 聚合根

### 5.1 领域概念

**Resource（资源）** 是授权的对象，表示系统中可以被授权访问的域对象类型。资源定义了允许的动作集合。

### 5.2 聚合根定义

```go
// internal/apiserver/domain/authz/resource/resource.go
package resource

type Resource struct {
    ID          ResourceID  // 资源ID
    Key         string      // 资源键（如 scale:form:*）
    DisplayName string      // 显示名称
    AppName     string      // 应用名称
    Domain      string      // 业务域
    Type        string      // 对象类型
    Actions     []string    // 允许的动作列表
    Description string      // 描述
}

// NewResource 创建新资源
func NewResource(key string, actions []string, 
    opts ...ResourceOption) Resource

// HasAction 检查资源是否包含指定动作
func (r *Resource) HasAction(action string) bool
```

### 5.3 领域职责

| 职责类别 | 具体职责 | 实现的领域知识 |
|---------|---------|---------------|
| **资源标识** | 定义资源的唯一键 | 格式 `<app>:<domain>:<type>:*` |
| **动作约束** | 定义资源支持的动作 | 策略规则的动作必须在此列表内 |
| **业务语义** | 表达资源的业务含义 | AppName、Domain、Type 三层结构 |
| **动作验证** | 验证动作的合法性 | 防止无效的策略规则 |

### 5.4 资源键格式

```text
格式：<app>:<domain>:<type>:*

示例：
- scale:form:*           (量表域-表单类型)
- scale:record:*         (量表域-记录类型)
- uc:user:*              (用户中心-用户类型)
- uc:child:*             (用户中心-儿童类型)
- file:document:*        (文件域-文档类型)

说明：
- app: 应用名称（如 scale、uc、file）
- domain: 业务域（如 form、user、document）
- type: 对象类型（与 domain 相同或更具体）
- *: 通配符，表示该类型的所有实例
```

### 5.5 标准动作集合

```go
// internal/apiserver/domain/authz/resource/action.go
package resource

var StandardActions = []Action{
    {Name: "create",      Scope: ScopeOwn},  // 创建
    {Name: "read_all",    Scope: ScopeAll},  // 读取(全部)
    {Name: "read_own",    Scope: ScopeOwn},  // 读取(自己)
    {Name: "update_all",  Scope: ScopeAll},  // 更新(全部)
    {Name: "update_own",  Scope: ScopeOwn},  // 更新(自己)
    {Name: "delete_all",  Scope: ScopeAll},  // 删除(全部)
    {Name: "delete_own",  Scope: ScopeOwn},  // 删除(自己)
    {Name: "approve",     Scope: ScopeAll},  // 审批
    {Name: "export",      Scope: ScopeAll},  // 导出
    {Name: "disable_all", Scope: ScopeAll},  // 禁用(全部)
}

// Scope 动作作用域
type Scope string

const (
    ScopeAll Scope = "all"  // 全局作用域
    ScopeOwn Scope = "own"  // 仅自己
)
```

### 5.6 业务不变性

1. **唯一性约束**
   - ✅ 资源键（Key）全局唯一
   - ✅ 资源 ID 全局唯一

2. **格式约束**
   - ✅ Key 必须符合 `<app>:<domain>:<type>:*` 格式
   - ✅ Actions 不能为空
   - ✅ Actions 中的动作必须在标准动作集合内

3. **语义约束**
   - ✅ AppName、Domain、Type 都不能为空
   - ✅ DisplayName 用于前端展示，不能为空

---

## 6. Policy 聚合根

### 6.1 领域概念

**Policy（策略）** 包含策略规则（PolicyRule）和分组规则（GroupingRule），是 Casbin 权限判定的核心数据。

### 6.2 策略规则（PolicyRule）

```go
// internal/apiserver/domain/authz/policy/rule.go
package policy

// PolicyRule 策略规则值对象（p 规则）
type PolicyRule struct {
    Sub string  // 主体（角色）
    Dom string  // 域（租户）
    Obj string  // 对象（资源）
    Act string  // 动作
}

// NewPolicyRule 创建策略规则
func NewPolicyRule(sub, dom, obj, act string) PolicyRule

// Casbin p 规则示例：
// p, role:admin, org001, scale:form:*, read_all
// p, role:therapist, org001, scale:form:*, read_own
// p, role:therapist, org001, scale:form:*, update_own
```

### 6.3 分组规则（GroupingRule）

```go
// GroupingRule 分组规则值对象（g 规则：用户/组 → 角色）
type GroupingRule struct {
    Sub  string  // 主体（用户/组）
    Dom  string  // 域（租户）
    Role string  // 角色
}

// NewGroupingRule 创建分组规则
func NewGroupingRule(sub, dom, role string) GroupingRule

// Casbin g 规则示例：
// g, user:1234567890, role:admin, org001
// g, user:9876543210, role:therapist, org001
// g, group:doctors, role:therapist, org001
```

### 6.4 策略版本（PolicyVersion）

```go
// internal/apiserver/domain/authz/policy/policy_version.go
package policy

type PolicyVersion struct {
    ID        PolicyVersionID  // 版本ID
    TenantID  string           // 租户ID
    Version   int64            // 版本号
    ChangedBy string           // 变更人
    Reason    string           // 变更原因
}

// NewPolicyVersion 创建新版本
func NewPolicyVersion(tenantID string, version int64, 
    opts ...PolicyVersionOption) PolicyVersion

// RedisKey 返回 Redis 中的版本键
func (pv *PolicyVersion) RedisKey() string {
    return "authz:policy_version:" + pv.TenantID
}

// PubSubChannel 返回发布订阅通道
func (pv *PolicyVersion) PubSubChannel() string {
    return "authz:policy_changed"
}
```

### 6.5 策略规则示例

```text
┌──────────────────────────────────────────────────────┐
│              策略规则示例 (Casbin)                    │
└──────────────────────────────────────────────────────┘

# g 规则（用户/组 → 角色）
g, user:1234567890, role:admin, org001
g, user:9876543210, role:therapist, org001
g, group:doctors, role:therapist, org001

# p 规则（角色 → 权限）
# 管理员：所有权限
p, role:admin, org001, scale:form:*, read_all
p, role:admin, org001, scale:form:*, update_all
p, role:admin, org001, scale:form:*, delete_all
p, role:admin, org001, scale:form:*, approve
p, role:admin, org001, scale:form:*, export

# 治疗师：部分权限
p, role:therapist, org001, scale:form:*, create
p, role:therapist, org001, scale:form:*, read_own
p, role:therapist, org001, scale:form:*, update_own
p, role:therapist, org001, scale:form:*, read_all

# 监护人：只读权限
p, role:guardian, org001, scale:record:*, read_own
```

### 6.6 策略版本管理

```text
┌──────────────────────────────────────────────────────┐
│                策略版本管理流程                        │
└──────────────────────────────────────────────────────┘

    [策略变更操作]
    - 创建/删除角色
    - 赋权/撤销赋权
    - 添加/删除策略规则
         │
         ▼
    [写入 Casbin 规则]
    更新 iam_casbin_rule 表
         │
         ▼
    [版本号递增]
    UPDATE iam_authz_policy_versions
    SET policy_version = policy_version + 1
    WHERE tenant_id = 'org001'
         │
         ▼
    [广播缓存失效通知]
    Redis Pub: authz:policy_changed
    Payload: {"tenant_id": "org001", "version": 123}
         │
         ▼
    [各业务服务订阅]
    Redis Sub: authz:policy_changed
         │
         ▼
    [Enforcer 缓存失效]
    enforcer.InvalidateCache()
         │
         ▼
    [下次权限判定时重新加载]
    enforcer.LoadPolicy()
```

---

## 7. 领域服务

### 7.1 ResourceValidator（资源验证器）

```go
// internal/apiserver/domain/authz/resource/validator.go
package resource

type validator struct {
    resourceRepo Repository
}

// NewValidator 创建资源验证器
func NewValidator(resourceRepo Repository) *validator

// CheckKeyUnique 检查资源键的唯一性
func (v *validator) CheckKeyUnique(ctx context.Context, 
    key string) error

// ValidateCreateCommand 验证创建命令
func (v *validator) ValidateCreateCommand(
    cmd CreateResourceCommand) error

// ValidateUpdateCommand 验证更新命令
func (v *validator) ValidateUpdateCommand(
    cmd UpdateResourceCommand) error

// ValidateCreateParameters 验证创建资源的参数
func (v *validator) ValidateCreateParameters(key string, 
    displayName string, appName string, domain string, 
    resourceType string, actions []string) error
```

**实现的领域知识**：

1. **唯一性验证**：资源键全局唯一
2. **格式验证**：Key、DisplayName、Actions 不能为空
3. **动作验证**：Actions 必须在标准动作集合内
4. **结构验证**：AppName、Domain、Type 都不能为空

### 7.2 PolicyService（策略服务）

策略服务负责将领域对象转换为 Casbin 规则，并管理策略版本。

**核心职责**：

1. **规则生成**：将 Assignment 转换为 g 规则
2. **规则删除**：删除对应的 p/g 规则
3. **版本管理**：策略变更时递增版本号
4. **缓存通知**：通过 Redis Pub/Sub 广播版本变更

---

## 8. 值对象

### 8.1 AssignmentID

```go
type AssignmentID meta.ID

func NewAssignmentID(value uint64) AssignmentID
func (id AssignmentID) Uint64() uint64
func (id AssignmentID) String() string
```

### 8.2 ResourceID

```go
type ResourceID idutil.ID

func NewResourceID(value uint64) ResourceID
func (id ResourceID) Uint64() uint64
func (id ResourceID) String() string
```

### 8.3 PolicyVersionID

```go
type PolicyVersionID meta.ID

func NewPolicyVersionID(value uint64) PolicyVersionID
func (id PolicyVersionID) Uint64() uint64
func (id PolicyVersionID) String() string
```

### 8.4 SubjectType

```go
type SubjectType string

const (
    SubjectTypeUser  SubjectType = "user"   // 用户
    SubjectTypeGroup SubjectType = "group"  // 组
)

func (st SubjectType) String() string
```

### 8.5 Action

```go
type Action struct {
    Name        string  // 动作名称（如 read_all, read_own）
    DisplayName string  // 显示名称
    Scope       Scope   // all/own，是否需要 owner 校验
}

type Scope string

const (
    ScopeAll Scope = "all"  // 全局作用域
    ScopeOwn Scope = "own"  // 仅自己
)
```

### 8.6 值对象特性

| 特性 | 说明 | 实现的领域知识 |
|-----|------|---------------|
| **不可变性** | 创建后不可修改 | 保证线程安全和语义清晰 |
| **值相等性** | 通过值而非引用比较 | 两个 ID 值相同即相等 |
| **自包含验证** | 验证逻辑封装在内 | 无效的值对象无法被创建 |
| **领域语义** | 表达领域概念 | SubjectType 比 string 更具语义 |

---

## 9. 仓储接口

### 9.1 Role 仓储

```go
// internal/apiserver/domain/authz/role/port/driven/repo.go
package driven

type Repository interface {
    Create(ctx context.Context, role *role.Role) error
    Update(ctx context.Context, role *role.Role) error
    Delete(ctx context.Context, id meta.ID) error
    FindByID(ctx context.Context, id meta.ID) (*role.Role, error)
    FindByName(ctx context.Context, tenantID, name string) (*role.Role, error)
    List(ctx context.Context, tenantID string, 
        offset, limit int) ([]*role.Role, int64, error)
}
```

### 9.2 Assignment 仓储

```go
// internal/apiserver/domain/authz/assignment/port/driven/repo.go
package driven

type Repository interface {
    Create(ctx context.Context, assignment *assignment.Assignment) error
    Delete(ctx context.Context, id assignment.AssignmentID) error
    FindByID(ctx context.Context, 
        id assignment.AssignmentID) (*assignment.Assignment, error)
    FindBySubject(ctx context.Context, subjectType assignment.SubjectType, 
        subjectID, tenantID string) ([]*assignment.Assignment, error)
    FindByRole(ctx context.Context, roleID uint64, 
        tenantID string) ([]*assignment.Assignment, error)
    Exists(ctx context.Context, subjectType assignment.SubjectType, 
        subjectID string, roleID uint64, tenantID string) (bool, error)
}
```

### 9.3 Resource 仓储

```go
// internal/apiserver/domain/authz/resource/repository.go
package resource

type Repository interface {
    Create(ctx context.Context, resource *Resource) error
    Update(ctx context.Context, resource *Resource) error
    Delete(ctx context.Context, id ResourceID) error
    FindByID(ctx context.Context, id ResourceID) (*Resource, error)
    FindByKey(ctx context.Context, key string) (*Resource, error)
    List(ctx context.Context, offset, limit int) ([]*Resource, int64, error)
}
```

### 9.4 PolicyVersion 仓储

```go
// internal/apiserver/domain/authz/policy/port/driven/repo.go
package driven

type Repository interface {
    Create(ctx context.Context, pv *policy.PolicyVersion) error
    GetCurrentVersion(ctx context.Context, 
        tenantID string) (*policy.PolicyVersion, error)
    IncrementVersion(ctx context.Context, tenantID, 
        changedBy, reason string) (*policy.PolicyVersion, error)
}
```

---

## 10. Casbin 集成

### 10.1 Casbin 模型配置

```ini
# configs/casbin_model.conf

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

### 10.2 Casbin 接口

```go
// internal/apiserver/domain/authz/policy/port/driven/casbin.go
package driven

type CasbinAdapter interface {
    // 策略规则
    AddPolicy(ctx context.Context, rule policy.PolicyRule) error
    RemovePolicy(ctx context.Context, rule policy.PolicyRule) error
    GetPoliciesByRole(ctx context.Context, roleKey, 
        domain string) ([]policy.PolicyRule, error)
    
    // 分组规则
    AddGroupingPolicy(ctx context.Context, 
        rule policy.GroupingRule) error
    RemoveGroupingPolicy(ctx context.Context, 
        rule policy.GroupingRule) error
    GetGroupingsBySubject(ctx context.Context, subjectKey, 
        domain string) ([]policy.GroupingRule, error)
    
    // 批量操作
    UpdatePolicies(ctx context.Context, 
        addRules, removeRules []policy.PolicyRule) error
}
```

### 10.3 权限判定流程

```text
┌──────────────────────────────────────────────────────┐
│                权限判定流程 (PEP)                     │
└──────────────────────────────────────────────────────┘

    [业务请求]
    用户 1234567890 要读取表单 form_001
         │
         ▼
    [提取上下文]
    - sub: user:1234567890
    - dom: org001 (租户)
    - obj: scale:form:*
    - act: read_all
         │
         ▼
    [第一次判定: read_all]
    enforcer.Enforce(
      "user:1234567890", 
      "org001", 
      "scale:form:*", 
      "read_all"
    )
         │
         ├──► [TRUE]  → 允许访问，返回全部数据
         │
         └──► [FALSE] → 第二次判定: read_own
                │
                ▼
             enforcer.Enforce(
               "user:1234567890", 
               "org001", 
               "scale:form:*", 
               "read_own"
             )
                │
                ├──► [TRUE]  → 允许访问，返回自己的数据
                │              需要业务层提供 OwnerID
                │
                └──► [FALSE] → 拒绝访问，返回 403
```

### 10.4 两段式判定（DomainGuard）

```go
// 使用示例（伪代码）
package handler

func (h *FormHandler) GetForm(c *gin.Context) {
    formID := c.Param("id")
    userID := getCurrentUserID(c)
    tenantID := getCurrentTenantID(c)
    
    // 使用 DomainGuard 进行权限判定
    guard := NewDomainGuard(enforcer, userID, tenantID)
    
    // 第一次判定：是否有 read_all 权限
    if guard.Can(ctx).Read("scale:form:*").All() {
        // 有全局读权限，返回任意表单
        form, _ := formRepo.GetByID(ctx, formID)
        c.JSON(200, form)
        return
    }
    
    // 第二次判定：是否有 read_own 权限
    if guard.Can(ctx).Read("scale:form:*").Own(ownerID) {
        // 只能读自己的，检查 Owner
        form, _ := formRepo.GetByIDAndOwner(ctx, formID, userID)
        if form != nil {
            c.JSON(200, form)
            return
        }
    }
    
    // 没有权限
    c.JSON(403, gin.H{"error": "forbidden"})
}
```

---

## 11. 总结

### 11.1 聚合根职责总结

| 聚合根/领域服务 | 核心职责 | 关键领域知识 |
|----------------|---------|------------|
| **Role** | 角色定义、权限集合管理 | 租户隔离、系统角色保护、Casbin 标识生成 |
| **Assignment** | 赋权关系管理、主体绑定 | 主体标识生成、g 规则生成、唯一性约束 |
| **Resource** | 资源目录、动作定义 | 资源键格式、标准动作集合、动作作用域 |
| **Policy** | 策略规则、版本管理 | p/g 规则生成、版本递增、缓存失效通知 |
| **ResourceValidator** | 资源验证、参数校验 | 唯一性验证、格式验证、动作验证 |

### 11.2 设计亮点

1. **RBAC with Domain**：支持多租户隔离，租户间权限独立
2. **两段式判定**：先判 `*_all`，再判 `*_own`，兼顾性能和灵活性
3. **资源目录约束**：所有策略规则的 obj/act 必须来自资源目录，防止无效规则
4. **策略版本管理**：版本号递增 + Redis Pub/Sub，实现分布式缓存失效
5. **Casbin 集成**：领域对象与 Casbin 规则解耦，便于测试和维护
6. **PAP-PRP-PDP-PEP 架构**：职责清晰，符合权限系统标准架构

### 11.3 核心流程

```text
┌─────────────────────────────────────────────────────┐
│               授权中心核心流程                        │
└─────────────────────────────────────────────────────┘

1. 创建角色
   Role → RoleRepository → iam_authz_roles

2. 定义策略
   PolicyRule → Casbin → iam_casbin_rule
   (p, role:admin, org001, scale:form:*, read_all)

3. 赋权用户
   Assignment → GroupingRule → iam_casbin_rule
   (g, user:1234567890, role:admin, org001)

4. 版本递增
   PolicyVersion++ → Redis Pub: authz:policy_changed

5. 权限判定
   Enforcer.Enforce(sub, dom, obj, act) → true/false
```

---

**最后更新**: 2025-11-20
**维护团队**: AuthZ Team
