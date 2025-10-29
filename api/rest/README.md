# REST API 文档

> IAM Contracts RESTful API 规范（OpenAPI 3.1）

## 📋 文档列表

### 1. [认证 API (authn.v1.yaml)](./authn.v1.yaml)

**功能域**: 认证与账户管理

#### 核心端点

| 分组 | 端点 | 方法 | 说明 |
|------|------|------|------|
| **认证** | `/api/v1/auth/login` | POST | 用户登录（用户名密码/微信） |
| | `/api/v1/auth/refresh` | POST | 刷新访问令牌 |
| | `/api/v1/auth/verify` | POST | 验证令牌有效性 |
| | `/api/v1/auth/logout` | POST | 退出登录（撤销令牌） |
| **账户** | `/api/v1/accounts/operation` | POST | 创建运营账号 |
| | `/api/v1/accounts/operation/{username}` | PATCH | 更新运营口令 |
| | `/api/v1/accounts/wechat/bind` | POST | 绑定微信账号 |
| | `/api/v1/accounts/{accountId}` | GET | 查询账户信息 |
| | `/api/v1/accounts/by-ref` | GET | 通过引用查询账户 |
| **JWKS** | `/.well-known/jwks.json` | GET | 获取公钥集（用于 JWT 验签） |

#### 登录流程示例

**运营账号登录**:

```bash
curl -X POST https://api.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "account_type": "operation",
    "username": "admin",
    "password": "SecureP@ss123"
  }'
```

**响应**:

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 86400,
  "refresh_token": "def50200a1b2c3d4e5f6...",
  "scope": "read write",
  "user": {
    "id": "usr_1234567890",
    "username": "admin",
    "status": "active"
  }
}
```

**微信小程序登录**:

```bash
curl -X POST https://api.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "account_type": "wechat",
    "wechat": {
      "app_id": "wx1234567890abcdef",
      "code": "061XYZ..."
    }
  }'
```

#### 令牌管理

**刷新令牌**:

```bash
curl -X POST https://api.example.com/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "def50200a1b2c3d4e5f6..."
  }'
```

**验证令牌**:

```bash
curl -X POST https://api.example.com/api/v1/auth/verify \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**退出登录**:

```bash
curl -X POST https://api.example.com/api/v1/auth/logout \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
```

#### JWKS 公钥集

**获取公钥用于验签**:

```bash
curl -X GET https://api.example.com/.well-known/jwks.json
```

**响应** (符合 RFC 7517):

```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "kid": "2024-10-key-1",
      "alg": "RS256",
      "n": "xGOr-H7A...",
      "e": "AQAB"
    }
  ]
}
```

---

### 2. [身份 API (identity.v1.yaml)](./identity.v1.yaml)

**功能域**: 用户、儿童、监护关系管理

#### 身份管理核心端点

| 分组 | 端点 | 方法 | 说明 |
|------|------|------|------|
| **用户** | `/api/v1/users` | POST | 创建用户（管理员） |
| | `/api/v1/users/{userId}` | GET | 查询用户详情 |
| | `/api/v1/users/{userId}` | PATCH | 更新用户信息 |
| | `/api/v1/users/profile` | GET | 获取当前用户资料 |
| **儿童** | `/api/v1/children/register` | POST | 注册儿童（建档+授监护） |
| | `/api/v1/children` | POST | 仅建档（不授监护） |
| | `/api/v1/children/{childId}` | GET | 查询儿童档案 |
| | `/api/v1/children/{childId}` | PATCH | 更新儿童档案 |
| | `/api/v1/children/search` | GET | 搜索相似儿童 |
| | `/api/v1/me/children` | GET | 我的孩子列表 |
| **监护** | `/api/v1/guardians/grant` | POST | 授予监护关系 |
| | `/api/v1/guardians/revoke` | POST | 撤销监护关系 |
| | `/api/v1/guardians` | GET | 查询监护关系 |

#### 用户管理示例

**创建用户**:

```bash
curl -X POST https://api.example.com/api/v1/users \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "nickname": "张三",
    "status": "active"
  }'
```

**查询用户**:

```bash
curl -X GET https://api.example.com/api/v1/users/usr_1234567890 \
  -H "Authorization: Bearer <token>"
```

**响应**:

