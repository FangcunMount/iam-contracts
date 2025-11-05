# Concepts & Glossary

> 快速定位系统中的核心概念、常用缩写与设计基石。

## 领域驱动设计（DDD）

- **Bounded Context**：界定业务范围，避免跨上下文耦合。
- **Aggregates**：聚合根负责维护业务一致性；示例：AuthN 的 `Authenticater` 聚合。

## 架构模式

- **Hexagonal Architecture**：核心领域通过端口（Ports）与适配器（Adapters）对外通信。
- **CQRS**：命令（Command）负责状态变更，查询（Query）负责数据读取。

## 多租户

- **Tenant**：租户是隔离的业务空间；数据库、缓存、身份标识均按租户隔离。
- **Tenant-aware Services**：所有入口需要显式提供 `TenantID`。

## 身份与凭证

- **Principal**：认证后的身份表示（UserID、AccountID、TenantID）。
- **Credential**：密码、OTP、OAuth 等登录凭据。
- **Token Pair**：`access_token` + `refresh_token`，由 Token Issuer 颁发。

## 授权模型

- **Policy**：授权规则（角色 → 资源 → 动作），通过 Casbin 管理。
- **JWKS**：JSON Web Key Set，用于 JWT 验签。

> TODO: 随着模块文档更新补充新的术语与引用链接。
