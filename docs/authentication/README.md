# 🔐 认证系统文档

完整的认证模块文档集合，基于 DDD（领域驱动设计）架构实现。

## 📚 文档导航

### 🎯 快速开始

1. **[快速参考](./authentication-quick-reference.md)** - API 速查表和常用命令
2. **[集成方案](./authentication-integration.md)** - 如何将认证系统集成到项目中
3. **[中间件指南](./authentication-middleware-guide.md)** - JWT 认证中间件详细使用

### 🏗️ 设计文档

- **[认证设计](./authentication-design.md)** - DDD 领域驱动设计，核心概念和模型

### 📖 实现文档

#### 完整实现

- **[实现总结](./authentication-implementation-summary.md)** - 完整的实现文档和架构说明

#### 分层总结

- **[服务层](./authentication-service-summary.md)** - Domain 层：认证器、令牌服务等
- **[基础设施层](./authentication-infrastructure-summary.md)** - Infrastructure 层：JWT、Redis、MySQL、WeChat 适配器
- **[应用层](./authentication-application-summary.md)** - Application 层：登录服务、令牌服务等

## 🔍 按主题查找

### 认证流程

- [认证设计 - 认证流程](./authentication-design.md#认证流程)
- [实现总结 - 认证流程](./authentication-implementation-summary.md#认证流程)

### 令牌管理

- [服务层 - TokenService](./authentication-service-summary.md#tokenservice)
- [基础设施层 - JWT 生成器](./authentication-infrastructure-summary.md#jwt-生成器)
- [基础设施层 - Redis 令牌存储](./authentication-infrastructure-summary.md#redis-令牌存储)

### 中间件使用

- [中间件指南 - 完整教程](./authentication-middleware-guide.md)
- [集成方案 - 中间件集成](./authentication-integration.md#使用中间件)

### API 参考

- [快速参考 - API 端点](./authentication-quick-reference.md#api-端点)
- [快速参考 - 使用示例](./authentication-quick-reference.md#使用示例)

## 📋 功能特性

### ✅ 已实现功能

- **多种认证方式**
  - 基础认证（用户名/密码）
  - 微信小程序认证
  - 可扩展的认证器架构

- **完整令牌管理**
  - JWT 访问令牌（15 分钟有效期）
  - UUID 刷新令牌（7 天有效期）
  - 刷新令牌轮换
  - 令牌黑名单机制

- **安全特性**
  - Bcrypt 密码加密
  - JWT 签名验证
  - 令牌过期检查
  - 多设备登出支持

- **中间件支持**
  - JWT 认证中间件
  - 可选认证中间件
  - 角色检查（待完善）
  - 权限检查（待完善）

### 🔄 待完善功能

查看各文档中的 TODO 部分：

- [实现总结 - 待实现功能](./authentication-implementation-summary.md#待实现功能)
- [快速参考 - TODO](./authentication-quick-reference.md#todo)

## 🚀 典型使用场景

### 场景 1：用户登录

```bash
# 1. 查看 API 文档
→ authentication-quick-reference.md

# 2. 实现登录接口
→ authentication-integration.md#用户登录

# 3. 使用中间件保护端点
→ authentication-middleware-guide.md#使用方法
```

### 场景 2：理解架构

```bash
# 1. 了解整体设计
→ authentication-design.md

# 2. 查看各层实现
→ authentication-service-summary.md
→ authentication-infrastructure-summary.md
→ authentication-application-summary.md

# 3. 查看完整实现
→ authentication-implementation-summary.md
```

### 场景 3：集成到项目

```bash
# 1. 查看集成方案
→ authentication-integration.md

# 2. 配置中间件
→ authentication-middleware-guide.md

# 3. 参考 API
→ authentication-quick-reference.md
```

## 📊 架构概览

```text
认证系统 (4 层 DDD 架构)
│
├── 🎯 Domain Layer (领域层)
│   ├── 认证器：BasicAuthenticator, WeChatAuthenticator
│   ├── 服务：AuthenticationService, TokenService
│   └── 实体：Authentication, Token, Credential
│
├── 🔧 Infrastructure Layer (基础设施层)
│   ├── JWT 生成器
│   ├── Redis 令牌存储
│   ├── MySQL 仓储
│   └── WeChat 认证适配器
│
├── 💼 Application Layer (应用层)
│   ├── LoginService (登录服务)
│   └── TokenService (令牌应用服务)
│
└── 🌐 Interface Layer (接口层)
    ├── RESTful Handler
    └── Request/Response DTO
```

详细架构说明请查看：

- [认证设计 - 架构设计](./authentication-design.md#架构设计)
- [实现总结 - 架构图](./authentication-implementation-summary.md#架构图)

## 🔗 相关资源

### 代码位置

- **源码**: `internal/apiserver/modules/authn/`
- **中间件**: `internal/pkg/middleware/authn/`
- **容器配置**: `internal/apiserver/container/assembler/auth.go`

### 配置文件

- **主配置**: `configs/apiserver.yaml`
- **Redis 配置**: Redis 连接和令牌存储
- **JWT 配置**: 密钥和过期时间

### API 规范

- **OpenAPI**: `api/openapi/authn.v1.yaml`

## 💡 最佳实践

1. **安全性**
   - 生产环境使用 HTTPS
   - 定期轮换 JWT 密钥
   - 设置合理的令牌过期时间
   - 使用 HttpOnly Cookie 存储刷新令牌

2. **性能优化**
   - 使用 Redis 缓存令牌
   - 合理设置连接池大小
   - 避免频繁的令牌刷新

3. **可维护性**
   - 遵循 DDD 分层架构
   - 保持各层职责清晰
   - 编写完善的单元测试

## 📝 文档更新日志

- **2024-10-14**: 完成认证系统迁移，整理文档结构
- **2024-10-13**: 添加中间件使用指南
- **2024-10-12**: 完成 DDD 架构设计文档
- **2024-10-11**: 创建认证系统基础文档

---

📌 **提示**: 如果你是新手，建议从 [快速参考](./authentication-quick-reference.md) 开始！
