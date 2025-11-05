# 认证模块 RESTful API 设计

## 概述

本文档定义了认证模块 (authn) 对外提供的所有 RESTful API 接口。

## API 分组

### 1. 认证相关 API (Auth Handler)

#### 1.1 用户登录
```
POST /api/v1/authn/login
```

**功能**: 统一登录端点，支持多种认证方式

**认证方式支持**:
- `password` - 用户名密码登录
- `phone_otp` - 手机号验证码登录
- `wechat` - 微信小程序登录
- `wecom` - 企业微信登录

**请求体**:
```json
{
  "method": "password|phone_otp|wechat|wecom",
  "credentials": {
    // 根据 method 不同，credentials 结构不同
  }
}
```

**password 方式的 credentials**:
```json
{
  "username": "user@example.com",
  "password": "password123",
  "tenant_id": 1  // 可选
}
```

**phone_otp 方式的 credentials**:
```json
{
  "phone": "+8613800138000",
  "otp_code": "123456"
}
```

**wechat 方式的 credentials**:
```json
{
  "app_id": "wx1234567890",
  "code": "js_code_from_wechat"
}
```

**wecom 方式的 credentials**:
```json
{
  "corp_id": "corp123",
  "auth_code": "auth_code_from_wecom"
}
```

**响应**:
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "uuid-refresh-token"
}
```

#### 1.2 用户登出
```
POST /api/v1/authn/logout
```

**功能**: 撤销访问令牌和刷新令牌

**请求体**:
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",  // 可选
  "refresh_token": "uuid-refresh-token"       // 可选
}
```

**响应**:
```json
{
  "message": "Logout successful"
}
```

#### 1.3 刷新访问令牌
```
POST /api/v1/authn/token/refresh
```

**功能**: 使用刷新令牌获取新的访问令牌

**请求体**:
```json
{
  "refresh_token": "uuid-refresh-token"
}
```

**响应**:
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "new-uuid-refresh-token"
}
```

#### 1.4 验证访问令牌
```
POST /api/v1/authn/token/verify
```

**功能**: 验证访问令牌的有效性并返回声明信息

**请求体**:
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs..."
}
```

**响应**:
```json
{
  "valid": true,
  "claims": {
    "user_id": "1234567890",
    "account_id": "9876543210",
    "tenant_id": 1,
    "issuer": "iam-apiserver",
    "issued_at": 1699200000,
    "expires_at": 1699200900
  }
}
```

#### 1.5 撤销访问令牌
```
DELETE /api/v1/authn/token
```

**功能**: 撤销指定的访问令牌使其立即失效

