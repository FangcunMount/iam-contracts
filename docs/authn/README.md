# 认证中心 (AuthN) 架构设计

> 负责用户身份认证、JWT Token 管理、JWKS 公钥发布等核心能力

---

## 📚 文档导航

| 文档 | 说明 | 内容 |
|------|------|------|
| **本文档** | 架构概述 | 设计目标、核心职责、技术特性 |
| [目录结构](./DIRECTORY_STRUCTURE.md) | 代码组织 | 分层架构、端口适配器、设计模式 |
| [认证流程](./AUTHENTICATION_FLOWS.md) | 流程详解 | 微信登录、Token刷新、验证流程 |
| [Token 管理](./TOKEN_MANAGEMENT.md) | Token 生命周期 | 签发、刷新、撤销、密钥轮换 |
| [JWKS 指南](./JWKS_GUIDE.md) | 公钥集发布 | JWKS 标准、密钥轮换、业务集成 |
| [安全设计](./SECURITY_DESIGN.md) | 安全机制 | 密码加密、防重放、速率限制 |
| [API 参考](./API_REFERENCE.md) | 接口文档 | REST API、集成方案、客户端示例 |

---

## 1. 模块概述

认证中心（Authentication Center, Authn）是 IAM 平台的核心模块，负责用户身份验证和访问令牌管理。

### 1.1 设计目标

- ✅ **多渠道登录**: 支持微信、企业微信、本地密码等多种认证方式
- ✅ **JWT 标准**: 基于 RFC 7519 标准签发和验证 Token
- ✅ **JWKS 支持**: 公钥集发布，支持业务服务自验证
- ✅ **Token 刷新**: Refresh Token 机制，提升用户体验
- ✅ **会话管理**: Redis 存储活跃会话，支持强制登出

### 1.2 技术特性

| 特性 | 实现方式 |
|------|---------|
| **JWT 签名** | RS256 (RSA 非对称加密) |
| **密钥管理** | 定期轮换，支持多密钥并存 |
| **Token 存储** | Redis + 黑名单机制 |
| **密码加密** | BCrypt 哈希 |
| **防重放攻击** | Nonce + 时间戳验证 |

---

## 2. 核心职责

### 2.1 身份认证

支持多种认证方式：

- **微信小程序登录**: code2session + unionid
- **企业微信登录**: 企业身份验证
- **本地密码登录**: 用户名/手机号/邮箱 + 密码
- **第三方 OAuth**: 预留扩展接口

### 2.2 Token 签发

- **Access Token**: 短期有效（15分钟），用于 API 访问
- **Refresh Token**: 长期有效（7天），用于刷新 Access Token

### 2.3 JWKS 公钥发布

- 发布 JWKS 端点 `/.well-known/jwks.json`
- 支持业务服务本地验证 Token
- 自动密钥轮换机制

### 2.4 会话管理

- Redis 存储活跃会话
- 支持多设备登录
- 支持强制登出

---

## 3. 架构模式

### 3.1 六边形架构

```text
┌─────────────────────────────────────────────┐
│         Interface Layer (接口层)             │
│      REST API / gRPC / Event Handler        │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│       Application Layer (应用层)             │
│    LoginService / TokenService / JWKS       │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│          Domain Layer (领域层)               │
│  Account / Authentication / JWT / JWKS      │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│     Infrastructure Layer (基础设施层)        │
│  MySQL / Redis / WeChat SDK / JWT库         │
└─────────────────────────────────────────────┘
```

### 3.2 DDD 领域模型

**聚合根**:

- **Account**: 账户聚合，管理用户认证信息
- **Authentication**: 认证聚合，处理认证逻辑
- **JWKS**: 密钥聚合，管理公钥发布

**值对象**:

- UserID, Provider, ExternalID
- Token (Access/Refresh)
- PublicJWK, KeyStatus

**领域服务**:

- Authenticator (认证器策略)
- TokenIssuer (Token 签发)
- TokenVerifier (Token 验证)
- KeyRotation (密钥轮换)

### 3.3 CQRS 模式

- **命令**: CreateAccount, Login, RefreshToken, RevokeToken
- **查询**: GetAccount, VerifyToken, GetJWKS

---

## 4. 技术栈

| 层次 | 技术 | 用途 |
|------|------|------|
| **语言** | Go 1.21+ | 高性能、并发支持 |
| **Web 框架** | Gin | HTTP 服务器 |
| **ORM** | GORM | 数据库操作 |
| **数据库** | MySQL 8.0 | 持久化存储 |
| **缓存** | Redis 7.0 | Token 存储、会话管理 |
| **JWT** | golang-jwt/jwt | Token 签发验证 |
| **密码** | bcrypt | 密码哈希 |

---

## 5. 快速开始

### 5.1 微信小程序登录

```bash
# 1. 获取微信 code
# 2. 调用登录接口
curl -X POST https://api.example.com/api/v1/auth/wechat:login \
  -H "Content-Type: application/json" \
  -d '{
    "code": "051Ab2ll2QMRCH05o2nl2vhOX64Ab2lx",
    "device_id": "iPhone13_iOS16"
  }'

# 响应
{
  "access_token": "eyJhbGci...",
  "refresh_token": "eyJhbGci...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

### 5.2 使用 Token 访问 API

```bash
curl -X GET https://api.example.com/api/v1/users/me \
  -H "Authorization: Bearer eyJhbGci..."
```

### 5.3 刷新 Token

```bash
curl -X POST https://api.example.com/api/v1/auth/token:refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGci..."
  }'
```

---

## 6. 核心优势

1. **标准化**: 基于 JWT + JWKS 标准，易于集成
2. **高性能**: 本地 Token 验证，无需每次调用认证服务
3. **安全性**: RS256 签名、密钥轮换、黑名单机制
4. **多渠道**: 支持多种认证方式，易于扩展
5. **易集成**: 提供中间件和客户端 SDK

---

## 7. 下一步

- 📖 阅读 [目录结构](./DIRECTORY_STRUCTURE.md) 了解代码组织
- 🔄 阅读 [认证流程](./AUTHENTICATION_FLOWS.md) 理解业务流程
- 🔐 阅读 [Token 管理](./TOKEN_MANAGEMENT.md) 掌握 Token 生命周期
- 🛡️ 阅读 [安全设计](./SECURITY_DESIGN.md) 了解安全机制
- 🔌 阅读 [API 参考](./API_REFERENCE.md) 集成到业务系统

---

**维护者**: Authn Team  
**最后更新**: 2025-10-18  
**版本**: V2.0
