# Swagger 文档验证报告

生成日期: 2025-11-05

## 概述

本报告对比了 Swagger 文档 (`internal/apiserver/docs/swagger.json`) 和实际路由注册代码，验证文档的完整性和准确性。

## ✅ 已完成的模块

### 1. 认证模块 (Authn)

#### 认证端点 (Auth Handler)

| 端点 | 方法 | Swagger路径 | 实际路由 | 状态 |
|------|------|------------|----------|------|
| 用户登录 | POST | `/api/v1/auth/login` | `/api/v1/auth/login` | ✅ 匹配 |
| 用户登出 | POST | `/api/v1/auth/logout` | `/api/v1/auth/logout` | ✅ 匹配 |
| 刷新令牌 | POST | `/api/v1/auth/refresh_token` | `/api/v1/auth/refresh_token` | ✅ 匹配 |
| 验证令牌 | POST | `/api/v1/auth/verify` | `/api/v1/auth/verify` | ✅ 匹配 |

**说明**: 

- Login 端点支持多种登录方式：密码、手机验证码、微信小程序、企业微信
- 所有端点的 Swagger 注解完整，包含 Summary、Description、Tags、Parameters、Responses


#### Account Handler

- `PUT /api/v1/accounts/{accountId}/unionid` - SetUnionID

**说明**: 

**说明**:

- 核心端点已添加 Swagger 注解
- 部分管理端点（SetUnionID、Enable、Disable）未添加 Swagger 注解，建议后续补充

#### JWKS 端点 (JWKS Handler)

| 端点 | 方法 | Swagger路径 | 实际路由 | 状态 |
|------|------|------------|----------|------|
| 获取公钥集 | GET | `/.well-known/jwks.json` | `/.well-known/jwks.json` | ✅ 匹配 |
| 创建密钥 | POST | `/api/v1/admin/jwks/keys` | `/api/v1/admin/jwks/keys` | ✅ 匹配 |
| 列出密钥 | GET | `/api/v1/admin/jwks/keys` | `/api/v1/admin/jwks/keys` | ✅ 匹配 |
| 获取密钥详情 | GET | `/api/v1/admin/jwks/keys/{kid}` | `/api/v1/admin/jwks/keys/:kid` | ✅ 匹配 |
| 退役密钥 | POST | `/api/v1/admin/jwks/keys/{kid}/retire` | `/api/v1/admin/jwks/keys/:kid/retire` | ✅ 匹配 |
| 强制退役密钥 | POST | `/api/v1/admin/jwks/keys/{kid}/force-retire` | `/api/v1/admin/jwks/keys/:kid/force-retire` | ✅ 匹配 |
| 进入宽限期 | POST | `/api/v1/admin/jwks/keys/{kid}/grace` | `/api/v1/admin/jwks/keys/:kid/grace` | ✅ 匹配 |
| 清理过期密钥 | POST | `/api/v1/admin/jwks/keys/cleanup` | `/api/v1/admin/jwks/keys/cleanup` | ✅ 匹配 |
| 获取可发布密钥 | GET | `/api/v1/admin/jwks/keys/publishable` | `/api/v1/admin/jwks/keys/publishable` | ✅ 匹配 |

**说明**: 

- JWKS 模块完整性高，所有端点均有 Swagger 文档
- 符合 OAuth 2.0/OIDC 标准的 JWKS 端点规范

### 2. 授权模块 (Authz)

#### 角色管理 (Role Handler)

| 端点 | 方法 | Swagger路径 | 实际路由 | 状态 |
|------|------|------------|----------|------|
| 创建角色 | POST | `/authz/roles` | `/authz/roles` | ✅ 匹配 |
| 更新角色 | PUT | `/authz/roles/{id}` | `/authz/roles/{id}` | ✅ 匹配 |
| 删除角色 | DELETE | `/authz/roles/{id}` | `/authz/roles/{id}` | ✅ 匹配 |
| 获取角色 | GET | `/authz/roles/{id}` | `/authz/roles/{id}` | ✅ 匹配 |
| 列出角色 | GET | `/authz/roles` | `/authz/roles` | ✅ 匹配 |

#### 权限分配 (Assignment Handler)

