# IAM Contracts 文档中心

欢迎来到 IAM Contracts 项目文档中心！本项目是一个基于 **六边形架构 + DDD + CQRS** 的 IAM（身份与访问管理）系统。

## 📚 文档导航

### 🏗️ 架构设计

位置：[architecture/](./architecture/)

- [框架概览](./architecture/framework-overview.md) - 整体架构设计
- [六边形容器](./architecture/hexagonal-container.md) - DDD 容器化架构
- [代码结构](./architecture/code-structure-apiserver.md) - API Server 代码组织
- [项目结构](./architecture/project-structure.md) - 目录结构说明
- [架构总览](./architecture-overview.md) - 完整架构文档 ⭐

### 🔐 认证系统 (AuthN)

位置：[authentication/](./authentication/)

认证中心提供多渠道登录、JWT Token 管理、JWKS 公钥发布等功能。

#### 核心文档

- **[认证设计](./authentication/authentication-design.md)** ⭐ - DDD 领域驱动设计
- **[实现总结](./authentication/authentication-implementation-summary.md)** - 完整实现文档
- **[架构设计](./authn-architecture.md)** - Authn 模块完整设计

#### 分层总结

- [Domain 层](./authentication/authentication-service-summary.md) - 领域服务设计
- [Infrastructure 层](./authentication/authentication-infrastructure-summary.md) - 基础设施适配器
- [Application 层](./authentication/authentication-application-summary.md) - 应用服务

#### 使用指南

- [中间件指南](./authentication/authentication-middleware-guide.md) - JWT 认证中间件详解
- [集成方案](./authentication/authentication-integration.md) - 快速集成指南
- [快速参考](./authentication/authentication-quick-reference.md) - API 速查表

**核心特性**:

- ✅ 多渠道登录（微信小程序、企业微信、本地密码）
- ✅ JWT Token 管理（签发、验证、刷新、撤销）
- ✅ JWKS 公钥发布与密钥轮换
- ✅ 安全设计（密码哈希、防重放、速率限制）

### 🛡️ 授权系统 (AuthZ)

位置：[authorization/](./authorization/)

授权中心提供基于 RBAC 的域对象级权限控制，遵循 XACML 标准的 PAP-PRP-PDP-PEP 四层架构。

#### 核心文档

- **[授权概览](./authorization/authz-overview.md)** ⭐ - 快速入口
- **[架构文档](./authorization/README.md)** - 完整架构设计
- **[重构总结](./authorization/REFACTORING_SUMMARY.md)** - 项目现状和待办

#### 详细文档

- [文档索引](./authorization/INDEX.md) - 完整文档导航
- [目录树](./authorization/DIRECTORY_TREE.md) - 目录结构详解
- [架构图集](./authorization/ARCHITECTURE_DIAGRAMS.md) - Mermaid 流程图

#### 配置与数据

- [资源目录](./authorization/resources.seed.yaml) - 预定义资源
- [策略示例](./authorization/policy_init.csv) - 初始策略配置

**核心特性**:

- ✅ RBAC 模型（角色继承 + 域隔离）
- ✅ 域对象级权限控制
- ✅ 两段式权限判定（*_all / *_own）
- ✅ 嵌入式决策引擎（Casbin CachedEnforcer）
- ✅ 策略版本管理与缓存失效
- 🔜 菜单管理、API 扫描、ABAC 支持（V2）

### 👥 用户中心 (UC)

位置：[uc-architecture.md](./uc-architecture.md)

- UC 模块完整设计
- 领域模型（User, Child, Guardianship）
- CQRS 命令查询分离实现
- 四层架构详解
- RESTful + gRPC API 设计

### 🔧 系统功能

- [数据库注册](./database-registry.md) - 多数据库管理
- [错误处理](./error-handling.md) - 统一错误处理机制
- [日志系统](./logging-system.md) - 结构化日志方案

### 📐 IAM 规范

位置：[iam/](./iam/)

- IAM 相关的规范和标准文档

### 🐛 错误码设计

- [错误码重构](./error-code-refactoring.md) - 错误码设计方案
- [重构总结](./error-code-refactoring-summary.md) - 重构实施记录
- [注册修复](./error-code-registration-fix.md) - 错误码注册问题修复

### 🔍 领域层设计

- [领域层设计分析](./domain-layer-design-analysis.md) - DDD 领域层深度分析

## 🎯 核心概念

### 六边形架构（Hexagonal Architecture）

```
┌─────────────────────────────────────────────┐
│              Interface Layer                 │  HTTP/gRPC/CLI
│         (REST API / gRPC Service)            │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│            Application Layer                 │  Use Cases
│         (Application Services)               │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│              Domain Layer                    │  Business Logic
│    (Aggregates / Entities / Services)        │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│          Infrastructure Layer                │  DB/Cache/MQ
│      (Repositories / Adapters)               │
└─────────────────────────────────────────────┘
```

