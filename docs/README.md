# IAM Contracts 文档中心# iam contracts 使用说明

欢迎来到 IAM Contracts 项目文档中心！这里包含了项目的完整架构设计、开发指南和最佳实践。本文档详细介绍了基于六边形架构的Go Web框架的设计理念、核心组件和使用方法。

## 📚 文档目录## 📚 IAM Contracts 文档中心

### 核心架构文档欢迎来到 IAM Contracts 项目文档中心

1. **[IAM 项目架构说明](./architecture-overview.md)** ⭐## 🗂️ 文档目录

   - 项目概述与设计目标
   - 六边形架构 + DDD + CQRS### 🏗️ 架构设计
   - 技术栈与目录结构
   - 模块划分与依赖关系位置：[architecture/](./architecture/)
   - 数据流转与部署架构
   - 开发指南与最佳实践- [框架概览](./architecture/framework-overview.md) - 整体架构设计- [六边形容器](./architecture/hexagonal-container.md) - DDD 容器化架构

2. **[用户中心架构设计](./uc-architecture.md)** ⭐- [代码结构](./architecture/code-structure-apiserver.md) - API Server 代码组织

   - UC 模块完整设计- [项目结构](./architecture/project-structure.md) - 目录结构说明
   - 领域模型（User, Child, Guardianship）
   - CQRS 命令查询分离实现### 🔐 认证系统
   - 四层架构详解（Interface, Application, Domain, Infrastructure）
   - RESTful + gRPC API 设计位置：[authentication/](./authentication/)
   - 数据模型与业务流程

#### 设计文档

1. **[认证中心架构设计](./authn-architecture.md)** ⭐

   - Authn 模块完整设计- [认证设计](./authentication/authentication-design.md) - DDD 领域驱动设计
   - 多渠道登录（微信、企业微信、本地密码）- [实现总结](./authentication/authentication-implementation-summary.md) - 完整实现文档
   - JWT Token 管理（签发、验证、刷新、撤销）
   - JWKS 公钥发布与密钥轮换#### 分层总结
   - 安全设计（密码哈希、防重放、速率限制）
   - 业务服务集成方案- [服务层总结](./authentication/authentication-service-summary.md) - Domain 层设计

- [基础设施层](./authentication/authentication-infrastructure-summary.md) - 基础设施适配器

---- [应用层](./authentication/authentication-application-summary.md) - 应用服务

## 🚀 快速开始#### 使用指南

### 新人指引- [中间件指南](./authentication/authentication-middleware-guide.md) - JWT 认证中间件详解

- [集成方案](./authentication/authentication-integration.md) - 快速集成指南

如果你是第一次接触本项目，建议按以下顺序阅读：- [快速参考](./authentication/authentication-quick-reference.md) - API 速查表

1. **阅读 [架构总览](./architecture-overview.md)**### 🔧 系统功能

   - 了解项目整体架构
   - 熟悉技术栈和目录结构- [数据库注册](./database-registry.md) - 多数据库管理
   - 理解核心设计原则- [错误处理](./error-handling.md) - 统一错误处理机制
   - [日志系统](./logging-system.md) - 结构化日志方案

2. **学习 [用户中心设计](./uc-architecture.md)**

   - 掌握 DDD 建模方法### � IAM 规范
   - 理解 CQRS 模式应用
   - 熟悉代码分层结构位置：[iam/](./iam/)

3. **了解 [认证中心设计](./authn-architecture.md)**- IAM 相关的规范和标准文档

   - 理解 JWT 认证流程
   - 学习安全最佳实践## 🚀 快速开始
   - 掌握业务集成方法

4. **新手入门**: 从 [框架概览](./architecture/framework-overview.md) 开始

### 开发环境搭建

1. **认证开发**: 查看 [认证集成方案](./authentication/authentication-integration.md)

2. **API 参考**: 使用 [快速参考手册](./authentication/authentication-quick-reference.md)

```bash

# 1. 克隆项目## 📖 文档结构

git clone https://github.com/fangcun-mount/iam-contracts.git

cd iam-contracts```text

docs/

# 2. 启动基础设施（MySQL + Redis）├── README.md                    # 📚 本文档（文档索引）

cd build/docker/infra├── architecture/                # 🏗️ 架构设计文档

docker-compose up -d│   ├── framework-overview.md