| 端点 | 方法 | Swagger路径 | 实际路由 | 状态 |
|------|------|------------|----------|------|
| 授予权限 | POST | `/authz/assignments/grant` | `/authz/assignments/grant` | ✅ 匹配 |
| 撤销权限 | POST | `/authz/assignments/revoke` | `/authz/assignments/revoke` | ✅ 匹配 |
| 删除分配 | DELETE | `/authz/assignments/{id}` | `/authz/assignments/{id}` | ✅ 匹配 |
| 获取主体权限 | GET | `/authz/assignments/subject` | `/authz/assignments/subject` | ✅ 匹配 |
| 获取角色分配 | GET | `/authz/roles/{id}/assignments` | `/authz/roles/{id}/assignments` | ✅ 匹配 |

#### 策略管理 (Policy Handler)

| 端点 | 方法 | Swagger路径 | 实际路由 | 状态 |
|------|------|------------|----------|------|
| 添加策略 | POST | `/authz/policies` | `/authz/policies` | ✅ 匹配 |
| 删除策略 | DELETE | `/authz/policies` | `/authz/policies` | ✅ 匹配 |
| 获取角色策略 | GET | `/authz/roles/{id}/policies` | `/authz/roles/{id}/policies` | ✅ 匹配 |
| 获取策略版本 | GET | `/authz/policies/version` | `/authz/policies/version` | ✅ 匹配 |

#### 资源管理 (Resource Handler)

| 端点 | 方法 | Swagger路径 | 实际路由 | 状态 |
|------|------|------------|----------|------|
| 创建资源 | POST | `/authz/resources` | `/authz/resources` | ✅ 匹配 |
| 更新资源 | PUT | `/authz/resources/{id}` | `/authz/resources/{id}` | ✅ 匹配 |
| 删除资源 | DELETE | `/authz/resources/{id}` | `/authz/resources/{id}` | ✅ 匹配 |
| 获取资源 | GET | `/authz/resources/{id}` | `/authz/resources/{id}` | ✅ 匹配 |
| 按Key获取资源 | GET | `/authz/resources/key/{key}` | `/authz/resources/key/{key}` | ✅ 匹配 |
| 列出资源 | GET | `/authz/resources` | `/authz/resources` | ✅ 匹配 |
| 验证操作 | POST | `/authz/resources/validate-action` | `/authz/resources/validate-action` | ✅ 匹配 |

**说明**: 授权模块文档完整，基于 Casbin 的 RBAC 授权模型

### 3. 用户中心模块 (UC)

#### 用户管理 (User Handler)

| 端点 | 方法 | Swagger路径 | 实际路由 | 状态 |
|------|------|------------|----------|------|
| 获取用户资料 | GET | `/users/profile` | `/users/profile` | ✅ 匹配 |
| 更新用户信息 | PATCH | `/users/{userId}` | `/users/{userId}` | ✅ 匹配 |

#### 儿童管理 (Child Handler)

| 端点 | 方法 | Swagger路径 | 实际路由 | 状态 |
|------|------|------------|----------|------|
| 获取我的儿童 | GET | `/me/children` | `/me/children` | ✅ 匹配 |
| 儿童注册 | POST | `/children/register` | `/children/register` | ✅ 匹配 |
| 创建儿童 | POST | `/children` | `/children` | ✅ 匹配 |
| 获取儿童详情 | GET | `/children/{id}` | `/children/{id}` | ✅ 匹配 |
| 更新儿童信息 | PATCH | `/children/{id}` | `/children/{id}` | ✅ 匹配 |
| 搜索儿童 | GET | `/children/search` | `/children/search` | ✅ 匹配 |

#### 监护关系 (Guardianship Handler)

| 端点 | 方法 | Swagger路径 | 实际路由 | 状态 |
|------|------|------------|----------|------|
| 授予监护权 | POST | `/guardians/grant` | `/guardians/grant` | ✅ 匹配 |
| 撤销监护权 | POST | `/guardians/revoke` | `/guardians/revoke` | ✅ 匹配 |
| 获取监护关系 | GET | `/guardians` | `/guardians` | ✅ 匹配 |

**说明**: UC 模块完整性高，支持家庭教育场景的儿童和监护关系管理

### 4. 身份提供者模块 (IDP)

#### 微信应用管理 (WechatApp Handler)