### DDD（领域驱动设计）

- **聚合根**: 每个聚合独立管理生命周期
- **值对象**: 不可变对象，用于描述领域概念
- **领域服务**: 封装复杂业务逻辑
- **仓储模式**: 抽象数据访问
- **端口适配器**: 依赖倒置，面向接口编程

### CQRS（命令查询分离）

- **命令**: 写操作，改变系统状态
- **查询**: 读操作，不改变状态
- **分离优势**: 读写优化、扩展灵活、模型清晰

## 🚀 快速开始

### 新手入门路径

1. **理解架构** → [框架概览](./architecture/framework-overview.md)
   - 了解六边形架构 + DDD + CQRS
   - 熟悉项目目录结构
   - 理解核心设计原则

2. **学习认证** → [认证设计](./authentication/authentication-design.md)
   - 掌握 JWT 认证流程
   - 理解多渠道登录实现
   - 学习安全最佳实践

3. **学习授权** → [授权概览](./authorization/authz-overview.md)
   - 理解 RBAC 权限模型
   - 掌握两段式权限判定
   - 学习 Casbin 策略引擎

4. **实战开发** → [集成方案](./authentication/authentication-integration.md)
   - 业务服务集成认证
   - 使用权限判定 SDK
   - API 参考和示例

### 开发环境搭建

```bash
# 1. 克隆项目
git clone https://github.com/fangcun-mount/iam-contracts.git
cd iam-contracts

# 2. 启动基础设施（MySQL + Redis）
cd build/docker/infra
docker-compose up -d

# 3. 安装依赖
go mod download

# 4. 运行测试
go test ./...

# 5. 启动 API Server
go run cmd/apiserver/apiserver.go
```

## 📖 文档结构

```text
docs/
├── README.md                           # 📚 本文档（文档索引）
│
├── architecture/                       # 🏗️ 架构设计文档
│   ├── framework-overview.md
│   ├── hexagonal-container.md
│   ├── code-structure-apiserver.md
│   └── project-structure.md
│
├── authentication/                     # 🔐 认证系统文档
│   ├── authentication-design.md       # 核心设计 ⭐
│   ├── authentication-implementation-summary.md
│   ├── authentication-service-summary.md
│   ├── authentication-infrastructure-summary.md
│   ├── authentication-application-summary.md
│   ├── authentication-middleware-guide.md
│   ├── authentication-integration.md
│   └── authentication-quick-reference.md
│
├── authorization/                      # 🛡️ 授权系统文档
│   ├── authz-overview.md              # 授权概览 ⭐
│   ├── README.md                      # 完整架构文档
│   ├── REFACTORING_SUMMARY.md         # 重构总结
│   ├── INDEX.md                       # 文档索引
│   ├── DIRECTORY_TREE.md              # 目录树
│   ├── ARCHITECTURE_DIAGRAMS.md       # 架构图集
│   ├── resources.seed.yaml            # 资源配置
│   └── policy_init.csv                # 策略示例
│
├── iam/                                # 📐 IAM 规范
│
├── architecture-overview.md            # 完整架构总览 ⭐
├── authn-architecture.md               # 认证架构设计
├── uc-architecture.md                  # 用户中心设计
├── domain-layer-design-analysis.md     # 领域层分析
├── database-registry.md                # 数据库管理
├── error-handling.md                   # 错误处理
├── logging-system.md                   # 日志系统
├── error-code-refactoring.md           # 错误码设计
├── error-code-refactoring-summary.md
└── error-code-registration-fix.md
```

## 🎨 技术栈

### 后端框架

- **Go 1.21+**: 编程语言
- **Gin**: Web 框架
- **gRPC**: RPC 框架
- **GORM**: ORM 框架

### 认证授权

- **JWT**: Token 认证
- **Casbin**: 权限引擎
- **Argon2**: 密码哈希

### 存储

- **MySQL**: 主数据库
- **Redis**: 缓存 & Pub/Sub

### 开发工具

- **Air**: 热重载
- **golangci-lint**: 代码检查
- **protoc**: gRPC 代码生成

## 💡 设计原则

1. **依赖倒置原则** (DIP): Domain 定义接口，Infrastructure 实现
2. **单一职责原则** (SRP): 每个模块只负责一个功能
3. **开闭原则** (OCP): 对扩展开放，对修改关闭
4. **接口隔离原则** (ISP): 接口最小化，避免冗余依赖
5. **里氏替换原则** (LSP): 子类可以替换父类

## 🤝 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 📞 联系方式

- **项目仓库**: https://github.com/fangcun-mount/iam-contracts
- **问题反馈**: https://github.com/fangcun-mount/iam-contracts/issues

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](../LICENSE) 文件

---

**最后更新**: 2025-10-18  
**版本**: V1.0

欢迎贡献文档和代码！🎉