│   ├── hexagonal-container.md

# 3. 安装依赖│   ├── code-structure-apiserver.md

go mod download│   └── project-structure.md

├── authentication/              # 🔐 认证系统文档

# 4. 运行数据库迁移│   ├── authentication-design.md

# TODO: 添加迁移命令│   ├── authentication-implementation-summary.md

│   ├── authentication-service-summary.md

# 5. 启动 API Server│   ├── authentication-infrastructure-summary.md

make run│   ├── authentication-application-summary.md

│   ├── authentication-middleware-guide.md

# 或使用热重载│   ├── authentication-integration.md

air│   └── authentication-quick-reference.md

```├── database-registry.md         # 🗄️ 数据库注册

├── error-handling.md            # ⚠️ 错误处理

### 运行测试├── logging-system.md            # 📝 日志系统

└── iam/                         # 📋 IAM 规范

```bash```

# 单元测试

go test ./...## 🔄 最近更新



# 集成测试- ✅ 完成认证系统迁移：从旧的 `middleware/auth` 迁移到新的 DDD 架构

go test -tags=integration ./...- ✅ 整理文档结构：按主题组织，便于查找

- ✅ 删除废弃文档：移除迁移过程中的临时文档

# 测试覆盖率

go test -cover ./...## 💡 贡献指南

```

更新文档时请遵循以下原则：

---

1. **按主题组织**: 将相关文档放在对应的子目录
2. **保持更新**: 代码变更时同步更新文档
3. **清晰简洁**: 使用清晰的标题和示例
4. **添加索引**: 在本文档中添加新文档的链接

| 文档 | 内容 | 适合人群 |## � 获取帮助

|------|------|---------|

| [architecture-overview.md](./architecture-overview.md) | 项目整体架构、技术选型、部署方案 | 所有人 |- 查看具体文档了解详细信息

| [uc-architecture.md](./uc-architecture.md) | 用户中心详细设计、领域模型、API | 后端开发 |- 查阅代码中的注释

| [authn-architecture.md](./authn-architecture.md) | 认证中心详细设计、JWT、安全 | 后端/客户端开发 |- 参考 `examples/` 目录中的示例

### 开发文档（待补充）

- [ ] `api-reference.md` - API 接口文档
- [ ] `database-schema.md` - 数据库设计文档
- [ ] `deployment-guide.md` - 部署运维指南
- [ ] `testing-guide.md` - 测试规范与用例
- [ ] `troubleshooting.md` - 常见问题排查

### 业务文档（待补充）

- [ ] `business-requirements.md` - 业务需求文档
- [ ] `user-manual.md` - 用户使用手册
- [ ] `admin-manual.md` - 管理员手册

---

## 🎯 核心概念速查

### 六边形架构

```text
External World (REST/gRPC/Event)
    ↓ Primary Adapters
Application Layer (Use Cases)
    ↓
Domain Layer (Business Logic)
    ↑ Secondary Adapters
Infrastructure (MySQL/Redis/External Services)
```

### CQRS 模式

```text
命令（Command）              查询（Query）
- 写操作                    - 读操作
- 业务规则验证              - 最小验证
- 事务管理                  - 可缓存
- ApplicationService        - QueryApplicationService
```

### DDD 战术设计

- **聚合根（Aggregate Root）**: User, Child, Guardianship
- **实体（Entity）**: 具有唯一标识的领域对象
- **值对象（Value Object）**: Phone, Email, IDCard, Birthday
- **领域服务（Domain Service）**: 跨实体的业务逻辑
- **仓储（Repository）**: 聚合的持久化接口

### 分层职责

| 层次 | 职责 | 依赖方向 |
|------|------|---------|
| **Interface** | HTTP/gRPC 适配器，DTO 转换 | → Application |
| **Application** | 用例编排，事务边界，DTO 转换 | → Domain |
| **Domain** | 业务逻辑，领域规则（核心） | 无依赖 |
| **Infrastructure** | 数据库、缓存、外部服务 | 实现 Domain Ports |

---

## 🔧 开发规范

### 代码风格

