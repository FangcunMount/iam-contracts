# iam contracts 使用说明

本文档详细介绍了基于六边形架构的Go Web框架的设计理念、核心组件和使用方法。

## 📚 IAM Contracts 文档中心

欢迎来到 IAM Contracts 项目文档中心！

## 🗂️ 文档目录

### 🏗️ 架构设计

位置：[architecture/](./architecture/)

- [框架概览](./architecture/framework-overview.md) - 整体架构设计
- [六边形容器](./architecture/hexagonal-container.md) - DDD 容器化架构
- [代码结构](./architecture/code-structure-apiserver.md) - API Server 代码组织
- [项目结构](./architecture/project-structure.md) - 目录结构说明

### 🔐 认证系统

位置：[authentication/](./authentication/)

#### 设计文档

- [认证设计](./authentication/authentication-design.md) - DDD 领域驱动设计
- [实现总结](./authentication/authentication-implementation-summary.md) - 完整实现文档

#### 分层总结

- [服务层总结](./authentication/authentication-service-summary.md) - Domain 层设计
- [基础设施层](./authentication/authentication-infrastructure-summary.md) - 基础设施适配器
- [应用层](./authentication/authentication-application-summary.md) - 应用服务

#### 使用指南

- [中间件指南](./authentication/authentication-middleware-guide.md) - JWT 认证中间件详解
- [集成方案](./authentication/authentication-integration.md) - 快速集成指南
- [快速参考](./authentication/authentication-quick-reference.md) - API 速查表

### 🔧 系统功能

- [数据库注册](./database-registry.md) - 多数据库管理
- [错误处理](./error-handling.md) - 统一错误处理机制
- [日志系统](./logging-system.md) - 结构化日志方案

### � IAM 规范

位置：[iam/](./iam/)

- IAM 相关的规范和标准文档

## �🚀 快速开始

1. **新手入门**: 从 [框架概览](./architecture/framework-overview.md) 开始
2. **认证开发**: 查看 [认证集成方案](./authentication/authentication-integration.md)
3. **API 参考**: 使用 [快速参考手册](./authentication/authentication-quick-reference.md)

## 📖 文档结构

```text
docs/
├── README.md                    # 📚 本文档（文档索引）
├── architecture/                # 🏗️ 架构设计文档
│   ├── framework-overview.md
│   ├── hexagonal-container.md
│   ├── code-structure-apiserver.md
│   └── project-structure.md
├── authentication/              # 🔐 认证系统文档
│   ├── authentication-design.md
│   ├── authentication-implementation-summary.md
│   ├── authentication-service-summary.md
│   ├── authentication-infrastructure-summary.md
│   ├── authentication-application-summary.md
│   ├── authentication-middleware-guide.md
│   ├── authentication-integration.md
│   └── authentication-quick-reference.md
├── database-registry.md         # 🗄️ 数据库注册
├── error-handling.md            # ⚠️ 错误处理
├── logging-system.md            # 📝 日志系统
└── iam/                         # 📋 IAM 规范
```

## 🔄 最近更新

- ✅ 完成认证系统迁移：从旧的 `middleware/auth` 迁移到新的 DDD 架构
- ✅ 整理文档结构：按主题组织，便于查找
- ✅ 删除废弃文档：移除迁移过程中的临时文档

## 💡 贡献指南

更新文档时请遵循以下原则：

1. **按主题组织**: 将相关文档放在对应的子目录
2. **保持更新**: 代码变更时同步更新文档
3. **清晰简洁**: 使用清晰的标题和示例
4. **添加索引**: 在本文档中添加新文档的链接

## � 获取帮助

- 查看具体文档了解详细信息
- 查阅代码中的注释
- 参考 `examples/` 目录中的示例
