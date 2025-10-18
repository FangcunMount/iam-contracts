# Authz 模块领域层重构 Todolist

## 问题分析

当前 authz 模块将大量领域知识放到了应用服务层，违反了 DDD 分层原则：
- ✅ authn 模块: 应用层 → 领域服务 → 领域对象
- ❌ authz 模块: 应用层 (包含领域逻辑) → 领域对象

## 重构目标

将领域逻辑从应用层下沉到领域服务层，使应用层成为薄薄的编排层。

---

## Phase 1: 创建领域服务结构

### 1.1 创建角色领域服务

- [ ] 创建 `domain/role/service/` 目录
- [ ] 创建 `RoleManager` 领域服务
  - 职责：角色的创建、更新、删除、查询
  - 包含业务规则：
    - 角色名称唯一性校验（同租户内）
    - 角色的业务逻辑验证
    - 角色键值生成规则

### 1.2 创建角色分配领域服务

- [ ] 创建 `domain/assignment/service/` 目录
- [ ] 创建 `AssignmentManager` 领域服务
  - 职责：角色分配的授予和撤销
  - 包含业务规则：
    - 分配的一致性保证（数据库 + Casbin 双写）
    - 分配的事务性处理（回滚机制）
    - 主体标识符生成规则（SubjectKey, RoleKey）

### 1.3 创建策略领域服务

- [ ] 创建 `domain/policy/service/` 目录
- [ ] 创建 `PolicyManager` 领域服务
  - 职责：策略规则的管理和版本控制
  - 包含业务规则：
    - 策略规则的构建（Role + Resource + Action → Casbin Rule）
    - 策略版本的递增逻辑
    - 策略变更的通知机制
    - 策略的一致性保证（Casbin + 版本 + 通知）

### 1.4 创建资源领域服务

- [ ] 创建 `domain/resource/service/` 目录
- [ ] 创建 `ResourceManager` 领域服务
  - 职责：资源目录的管理和验证
  - 包含业务规则：
    - 资源键的唯一性校验
    - 资源动作的验证逻辑
    - 资源类型的业务规则

---

## Phase 2: 重构角色模块

### 2.1 创建 RoleManager 领域服务

- [ ] 定义 `RoleManager` 结构

  ```go
  type RoleManager struct {
      roleRepo driven.RoleRepo
  }
  ```

- [ ] 实现 `CreateRole(ctx, name, displayName, tenantID, opts) (*Role, error)`
  - 包含名称唯一性检查
  - 领域对象创建
  - 持久化
- [ ] 实现 `UpdateRole(ctx, roleID, opts) (*Role, error)`
  - 获取角色
  - 更新字段
  - 持久化
- [ ] 实现 `DeleteRole(ctx, roleID, tenantID) error`
  - 租户隔离检查
  - 删除操作
- [ ] 实现 `GetRole(ctx, roleID, tenantID) (*Role, error)`
  - 租户隔离检查
  - 查询操作
- [ ] 实现 `ListRoles(ctx, query) (ListResult, error)`
  - 分页查询

### 2.2 重构角色应用服务

- [ ] 简化 `application/role/Service`
- [ ] 注入 `RoleManager` 领域服务
- [ ] 将业务逻辑委托给领域服务
- [ ] 保留应用层职责：
  - 命令对象到领域方法的转换
  - 事务边界管理（如果需要）
  - 跨领域服务编排（如果需要）

---

## Phase 3: 重构角色分配模块

### 3.1 创建 AssignmentManager 领域服务

- [ ] 定义 `AssignmentManager` 结构

  ```go
  type AssignmentManager struct {
      assignmentRepo driven.AssignmentRepo
      roleRepo       driven.RoleRepo
      casbinPort     policyDriven.CasbinPort
  }
  ```

- [ ] 实现 `GrantRole(ctx, subjectType, subjectID, roleID, tenantID, grantedBy) (*Assignment, error)`
  - 角色存在性和租户隔离检查
  - 创建分配领域对象
  - 构建 Casbin 分组规则
  - 双写（数据库 + Casbin）+ 回滚机制
- [ ] 实现 `RevokeRole(ctx, subjectType, subjectID, roleID, tenantID) error`
  - 查找分配记录
  - 构建 Casbin 规则
  - 双删（Casbin + 数据库）+ 回滚机制
- [ ] 实现 `RevokeByID(ctx, assignmentID, tenantID) error`
  - 租户隔离检查
  - 双删 + 回滚
- [ ] 实现查询方法

### 3.2 重构角色分配应用服务

- [ ] 简化 `application/assignment/Service`
- [ ] 注入 `AssignmentManager` 领域服务
- [ ] 将业务逻辑委托给领域服务

---

## Phase 4: 重构策略模块

### 4.1 创建 PolicyManager 领域服务

- [ ] 定义 `PolicyManager` 结构

  ```go
  type PolicyManager struct {
      policyVersionRepo policyDriven.PolicyVersionRepo
      roleRepo          roleDriven.RoleRepo
      resourceRepo      resourceDriven.ResourceRepo
      casbinPort        policyDriven.CasbinPort
      versionNotifier   policyDriven.VersionNotifier
  }
  ```

