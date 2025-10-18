# AuthZ 模块文档索引

欢迎来到 AuthZ（授权）模块！本目录包含完整的架构文档、设计说明和使用指南。

## 📚 文档导航

### 🎯 快速开始

1. **[代码检查摘要](./CODE_REVIEW_SUMMARY.md)** ⭐ 最新状态
   - 编译和代码检查结果
   - 完成度统计（66%）
   - 需要立即处理的问题
   - 下一步行动计划
   - 代码质量评分

2. **[重构总结](./REFACTORING_SUMMARY.md)** ⭐ 推荐首读
   - 重构目标和已完成工作
   - 当前文件清单
   - 待完成任务列表
   - 下一步行动计划

3. **[目录树](./DIRECTORY_TREE.md)**
   - 完整目录结构（带详细注释）
   - 关键文件说明
   - 设计模式对照
   - 工作流程说明
   - 依赖关系

### 📖 深入理解

1. **[完整代码检查报告](./CODE_REVIEW_REPORT.md)** 📊 详细分析
   - 所有文件的检查结果
   - 代码质量分析
   - 设计模式应用评审
   - 并发安全性检查
   - 数据库设计审查
   - 发现的问题和建议

2. **[架构文档](./README.md)**
   - 系统架构图
   - PAP/PRP/PDP/PEP 组件职责
   - 关键设计决策
   - 数据库 Schema
   - API 示例
   - 使用指南
   - V2 规划

3. **[架构图集](./ARCHITECTURE_DIAGRAMS.md)**
   - 系统架构图（Mermaid）
   - 分层架构图
   - 权限判定流程图
   - 策略管理流程图
   - XACML 映射图
   - 依赖关系图

### 🗂️ 配置与数据

1. **[资源目录](./resources.seed.yaml)**
   - 预定义域对象资源
   - 允许的动作列表
   - Seed 数据示例

2. **[策略示例](./policy_init.csv)**
   - 初始角色定义
   - p 规则示例（策略）
   - g 规则示例（赋权）
   - 多租户配置示例

## 🏗️ 架构概览

AuthZ 模块采用 **DDD（领域驱动设计）+ 六边形架构**，遵循 **XACML 标准**的 PAP-PRP-PDP-PEP 四层架构：

```text
┌─────────────────────────────────────────────┐
│  PEP (执行点)                                │  ← interface/sdk/pep/
│  两段式权限判定：*_all → *_own             │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│  PDP (决策点)                                │  ← infra/casbin/
│  Casbin CachedEnforcer 布尔判定            │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│  PRP (存储点)                                │  ← infra/mysql/
│  casbin_rule + 领域小表                     │
└─────────────────────────────────────────────┘
                    ↑
┌─────────────────────────────────────────────┐
│  PAP (管理点)                                │  ← application/ + interface/restful/
│  角色/赋权/策略/资源管理 REST API           │
└─────────────────────────────────────────────┘
```

## 📦 核心模块

### 领域层（Domain）

- **role/** - 角色聚合
- **assignment/** - 赋权聚合
- **resource/** - 资源聚合
- **policy/** - 策略聚合

### 基础设施层（Infrastructure）

- **infra/mysql/** - 持久化（PO + Mapper + Repo）
- **infra/casbin/** - 策略引擎（Model + Adapter + Enforcer）
- **infra/redis/** - 版本通知（Pub/Sub）

### 应用层（Application）

- **application/role/** - 角色管理服务
- **application/assignment/** - 赋权管理服务
- **application/policy/** - 策略管理服务
- **application/resource/** - 资源管理服务
- **application/version/** - 版本管理服务

### 接口层（Interface）

- **interface/restful/** - PAP 管理 REST API
- **interface/sdk/pep/** - PEP 执行点 SDK

## 🔑 核心概念

### RBAC 模型（V1）

```text
用户/组 →(g规则)→ 角色 →(p规则)→ 域对象 + 动作
```

### 域对象资源格式

```text
<app>:<domain>:<type>:*
例如: scale:form:*, ops:user:*
```

### 动作范围编码

```text
*_all  - 全局权限（不需要 owner 校验）
*_own  - 所有者权限（需要 owner 校验）
```

### 两段式权限判定

```go
// 1. 先判全局权限
if guard.Can(ctx).Read("scale:form:*").All() {
    return repo.FindByID(id)
}

// 2. 再判所有者权限
if guard.Can(ctx).Read("scale:form:*").Own(userID) {
    form := repo.FindByID(id)
    if form.OwnerID == userID {
        return form
    }
}

return ErrForbidden
```

## 🎨 设计原则

- ✅ **依赖倒置原则**（DIP）：domain 定义接口，infra 实现接口
- ✅ **单一职责原则**（SRP）：每个聚合管理自己的生命周期
- ✅ **开闭原则**（OCP）：通过 Port 接口扩展实现
- ✅ **接口隔离原则**（ISP）：每个聚合独立的 Repository 接口

## 🚀 快速链接

| 需求 | 文档 |
|------|------|
| 了解项目现状 | [重构总结](./REFACTORING_SUMMARY.md) |
| 查看目录结构 | [目录树](./DIRECTORY_TREE.md) |
| 理解架构设计 | [架构文档](./README.md) |
| 查看架构图 | [架构图集](./ARCHITECTURE_DIAGRAMS.md) |
| 配置资源 | [资源目录](./resources.seed.yaml) |
| 初始化策略 | [策略示例](./policy_init.csv) |

## 📝 注意事项

1. **V1 约束**:
   - 仅支持域对象类型级权限（无实例级）
   - 动作范围编码进动作名（避免 ABAC）
   - 所有者判断在业务服务内完成

2. **性能优化**:
   - 使用 CachedEnforcer 减少查询
   - 策略变更通过 Redis 广播通知
   - 合理设置缓存 TTL 和 LRU 大小

3. **安全考虑**:
   - 所有策略变更需审计日志
   - 敏感操作需二次确认
   - 定期审查权限分配

## 🤝 参考资源

- **Authn 模块**: `/internal/apiserver/modules/authn/`（本模块参照其架构）
- **BaseRepository**: `/internal/pkg/database/mysql/base.go`
- **Casbin 官方文档**: <https://casbin.org/>
- **RBAC 权限模型**: <https://en.wikipedia.org/wiki/Role-based_access_control>
- **XACML 架构**: <https://en.wikipedia.org/wiki/XACML>

## 📌 待办事项

查看 [重构总结](./REFACTORING_SUMMARY.md) 获取完整的待办列表和优先级。

主要待完成：

- ⏳ 完成其他 MySQL Repository 实现
- ⏳ 实现 Application 层服务
- ⏳ 实现 REST API Handler (PAP)
- ⏳ 实现 PEP SDK (DomainGuard)
- ⏳ 实现 Redis 版本通知
- ⏳ 编写单元测试和集成测试

---

**最后更新**: 2025-10-18  
**版本**: V1.0  
**状态**: 架构搭建完成，核心实现进行中

如有疑问，请参考各详细文档或联系架构团队。
