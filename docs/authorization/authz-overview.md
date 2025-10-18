# Authorization (AuthZ) 模块文档

欢迎来到 AuthZ（授权）模块文档！本模块实现了基于 RBAC 的域对象级权限控制系统。

## 📖 文档导航

### 快速开始

- **[文档索引](./INDEX.md)** ⭐ 推荐首读
  - 完整的文档导航和快速链接
  - 核心概念速览
  - 使用建议

- **[重构总结](./REFACTORING_SUMMARY.md)** ⭐ 了解项目现状
  - 重构目标和已完成工作
  - 当前文件清单
  - 待完成任务列表
  - 下一步行动计划

### 深入理解

- **[架构文档](./README.md)**
  - 系统架构图
  - PAP/PRP/PDP/PEP 组件职责
  - 关键设计决策
  - 数据库 Schema
  - API 示例和使用指南
  - V2 规划

- **[目录树](./DIRECTORY_TREE.md)**
  - 完整目录结构（带详细注释）
  - 关键文件说明
  - 设计模式对照
  - 工作流程说明
  - 依赖关系

- **[架构图集](./ARCHITECTURE_DIAGRAMS.md)**
  - 系统架构图（Mermaid）
  - 分层架构图
  - 权限判定流程图
  - 策略管理流程图
  - XACML 映射图
  - 依赖关系图

### 配置与数据

- **[资源目录](./resources.seed.yaml)**
  - 预定义域对象资源
  - 允许的动作列表
  - Seed 数据示例

- **[策略示例](./policy_init.csv)**
  - 初始角色定义
  - p 规则示例（策略）
  - g 规则示例（赋权）
  - 多租户配置示例

## 🎯 核心概念

### XACML 四层架构

AuthZ 模块遵循 XACML 标准的 PAP-PRP-PDP-PEP 四层架构：

```text
┌─────────────────────────────────────────────┐
│  PEP (执行点)                                │
│  两段式权限判定：*_all → *_own             │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│  PDP (决策点)                                │
│  Casbin CachedEnforcer 布尔判定            │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│  PRP (存储点)                                │
│  casbin_rule + 领域小表                     │
└─────────────────────────────────────────────┘
                    ↑
┌─────────────────────────────────────────────┐
│  PAP (管理点)                                │
│  角色/赋权/策略/资源管理 REST API           │
└─────────────────────────────────────────────┘
```

### RBAC 模型

```
用户/组 →(g规则)→ 角色 →(p规则)→ 域对象 + 动作
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

## 🏗️ 技术架构

### 设计模式

- **DDD（领域驱动设计）**: 4 个聚合根（role, assignment, resource, policy）
- **六边形架构**: Port/Adapter 模式实现依赖倒置
- **CQRS**: 读写分离，优化查询性能

### 技术栈

- **权限引擎**: Casbin v2（RBAC 模型）
- **持久化**: MySQL（GORM）
- **缓存**: CachedEnforcer（内置）+ Redis Pub/Sub
- **API**: Gin（REST）+ gRPC（可选）

### 核心模块

```
domain/              # 领域层（4个聚合）
  ├── role/          # 角色聚合
  ├── assignment/    # 赋权聚合
  ├── resource/      # 资源聚合
  └── policy/        # 策略聚合

infra/               # 基础设施层
  ├── mysql/         # MySQL 实现
  ├── casbin/        # Casbin 封装
  └── redis/         # Redis 实现

application/         # 应用层（用例编排）
  ├── role/          # 角色管理服务
  ├── assignment/    # 赋权管理服务
  ├── policy/        # 策略管理服务
  ├── resource/      # 资源管理服务
  └── version/       # 版本管理服务

interface/           # 接口层
  ├── restful/       # PAP 管理 API
  └── sdk/pep/       # PEP 执行点 SDK
```

## 🚀 快速链接

| 需求 | 文档 |
|------|------|
| 了解项目现状 | [重构总结](./REFACTORING_SUMMARY.md) |
| 查看目录结构 | [目录树](./DIRECTORY_TREE.md) |
| 理解架构设计 | [架构文档](./README.md) |
| 查看架构图 | [架构图集](./ARCHITECTURE_DIAGRAMS.md) |
| 配置资源 | [资源目录](./resources.seed.yaml) |
| 初始化策略 | [策略示例](./policy_init.csv) |

## 📝 V1 特性

- ✅ **RBAC 模型**: 角色继承 + 域隔离（租户）
- ✅ **域对象级权限**: `<app>:<domain>:<type>:*` 格式
- ✅ **动作范围编码**: `*_all` / `*_own` 区分全局和所有者权限
- ✅ **两段式判定**: 先判全局，再判所有者
- ✅ **嵌入式决策**: CachedEnforcer 在业务服务内（低延迟）
- ✅ **策略版本管理**: 版本递增 + Redis 广播通知
- ✅ **缓存失效机制**: 自动刷新策略缓存

## 🔮 V2 规划

- 📋 菜单/前端路由管理
- 📋 API 路由自动扫描与注册
- 📋 ABAC 属性判断支持
- 📋 数据权限（字段级、行级）
- 📋 策略模拟与冲突检测
- 📋 完整审计流水

## 🤝 相关资源

- **认证模块**: [/docs/authentication/](../authentication/) - AuthN 认证系统
- **项目架构**: [/docs/architecture-overview.md](../architecture-overview.md) - 整体架构设计
- **Casbin 官方文档**: https://casbin.org/
- **XACML 标准**: https://en.wikipedia.org/wiki/XACML

## 💡 注意事项

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

---

**最后更新**: 2025-10-18  
**版本**: V1.0  
**状态**: 架构搭建完成，核心实现进行中

如有疑问，请参考各详细文档或联系架构团队。
