# AuthZ 模块重构总结

## 概述

本次重构按照 **authn 模块的实现方式**，将 authz 模块拆分为清晰的领域驱动设计（DDD）架构，遵循六边形架构模式，实现了面向 V1 的 RBAC 域对象权限控制。

## 重构目标

✅ 按照 authn 模块的模式组织代码结构
✅ 领域对象与持久化对象分离（BO ↔ PO）
✅ 使用 Port/Adapter 模式实现依赖倒置
✅ 每个聚合独立目录管理
✅ 实现 PAP-PRP-PDP-PEP 四层架构
✅ 支持嵌入式 Casbin Enforcer 决策
✅ 支持策略版本管理与缓存失效通知

## 已完成的工作

### 1. 领域层（Domain Layer）

按照 authn 模式，将领域划分为 4 个独立聚合：

#### ✅ role/ - 角色聚合
- `role.go`: 角色实体 + RoleID 值对象
- `port/driven/repo.go`: 角色仓储接口

#### ✅ assignment/ - 赋权聚合
- `assignment.go`: 赋权实体 + AssignmentID 值对象 + SubjectType 枚举
- `port/driven/repo.go`: 赋权仓储接口

#### ✅ resource/ - 资源聚合
- `resource.go`: 资源实体 + ResourceID 值对象
- `action.go`: 动作值对象 + 预定义动作枚举（create, read_all, read_own 等）
- `port/driven/repo.go`: 资源仓储接口

#### ✅ policy/ - 策略聚合
- `policy_version.go`: 策略版本实体 + PolicyVersionID 值对象
- `rule.go`: 策略规则值对象（PolicyRule + GroupingRule）
- `port/driven/repo.go`: 版本仓储接口
- `port/driven/casbin.go`: Casbin 操作接口（CasbinPort）

**设计特点**:
- 每个聚合拥有独立的值对象（ID 类型）
- 使用 Port/Adapter 模式定义仓储接口
- 领域对象包含业务方法（如 `Key()`, `SubjectKey()`, `HasAction()`）
- 零外部依赖（纯业务逻辑）

### 2. 基础设施层（Infrastructure Layer）

#### ✅ infra/mysql/ - MySQL 持久化实现

按照 authn 的 PO + Mapper + Repo 三件套模式：