```json
{
  "id": "usr_1234567890",
  "nickname": "张三",
  "avatar": "https://cdn.example.com/avatars/usr_1234567890.jpg",
  "status": "active",
  "created_at": "2024-10-29T10:00:00Z",
  "updated_at": "2024-10-29T10:00:00Z"
}
```

#### 儿童档案管理

**注册儿童（推荐方式，自动建立监护关系）**:

```bash
curl -X POST https://api.example.com/api/v1/children/register \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: uuid-12345678-90ab-cdef-1234-567890abcdef" \
  -d '{
    "legal_name": "小明",
    "gender": 1,
    "dob": "2020-05-15",
    "id_type": "id_card",
    "id_card": "110101202005150012",
    "relation": "parent"
  }'
```

**响应**:

```json
{
  "child": {
    "id": "chd_9876543210",
    "legal_name": "小明",
    "gender": 1,
    "dob": "2020-05-15",
    "id_type": "id_card",
    "id_masked": "1101012020051***12",
    "created_at": "2024-10-29T11:00:00Z"
  },
  "guardianship": {
    "id": 12345,
    "user_id": "usr_1234567890",
    "child_id": "chd_9876543210",
    "relation": "parent",
    "since": "2024-10-29T11:00:00Z"
  }
}
```

**查询我的孩子**:

```bash
curl -X GET https://api.example.com/api/v1/me/children?limit=20&offset=0 \
  -H "Authorization: Bearer <token>"
```

**响应**:

```json
{
  "total": 2,
  "items": [
    {
      "id": "chd_9876543210",
      "legal_name": "小明",
      "gender": 1,
      "dob": "2020-05-15",
      "id_masked": "1101012020051***12",
      "height_cm": 105,
      "weight_kg": "18.5"
    },
    {
      "id": "chd_1111111111",
      "legal_name": "小红",
      "gender": 2,
      "dob": "2021-03-20",
      "id_masked": "1101012021032***45"
    }
  ]
}
```

**搜索相似儿童（防重复建档）**:

```bash
curl -X GET "https://api.example.com/api/v1/children/search?legal_name=小明&gender=1&dob=2020-05-15" \
  -H "Authorization: Bearer <token>"
```

#### 监护关系管理

**授予监护关系**:

```bash
curl -X POST https://api.example.com/api/v1/guardians/grant \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "usr_0987654321",
    "child_id": "chd_9876543210",
    "relation": "guardian"
  }'
```

**撤销监护关系**:

```bash
curl -X POST https://api.example.com/api/v1/guardians/revoke \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "usr_0987654321",
    "child_id": "chd_9876543210"
  }'
```

**查询监护关系**:

```bash
# 查询用户的所有监护儿童
curl -X GET "https://api.example.com/api/v1/guardians?user_id=usr_1234567890&active=true" \
  -H "Authorization: Bearer <token>"

# 查询儿童的所有监护人
curl -X GET "https://api.example.com/api/v1/guardians?child_id=chd_9876543210" \
  -H "Authorization: Bearer <token>"

# 查询特定监护关系
curl -X GET "https://api.example.com/api/v1/guardians?user_id=usr_1234567890&child_id=chd_9876543210" \
  -H "Authorization: Bearer <token>"
```

**响应**:

```json
{
  "total": 1,
  "items": [
    {
      "id": 12345,
      "user_id": "usr_1234567890",
      "child_id": "chd_9876543210",
      "relation": "parent",
      "since": "2024-10-29T11:00:00Z",
      "revoked_at": null
    }
  ]
}
```

---

## 🔐 认证与授权

### JWT 令牌结构

**Header**:

```json
{
  "alg": "RS256",
  "typ": "JWT",
  "kid": "2024-10-key-1"
}
```

**Payload**:

```json
{
  "sub": "usr_1234567890",
  "iat": 1698566400,
  "exp": 1698652800,
  "nbf": 1698566400,
  "jti": "jwt_abcdef123456",
  "iss": "https://api.example.com",
  "aud": ["iam-api"],
  "scope": "read write",
  "account_id": "acc_0987654321",
  "account_type": "operation"
}
```

### 使用 JWT 访问 API

```bash
curl -X GET https://api.example.com/api/v1/users/usr_1234567890 \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

## 🛡️ 安全最佳实践

### 1. 幂等性

所有 `POST` 请求都应提供幂等键：

```bash
curl -X POST https://api.example.com/api/v1/children/register \
  -H "X-Idempotency-Key: $(uuidgen)" \
  -H "Authorization: Bearer <token>" \
  ...
