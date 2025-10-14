# 🏗️ 架构设计文档

IAM Contracts 项目的架构设计文档集合。

## 📚 文档列表

### 📖 核心架构文档

1. **[框架概览](./framework-overview.md)**
   - 项目整体设计理念
   - 技术栈选择
   - 设计原则和最佳实践

2. **[六边形容器](./hexagonal-container.md)**
   - DDD（领域驱动设计）架构
   - 依赖注入容器
   - 模块化设计

3. **[项目结构](./project-structure.md)**
   - 目录组织
   - 文件命名规范
   - 模块划分

4. **[代码结构](./code-structure-apiserver.md)**
   - API Server 代码组织
   - 分层架构
   - 模块关系

## 🎯 快速导航

### 新手入门

1. 阅读 [框架概览](./framework-overview.md) 了解整体设计
2. 查看 [项目结构](./project-structure.md) 熟悉目录组织
3. 学习 [六边形容器](./hexagonal-container.md) 理解 DDD 架构

### 开发指南

- **添加新模块**: 参考 [六边形容器](./hexagonal-container.md#添加新模块)
- **理解分层**: 查看 [代码结构](./code-structure-apiserver.md)
- **最佳实践**: 阅读 [框架概览](./framework-overview.md#设计原则)

## 🏛️ 架构概览

```text
IAM Contracts
│
├── 🌐 Interface Layer (接口层)
│   ├── RESTful API
│   ├── gRPC Service
│   └── HTTP Handler
│
├── 💼 Application Layer (应用层)
│   ├── Use Cases
│   ├── DTOs
│   └── Application Services
│
├── 🎯 Domain Layer (领域层)
│   ├── Entities
│   ├── Value Objects
│   ├── Domain Services
│   └── Repositories (interfaces)
│
└── 🔧 Infrastructure Layer (基础设施层)
    ├── Database (MySQL)
    ├── Cache (Redis)
    ├── External APIs
    └── Repository Implementations
```

## 📝 设计原则

### DDD 四层架构

1. **Interface Layer**: 与外部交互
2. **Application Layer**: 编排业务流程
3. **Domain Layer**: 核心业务逻辑
4. **Infrastructure Layer**: 技术实现细节

### 依赖规则

- 外层依赖内层
- Domain Layer 不依赖任何层
- Infrastructure Layer 实现 Domain Layer 的接口

详细说明请查看：[六边形容器](./hexagonal-container.md)

## 🔗 相关文档

- [认证系统文档](../authentication/README.md)
- [数据库注册](../database-registry.md)
- [错误处理](../error-handling.md)
- [日志系统](../logging-system.md)

---

返回 [文档中心](../README.md)