**role/**
- `po.go`: RolePO 持久化对象（对应 `authz_roles` 表）
- `mapper.go`: BO ↔ PO 转换器
- `repo.go`: RoleRepository 实现（继承 BaseRepository）

**assignment/**
- `po.go`: AssignmentPO（对应 `authz_assignments` 表）

**resource/**
- `po.go`: ResourcePO（对应 `authz_resources` 表）

**policy/**
- `po.go`: PolicyVersionPO（对应 `authz_policy_versions` 表）

**设计特点**:
- 所有 PO 继承 `base.AuditFields`（包含审计字段）
- 实现 `BeforeCreate` 和 `BeforeUpdate` 钩子
- Repository 继承 `mysql.BaseRepository` 提供通用 CRUD
- Mapper 负责领域对象与持久化对象的双向转换

#### ✅ infra/casbin/ - Casbin 策略引擎

- `model.conf`: RBAC 模型定义（支持域隔离和角色继承）
- `adapter.go`: CasbinAdapter 实现（封装 CachedEnforcer）

**模型特点**:
```
r = sub, dom, obj, act  # 请求：主体、域、对象、动作
p = sub, dom, obj, act  # 策略：角色、域、对象、动作
g = _, _, _             # 分组：用户 → 角色（支持域）
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act
```

### 3. 文档（Documentation）

#### ✅ docs/README.md - 完整架构文档
包含：
- 系统架构图（文本版）
- 组件职责说明（PAP/PRP/PDP/PEP）
- 完整目录结构
- 关键设计决策
- 数据库 Schema
- API 示例
- 使用指南
- V2 规划

#### ✅ docs/DIRECTORY_TREE.md - 目录树文档
包含：
- 完整目录结构（带注释）
- 设计模式对照表
- XACML 架构映射
- 关键文件说明
- 工作流程说明
- 依赖关系图

#### ✅ docs/ARCHITECTURE_DIAGRAMS.md - 架构图集
包含 Mermaid 图表：
- 系统架构图
- 分层架构图
- 权限判定流程图
- 策略管理流程图
- XACML 映射图
- 依赖关系图

#### ✅ docs/resources.seed.yaml - 资源目录
预定义资源：
- `scale:form:*` - 量表表单
- `scale:report:*` - 量表报告
- `ops:user:*` - 用户管理
- `ops:role:*` - 角色管理
- `ops:permission:*` - 权限管理

#### ✅ docs/policy_init.csv - 策略示例
示例角色：
- `scale-editor` - 量表编辑员
- `scale-reviewer` - 量表审批员
- `scale-admin` - 量表管理员
- `report-viewer` - 报告查看员
- `ops-admin` - 运营管理员

### 4. 接口层（Interface Layer）

#### ✅ interface/restful/router.go
当前包含 health 检查端点，后续扩展为 PAP 管理 API。

## 当前文件清单

```
internal/apiserver/modules/authz/
├── docs/
│   ├── ARCHITECTURE_DIAGRAMS.md    ✅ 架构图集（Mermaid）
│   ├── DIRECTORY_TREE.md           ✅ 目录树文档
│   ├── README.md                   ✅ 完整架构文档
│   ├── policy_init.csv             ✅ 策略示例
│   └── resources.seed.yaml         ✅ 资源目录
│
├── domain/
│   ├── assignment/
│   │   ├── assignment.go           ✅ 赋权实体
│   │   └── port/driven/repo.go     ✅ 赋权仓储接口
│   ├── policy/
│   │   ├── policy_version.go       ✅ 策略版本实体
│   │   ├── rule.go                 ✅ 策略规则值对象
│   │   └── port/driven/
│   │       ├── casbin.go           ✅ Casbin 操作接口
│   │       └── repo.go             ✅ 版本仓储接口
│   ├── resource/
│   │   ├── action.go               ✅ 动作值对象
│   │   ├── resource.go             ✅ 资源实体
│   │   └── port/driven/repo.go     ✅ 资源仓储接口
│   └── role/
│       ├── role.go                 ✅ 角色实体
│       └── port/driven/repo.go     ✅ 角色仓储接口
│
├── infra/
│   ├── casbin/
│   │   ├── adapter.go              ✅ Casbin 适配器
│   │   └── model.conf              ✅ RBAC 模型
│   └── mysql/
│       ├── assignment/po.go        ✅ 赋权 PO
│       ├── policy/po.go            ✅ 策略版本 PO
│       ├── resource/po.go          ✅ 资源 PO
│       └── role/
│           ├── mapper.go           ✅ BO ↔ PO 转换
│           ├── po.go               ✅ 角色 PO
│           └── repo.go             ✅ 角色 Repository
│
└── interface/
    └── restful/
        └── router.go               ✅ 路由（基础）
```

## 待完成的工作

### 🔜 高优先级

1. **完成其他 MySQL Repository 实现**
   - `infra/mysql/assignment/mapper.go` + `repo.go`
   - `infra/mysql/resource/mapper.go` + `repo.go`
   - `infra/mysql/policy/mapper.go` + `repo.go`

2. **实现 Application 层服务**
   - `application/role/service.go` - 角色管理服务
   - `application/assignment/service.go` - 赋权管理服务
   - `application/policy/service.go` - 策略管理服务
   - `application/resource/service.go` - 资源管理服务
   - `application/version/service.go` - 版本管理服务

3. **实现 REST API（PAP）**
   - `interface/restful/handler_pap.go` - PAP 管理接口
   - `interface/restful/dto/*.go` - DTO 对象

4. **实现 PEP SDK**
   - `interface/sdk/go/pep/guard.go` - DomainGuard 核心
   - `interface/sdk/go/pep/context.go` - 上下文提取
   - `interface/sdk/go/pep/middleware.go` - 中间件（可选）

5. **实现 Redis 版本通知**
   - `infra/redis/version_pubsub.go` - 发布/订阅实现

### 🔜 中优先级

6. **数据库迁移脚本**
   - 创建表结构的 SQL 或 GORM AutoMigrate
   - Seed 数据导入脚本

7. **单元测试**
   - 领域层单元测试
   - Repository 集成测试
   - Application 服务测试

8. **集成测试**
   - Casbin 规则测试
   - E2E 权限判定测试

### 🔜 低优先级

9. **PDP 决策服务（可选）**
   - `interface/restful/handler_pdp.go` - `/v1/decide` REST API

10. **审计日志**
    - 策略变更审计
    - 权限判定失败采样

## 核心设计原则

### 1. 依赖倒置原则（DIP）
```
interface/    →  application/  →  domain/port/
infra/        →  domain/port/  (实现接口)
```

- interface 和 infra 层依赖 domain 定义的接口
- domain 层零外部依赖

### 2. 单一职责原则（SRP）
- 每个聚合管理自己的生命周期
- Repository 只负责持久化
- Service 只负责用例编排
- Handler 只负责 HTTP 处理

### 3. 开闭原则（OCP）
- 通过 Port 接口扩展实现
- 可轻松替换 MySQL → MongoDB
- 可轻松替换 Casbin → OPA

### 4. 接口隔离原则（ISP）
- 每个聚合独立的 Repository 接口
- CasbinPort 只暴露必要操作

## 与 Authn 模块的一致性

| 特性 | Authn | AuthZ | 一致性 |
|------|-------|-------|--------|
| 聚合独立目录 | ✅ account/, authentication/, jwks/ | ✅ role/, assignment/, resource/, policy/ | ✅ |
| Port/Adapter 模式 | ✅ port/driven/ | ✅ port/driven/ | ✅ |
| PO + Mapper + Repo | ✅ | ✅ | ✅ |
| 继承 BaseRepository | ✅ | ✅ | ✅ |
| 值对象（ID 类型） | ✅ AccountID, UserID | ✅ RoleID, AssignmentID, ResourceID | ✅ |
| 领域服务目录 | ✅ domain/account/service/ | 🔜 application/ | ⚠️ 命名不同但概念一致 |

**注**: AuthZ 使用 `application/` 目录放置服务层，概念上等同于 authn 的 `domain/*/service/`，都是用例编排层。

## 技术栈

- **ORM**: GORM
- **权限引擎**: Casbin v2
- **缓存**: CachedEnforcer（内置）
- **消息**: Redis Pub/Sub（策略版本通知）
- **Web 框架**: Gin（REST API）
- **文档**: Markdown + Mermaid

## 下一步行动

### 立即开始
1. 安装 Casbin 依赖: `go get github.com/casbin/casbin/v2 github.com/casbin/gorm-adapter/v3`
2. 完成其他 3 个 Repository 实现（assignment, resource, policy）
3. 实现 Application 层 5 个服务

### 短期目标（本周）
4. 实现 REST API Handler（PAP 管理接口）
5. 实现 PEP SDK（DomainGuard）
6. 实现 Redis 版本通知

### 中期目标（2周内）
7. 编写单元测试和集成测试
8. 创建数据库迁移脚本
9. 导入 seed 数据

### 长期目标（1个月内）
10. 在实际业务服务中集成测试
11. 性能测试和优化
12. 完善文档和示例代码

## 相关资源

- **Authn 模块参考**: `/internal/apiserver/modules/authn/`
- **BaseRepository**: `/internal/pkg/database/mysql/base.go`
- **Casbin 文档**: https://casbin.org/
- **XACML 标准**: https://en.wikipedia.org/wiki/XACML

## 总结

本次重构成功地按照 authn 模块的实现方式，建立了清晰的 DDD 架构和六边形架构模式。核心领域逻辑、端口定义、基础设施实现完全分离，为后续扩展和测试打下了坚实基础。

架构文档完整详尽，包含架构图、目录树、设计决策、使用示例等，便于团队理解和协作。

下一步重点是完成基础设施层的其他 Repository 实现和应用层服务，然后快速进入集成测试阶段。

---

**创建时间**: 2025-10-18
**版本**: V1.0
**状态**: 架构搭建完成，等待实现细节补充