```

### 2. 请求追踪

建议所有请求携带追踪 ID：

```bash
curl -X GET https://api.example.com/api/v1/users/usr_123 \
  -H "X-Request-Id: $(uuidgen)" \
  -H "Authorization: Bearer <token>"
```

### 3. HTTPS 强制

- 生产环境必须使用 HTTPS
- 本地开发可使用 HTTP (<http://localhost:8080>)

### 4. 令牌存储

- **不要**将令牌存储在 localStorage（易受 XSS 攻击）
- **推荐**使用 HttpOnly Cookie 或内存存储
- **移动端**使用 Keychain (iOS) 或 Keystore (Android)

### 5. 令牌刷新策略

```javascript
// 推荐：在 access_token 过期前 5 分钟刷新
const shouldRefresh = (expiresIn) => expiresIn < 300; // 300秒 = 5分钟

if (shouldRefresh(tokenExpiresIn)) {
  const newToken = await refreshAccessToken(refreshToken);
}
```

---

## 📊 HTTP 状态码

| 状态码 | 说明 | 示例场景 |
|--------|------|----------|
| **200** | 成功 | GET/PATCH 成功 |
| **201** | 创建成功 | POST 创建资源成功 |
| **204** | 无内容 | DELETE 成功 |
| **400** | 请求参数错误 | 缺少必填字段、格式错误 |
| **401** | 未认证 | 缺少令牌、令牌无效 |
| **403** | 无权限 | 令牌有效但无操作权限 |
| **404** | 资源不存在 | 用户/儿童不存在 |
| **409** | 冲突 | 用户名重复、幂等键冲突 |
| **422** | 业务规则错误 | 儿童年龄不符合规则 |
| **429** | 请求过多 | 触发限流 |
| **500** | 服务器错误 | 内部错误 |
| **503** | 服务不可用 | 维护中或过载 |

---

## 🧪 测试示例

### Postman Collection

导入 Postman Collection:

```bash
# 下载 Collection
curl -o iam-api.postman_collection.json \
  https://api.example.com/docs/postman/collection.json

# 导入环境变量
curl -o iam-api.postman_environment.json \
  https://api.example.com/docs/postman/environment.json
```

### cURL 测试脚本

```bash
#!/bin/bash
# 完整流程测试脚本

# 1. 登录
TOKEN=$(curl -s -X POST https://api.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"account_type":"operation","username":"admin","password":"admin123"}' \
  | jq -r '.access_token')

echo "Token: $TOKEN"

# 2. 创建用户
USER_ID=$(curl -s -X POST https://api.example.com/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"nickname":"测试用户"}' \
  | jq -r '.id')

echo "User ID: $USER_ID"

# 3. 注册儿童
CHILD_RESPONSE=$(curl -s -X POST https://api.example.com/api/v1/children/register \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: $(uuidgen)" \
  -d '{
    "legal_name": "测试儿童",
    "gender": 1,
    "dob": "2020-01-01",
    "id_type": "id_card",
    "id_card": "110101202001010012",
    "relation": "parent"
  }')

echo "Child Response: $CHILD_RESPONSE"

# 4. 查询我的孩子
curl -s -X GET https://api.example.com/api/v1/me/children \
  -H "Authorization: Bearer $TOKEN" \
  | jq .
```

---

## 📚 相关资源

- **OpenAPI 规范文件**:
  - [authn.v1.yaml](./authn.v1.yaml) - 认证 API 完整规范
  - [identity.v1.yaml](./identity.v1.yaml) - 身份 API 完整规范

- **在线文档**:
  - [Swagger UI](https://api.example.com/swagger) - 交互式 API 文档
  - [ReDoc](https://api.example.com/redoc) - 美化版 API 文档

- **SDK 与工具**:
  - [TypeScript SDK](https://www.npmjs.com/package/@iam/api-client)
  - [Go SDK](https://github.com/FangcunMount/iam-sdk-go)
  - [Postman Collection](https://api.example.com/docs/postman/collection.json)

---

## 📞 技术支持

- **API 问题**: [GitHub Issues](https://github.com/FangcunMount/iam-contracts/issues)
- **功能请求**: [Feature Request](https://github.com/FangcunMount/iam-contracts/issues/new?template=feature_request.md)
- **安全问题**: <security@example.com>