- 遵循 [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- 使用 `golangci-lint` 进行代码检查
- 函数名使用驼峰命名
- 包名使用小写单数形式

### 提交规范

```text
<type>(<scope>): <subject>

<body>

<footer>
```

**Type**:

- `feat`: 新功能
- `fix`: 修复 Bug
- `docs`: 文档更新
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建/工具相关

**示例**:

```text
feat(uc): add child profile update API

- Add UpdateProfile method to ChildProfileApplicationService
- Implement PATCH /api/v1/children/{id} endpoint
- Add integration tests

Closes #123
```

### 分支策略

```text
main (生产)
  ↑
develop (开发)
  ↑
feature/xxx (功能分支)
  ↑
hotfix/xxx (紧急修复)
```

---

## 📊 项目进度

### 已完成 ✅

- [x] 项目基础架构搭建
- [x] UC 模块（用户、儿童、监护关系）
- [x] CQRS 重构（命令查询分离）
- [x] DI 容器（依赖注入）
- [x] RESTful API（基础用户管理）
- [x] gRPC API（身份查询）
- [x] MySQL 仓储实现
- [x] 错误处理机制
- [x] 日志系统
- [x] 完整架构文档

### 进行中 🚧

- [ ] Authn 模块（认证中心）
  - [ ] 微信小程序登录
  - [ ] JWT 签发与验证
  - [ ] Token 刷新机制
  - [ ] JWKS 公钥发布
- [ ] Authz 模块（授权中心）
  - [ ] RBAC 权限模型
  - [ ] 关系授权判定
- [ ] 中间件
  - [ ] 认证中间件
  - [ ] 权限中间件
  - [ ] 日志中间件

### 计划中 📋

- [ ] 单元测试覆盖率 > 80%
- [ ] 集成测试
- [ ] API 文档（Swagger/OpenAPI）
- [ ] 性能测试与优化
- [ ] 监控告警系统
- [ ] CI/CD 流水线
- [ ] Docker 镜像构建
- [ ] Kubernetes 部署配置

---

## 🤝 贡献指南

我们欢迎所有形式的贡献！

### 如何贡献

1. **Fork 项目**
2. **创建功能分支**: `git checkout -b feature/amazing-feature`
3. **提交更改**: `git commit -m 'feat: add amazing feature'`
4. **推送分支**: `git push origin feature/amazing-feature`
5. **提交 Pull Request**

### 贡献类型

- 🐛 报告 Bug
- 💡 提出新功能建议
- 📝 改进文档
- 🔧 修复 Bug
- ✨ 实现新功能
- ✅ 添加测试
- 🎨 优化代码

### Code Review 标准

- ✅ 代码符合项目规范
- ✅ 包含必要的单元测试
- ✅ 更新相关文档
- ✅ 无明显性能问题
- ✅ 无安全隐患

---

## 📞 联系方式

### 团队

- **项目负责人**: IAM Team
- **架构师**: @architect
- **后端开发**: @backend-team
- **前端开发**: @frontend-team

### 沟通渠道

- **Issue Tracker**: [GitHub Issues](https://github.com/fangcun-mount/iam-contracts/issues)
- **讨论区**: [GitHub Discussions](https://github.com/fangcun-mount/iam-contracts/discussions)
- **邮件**: <iam-team@example.com>
- **文档**: 本仓库 `docs/` 目录

---

## 📜 许可证

本项目采用 [MIT License](../LICENSE) 许可证。

---

## 🔖 版本历史

| 版本 | 日期 | 内容 |
|------|------|------|
| v1.0.0 | 2025-10-17 | 初始版本，包含完整架构文档 |
| v0.5.0 | 2025-10-15 | UC 模块 CQRS 重构完成 |
| v0.3.0 | 2025-10-10 | UC 模块基础功能完成 |
| v0.1.0 | 2025-10-01 | 项目初始化 |

---

## 🎓 扩展阅读

### 架构模式

- [六边形架构](https://alistair.cockburn.us/hexagonal-architecture/)
- [领域驱动设计](https://domainlanguage.com/ddd/)
- [CQRS 模式](https://martinfowler.com/bliki/CQRS.html)
- [事件溯源](https://martinfowler.com/eaaDev/EventSourcing.html)

### Go 开发

- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### 微服务

- [微服务架构](https://microservices.io/)
- [API 网关模式](https://microservices.io/patterns/apigateway.html)
- [Service Mesh](https://istio.io/latest/docs/concepts/what-is-istio/)

---

**最后更新**: 2025-10-17  
**文档版本**: v1.0.0