| 端点 | 方法 | Swagger路径 | 实际路由 | 状态 |
|------|------|------------|----------|------|
| 创建微信应用 | POST | `/idp/wechat-apps` | `/idp/wechat-apps` | ✅ 匹配 |
| 获取微信应用 | GET | `/idp/wechat-apps/{app_id}` | `/idp/wechat-apps/{app_id}` | ✅ 匹配 |
| 轮换认证密钥 | POST | `/idp/wechat-apps/rotate-auth-secret` | `/idp/wechat-apps/rotate-auth-secret` | ✅ 匹配 |
| 轮换消息密钥 | POST | `/idp/wechat-apps/rotate-msg-secret` | `/idp/wechat-apps/rotate-msg-secret` | ✅ 匹配 |
| 获取访问令牌 | GET | `/idp/wechat-apps/{app_id}/access-token` | `/idp/wechat-apps/{app_id}/access-token` | ✅ 匹配 |
| 刷新访问令牌 | POST | `/idp/wechat-apps/refresh-access-token` | `/idp/wechat-apps/refresh-access-token` | ✅ 匹配 |

#### 微信认证

- IDP 模块的微信认证端点（`/idp/wechat/*`）已下线，Swagger 文档与实际路由保持一致，不再暴露这些路径。
- 微信登录统一由 Authn 模块的 `/api/v1/auth/login` 端点提供，客户端需要在请求体中将 `method` 设置为 `wx:minip`。

## ⚠️ 发现的问题

### 1. 缺少 Swagger 注解的端点

以下端点已在路由中注册，但缺少 Swagger 注解：

#### Account Handler

- `PUT /api/v1/accounts/{accountId}/unionid` - SetUnionID
- `POST /api/v1/accounts/{accountId}/enable` - EnableAccount
- `POST /api/v1/accounts/{accountId}/disable` - DisableAccount

**建议**: 为这些端点添加 Swagger 注解，提高文档完整性

### 2. 端点路径格式

- Swagger 使用 `{param}` 格式表示路径参数
- Gin 路由使用 `:param` 格式
- 两者可以正确映射，无需修改

## 📊 统计数据

| 模块 | 端点总数 | 已文档化 | 完整性 |
|------|---------|---------|--------|
| Authn - Auth | 4 | 4 | 100% |
| Authn - Account | 7 | 4 | 57% |
| Authn - JWKS | 9 | 9 | 100% |
| Authz - Role | 5 | 5 | 100% |
| Authz - Assignment | 5 | 5 | 100% |
| Authz - Policy | 4 | 4 | 100% |
| Authz - Resource | 7 | 7 | 100% |
| UC - User | 2 | 2 | 100% |
| UC - Child | 6 | 6 | 100% |
| UC - Guardianship | 3 | 3 | 100% |
| IDP - WechatApp | 6 | 6 | 100% |
| **总计** | **58** | **55** | **95%** |

## ✅ 验证结论

1. **核心功能完整**: 认证、授权、用户管理、IDP 等核心功能的主要端点都已有完整的 Swagger 文档
2. **文档质量高**: 已文档化的端点包含完整的 Summary、Description、Tags、Parameters、Responses
3. **少量遗漏**: Account Handler 的 3 个管理端点缺少文档，建议后续补充
4. **架构清晰**: 端点按模块分组，符合 DDD 分层架构

## 🔧 后续改进建议

### 优先级 P1（建议立即完成）

- 暂无

### 优先级 P2（可以后续完成）

- [ ] 为 Account Handler 的 3 个管理端点添加 Swagger 注解
  - SetUnionID
  - EnableAccount
  - DisableAccount

### 优先级 P3（长期维护）

- [ ] 建立 CI/CD 流程，在代码提交时自动生成和验证 Swagger 文档
- [ ] 添加 Swagger 文档的自动化测试，确保与实际路由保持同步

## 📝 验证方法

本次验证采用以下方法：

1. 使用 `grep_search` 工具搜索所有 handler 文件中的 `@Router` 注解
2. 对比 `internal/apiserver/routers.go` 和各模块的 `router.go` 中的路由注册
3. 检查 `swagger.json` 中的路径定义
4. 交叉验证端点的存在性和一致性

---

**报告生成时间**: 2025-11-05 19:44  
**验证工具**: GitHub Copilot + swag v1.8.12  
**验证范围**: internal/apiserver/interface/*/restful/handler/*.go