**请求体**:
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs..."
}
```

**响应**:
```json
{
  "message": "Token revoked successfully"
}
```

#### 1.6 撤销刷新令牌
```
DELETE /api/v1/authn/token/refresh
```

**功能**: 撤销指定的刷新令牌使其立即失效

**请求体**:
```json
{
  "refresh_token": "uuid-refresh-token"
}
```

**响应**:
```json
{
  "message": "Refresh token revoked successfully"
}
```

---

### 2. 账户管理 API (Account Handler)

#### 2.1 微信账户注册
```
POST /api/v1/authn/accounts/register/wechat
```

**功能**: 微信用户注册或绑定账户

**请求体**:
```json
{
  "name": "张三",
  "phone": "+8613800138000",
  "email": "user@example.com",  // 可选
  "app_id": "wx1234567890",
  "open_id": "oABC123XYZ",
  "union_id": "uABC123XYZ",      // 可选
  "nickname": "微信昵称",         // 可选
  "avatar": "https://...",       // 可选
  "meta": {                      // 可选
    "custom_field": "value"
  }
}
```

**响应**:
```json
{
  "user_id": "1234567890",
  "user_name": "张三",
  "phone": "+8613800138000",
  "email": "user@example.com",
  "account_id": "9876543210",
  "account_type": "wechat",
  "external_id": "oABC123XYZ",
  "credential_id": 100,
  "is_new_user": true,
  "is_new_account": true
}
```

#### 2.2 获取账户信息
```
GET /api/v1/authn/accounts/:accountId
```

**功能**: 根据账户 ID 获取账户详细信息

**响应**:
```json
{
  "account_id": "9876543210",
  "user_id": "1234567890",
  "type": "wechat",
  "app_id": "wx1234567890",
  "external_id": "oABC123XYZ",
  "unique_id": "uABC123XYZ",
  "profile": {
    "nickname": "微信昵称",
    "avatar": "https://..."
  },
  "meta": {
    "custom_field": "value"
  },
  "status": "active"
}
```

#### 2.3 更新账户资料（待实现）
```
PUT /api/v1/authn/accounts/:accountId/profile
```

**功能**: 更新账户的 profile 信息

#### 2.4 设置 UnionID（待实现）
```
PUT /api/v1/authn/accounts/:accountId/unionid
```

**功能**: 为账户设置微信 UnionID

#### 2.5 禁用账户（待实现）
```
DELETE /api/v1/authn/accounts/:accountId
```

**功能**: 禁用指定账户

#### 2.6 获取凭证列表（待实现）
```
GET /api/v1/authn/accounts/:accountId/credentials
```

**功能**: 获取账户的所有认证凭证

---

### 3. JWKS 管理 API (JWKS Handler)

#### 3.1 获取 JWKS（公开端点）
```
GET /.well-known/jwks.json
```

**功能**: 获取 JSON Web Key Set，用于验证 JWT 签名

**响应头**:
- `ETag`: 实体标签
- `Last-Modified`: 最后修改时间
- `Cache-Control`: 缓存控制（public, max-age=3600）

**响应**:
```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "kid": "key-id-123",
      "n": "modulus...",
      "e": "AQAB"
    }
  ]
}
```

#### 3.2 创建密钥（管理员）
```
POST /api/v1/authn/jwks/keys
```

**功能**: 创建新的签名密钥

**请求体**:
```json
{
  "key_type": "RSA",
  "key_size": 2048
}
```

#### 3.3 列出密钥（管理员）
```
GET /api/v1/authn/jwks/keys
```

**功能**: 列出所有密钥及其状态

#### 3.4 获取密钥详情（管理员）
```
GET /api/v1/authn/jwks/keys/:kid
```

**功能**: 获取指定密钥的详细信息

#### 3.5 激活密钥（管理员）
```
PUT /api/v1/authn/jwks/keys/:kid/activate
```

**功能**: 激活指定密钥用于签名

#### 3.6 撤销密钥（管理员）
```
PUT /api/v1/authn/jwks/keys/:kid/revoke
```

**功能**: 撤销指定密钥，不再用于签名

#### 3.7 删除密钥（管理员）
```
DELETE /api/v1/authn/jwks/keys/:kid
```

**功能**: 删除指定密钥

#### 3.8 手动触发轮换（管理员）
```
POST /api/v1/authn/jwks/rotation/trigger
```

**功能**: 手动触发密钥轮换流程

---

## 错误响应格式

所有 API 的错误响应统一格式：

```json
{
  "code": 400001,
  "message": "Invalid argument: username is required",
  "reference": "https://docs.example.com/errors/400001"
}
```

常见错误码：
- `400000` - 请求参数错误
- `401000` - 认证失败
- `403000` - 权限不足
- `404000` - 资源不存在
- `500000` - 服务器内部错误

---

## 中间件

### JWT 验证中间件

用于保护需要认证的 API 端点。

**使用方式**:
```go
router.Use(middleware.JWTAuth(tokenService))
```

**功能**:
1. 从 `Authorization` 头提取 Bearer token
2. 调用 `TokenApplicationService.VerifyToken` 验证令牌
3. 将用户信息注入 Gin Context：
   - `user_id`
   - `account_id`
   - `tenant_id`
4. 验证失败返回 401

---

## 实现状态

### ✅ 已实现
- [x] AuthHandler 基础结构
- [x] 登录 API (Login)
- [x] 登出 API (Logout)
- [x] 刷新令牌 API (RefreshToken)
- [x] 验证令牌 API (VerifyToken)
- [x] 撤销令牌 API (RevokeToken, RevokeRefreshToken)
- [x] AccountHandler 基础结构
- [x] 微信注册 API (RegisterWithWeChat)
- [x] 获取账户信息 API (GetAccountByID)
- [x] JWKSHandler 基础结构
- [x] 获取 JWKS 公开端点

### 🚧 待补充
- [ ] Request/Response DTO 完善
- [ ] 账户更新相关 API
- [ ] JWKS 管理 API 完整实现
- [ ] JWT 验证中间件
- [ ] API 单元测试
- [ ] Swagger 文档注解完善

---

## 下一步工作

1. 补充缺失的 Request/Response 类型定义
2. 实现 JWT 验证中间件
3. 配置路由器绑定 Handler 方法
4. 添加单元测试
5. 完善 Swagger 文档