- [ ] 实现 `AddPolicyRule(ctx, roleID, resourceID, action, tenantID, changedBy, reason) error`
  - 角色和资源的存在性检查
  - 租户隔离检查
  - 资源动作验证
  - 策略规则构建
  - Casbin 添加
  - 版本递增 + 通知（幂等处理错误）
- [ ] 实现 `RemovePolicyRule(ctx, roleID, resourceID, action, tenantID, changedBy, reason) error`
  - 同上逻辑
- [ ] 实现查询方法

### 4.2 重构策略应用服务

- [ ] 简化 `application/policy/Service`
- [ ] 注入 `PolicyManager` 领域服务
- [ ] 将业务逻辑委托给领域服务

---

## Phase 5: 重构资源模块

### 5.1 创建 ResourceManager 领域服务

- [ ] 定义 `ResourceManager` 结构

  ```go
  type ResourceManager struct {
      resourceRepo resourceDriven.ResourceRepo
  }
  ```

- [ ] 实现 `CreateResource(ctx, key, actions, opts) (*Resource, error)`
  - 资源键唯一性检查
  - 领域对象创建
  - 持久化
- [ ] 实现 `UpdateResource(ctx, resourceID, opts) (*Resource, error)`
- [ ] 实现 `DeleteResource(ctx, resourceID) error`
- [ ] 实现 `ValidateAction(ctx, resourceKey, action) (bool, error)`
  - 资源动作验证逻辑
- [ ] 实现查询方法

### 5.2 重构资源应用服务

- [ ] 简化 `application/resource/Service`
- [ ] 注入 `ResourceManager` 领域服务
- [ ] 将业务逻辑委托给领域服务

---

## Phase 6: 更新 Assembler 和测试

### 6.1 更新 AuthzModule Assembler

- [ ] 创建领域服务实例

  ```go
  roleManager := roleService.NewRoleManager(roleRepository)
  assignmentManager := assignmentService.NewAssignmentManager(assignmentRepository, roleRepository, casbinAdapter)
  policyManager := policyService.NewPolicyManager(policyVersionRepository, roleRepository, resourceRepository, casbinAdapter, versionNotifier)
  resourceManager := resourceService.NewResourceManager(resourceRepository)
  ```

- [ ] 注入到应用服务

  ```go
  m.RoleService = role.NewService(roleManager)
  m.AssignmentService = assignment.NewService(assignmentManager)
  m.PolicyService = policy.NewService(policyManager)
  m.ResourceService = resource.NewService(resourceManager)
  ```

### 6.2 编写单元测试

- [ ] 为每个领域服务编写单元测试
- [ ] Mock 仓储接口
- [ ] 测试业务规则和异常场景

### 6.3 集成测试

- [ ] 测试应用服务 → 领域服务 → 仓储的完整链路
- [ ] 测试 Casbin 同步
- [ ] 测试版本通知
- [ ] 测试事务回滚

---

## Phase 7: 文档和验证

### 7.1 更新文档

- [ ] 更新架构文档（`docs/uc-architecture.md`）
- [ ] 创建重构总结文档
- [ ] 更新 API 文档（如果有变化）

### 7.2 编译和验证

- [ ] 编译整个项目 `go build ./...`
- [ ] 运行所有测试 `go test ./...`
- [ ] 代码质量检查 `go vet ./...`
- [ ] 格式化代码 `go fmt ./...`

---

## 预期效果

### 重构前（当前）

```
HTTP Request
    ↓
Handler
    ↓
Application Service (含大量业务逻辑) ❌
    ├─ 唯一性校验
    ├─ 租户隔离
    ├─ Casbin 规则构建
    ├─ 事务回滚
    ├─ 版本管理
    └─ ...
    ↓
Repository
```

### 重构后（目标）

```
HTTP Request
    ↓
Handler
    ↓
Application Service (薄编排层) ✅
    ├─ 命令转换
    └─ 领域服务调用
    ↓
Domain Service (领域逻辑) ✅
    ├─ 唯一性校验
    ├─ 租户隔离
    ├─ Casbin 规则构建
    ├─ 事务回滚
    ├─ 版本管理
    └─ ...
    ↓
Repository
```

---

## 关键原则

1. **单一职责**：应用服务只负责编排，领域服务负责业务逻辑
2. **领域驱动**：业务规则放在领域层，靠近领域对象
3. **可测试性**：领域服务独立可测，不依赖外部框架
4. **一致性**：与 authn 模块保持架构一致
5. **渐进式**：分模块重构，每个模块完成后验证

---

## 执行顺序建议

1. **先重构 Resource 模块**（最简单，没有复杂依赖）
2. **再重构 Role 模块**（依赖较少）
3. **然后重构 Policy 模块**（依赖 Role + Resource）
4. **最后重构 Assignment 模块**（依赖 Role + Policy）

每个模块重构完成后：
- ✅ 编译验证
- ✅ 单元测试
- ✅ 集成测试
- ✅ 文档更新

---

## 估计工作量

- Phase 1-5: 每个模块 2-3 小时（共 8-12 小时）
- Phase 6: 2-3 小时
- Phase 7: 1-2 小时

**总计**: 约 11-17 小时

---

## 开始重构？

请确认是否开始重构，我建议：
1. 先从 **Resource 模块**开始（最简单）
2. 每完成一个模块就验证编译和测试
3. 渐进式推进，确保质量
