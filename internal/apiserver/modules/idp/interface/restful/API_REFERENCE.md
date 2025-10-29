# IDP 模块 API 参考

## 概述

本文档提供 IDP（Identity Provider）模块的完整 API 参考，包括所有端点、请求格式、响应格式和示例。

## 基础信息

- **基础路径：** `/api/v1/idp`
- **认证方式：** JWT Bearer Token（部分端点需要）
- **内容类型：** `application/json`
- **字符编码：** `UTF-8`

## API 端点列表

### 微信应用管理

| 端点 | 方法 | 说明 | 认证 |
|------|------|------|------|
| `/wechat-apps` | POST | 创建微信应用 | ✓ |
| `/wechat-apps/:app_id` | GET | 查询微信应用 | ✓ |
| `/wechat-apps/:app_id/access-token` | GET | 获取访问令牌 | ✓ |
| `/wechat-apps/rotate-auth-secret` | POST | 轮换认证密钥 | ✓ |
| `/wechat-apps/rotate-msg-secret` | POST | 轮换消息密钥 | ✓ |
| `/wechat-apps/refresh-access-token` | POST | 刷新访问令牌 | ✓ |

### 微信认证

| 端点 | 方法 | 说明 | 认证 |
|------|------|------|------|
| `/wechat/login` | POST | 微信小程序登录 | ✗ |
| `/wechat/decrypt-phone` | POST | 解密用户手机号 | ✓ |

### 系统

| 端点 | 方法 | 说明 | 认证 |
|------|------|------|------|
| `/health` | GET | 健康检查 | ✗ |

---

## 详细 API 说明

### 1. 创建微信应用

创建一个新的微信应用配置。

#### 请求

```
POST /api/v1/idp/wechat-apps
```

**Headers:**
```
Content-Type: application/json
Authorization: Bearer <jwt_token>
```

**Body:**
```json
{
  "app_id": "wx1234567890abcdef",
  "name": "我的小程序",
  "type": "MiniProgram",
  "app_secret": "1a2b3c4d5e6f7g8h9i0j"
}
```

**参数说明：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `app_id` | string | ✓ | 微信应用 ID（18-32 字符） |
| `name` | string | ✓ | 应用名称（最多 100 字符） |
| `type` | string | ✓ | 应用类型：`MiniProgram` 或 `OfficialAccount` |
| `app_secret` | string | ✗ | AppSecret（可选，创建时设置） |

#### 响应

**成功响应（201 Created）：**
```json
{
  "id": "app-uuid-123",
  "app_id": "wx1234567890abcdef",
  "name": "我的小程序",
  "type": "MiniProgram",
  "status": "Active"
}
```

**响应字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 内部应用 ID（UUID） |
| `app_id` | string | 微信应用 ID |
| `name` | string | 应用名称 |
| `type` | string | 应用类型 |
| `status` | string | 应用状态：`Active` 或 `Inactive` |

**错误响应：**

| 状态码 | 说明 | 示例 |
|--------|------|------|
| 400 | 参数错误 | `{"code": 100101, "message": "app_id 不能为空"}` |
| 401 | 未认证 | `{"code": 100201, "message": "未提供认证令牌"}` |
| 409 | 应用已存在 | `{"code": 100901, "message": "应用已存在"}` |
| 500 | 服务器错误 | `{"code": 100501, "message": "内部服务器错误"}` |

#### 示例

**cURL:**
```bash
curl -X POST https://api.example.com/api/v1/idp/wechat-apps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d '{
    "app_id": "wx1234567890abcdef",
    "name": "我的小程序",
    "type": "MiniProgram",
    "app_secret": "1a2b3c4d5e6f7g8h9i0j"
  }'
```

**JavaScript:**
```javascript
const response = await fetch('/api/v1/idp/wechat-apps', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  },
  body: JSON.stringify({
    app_id: 'wx1234567890abcdef',
    name: '我的小程序',
    type: 'MiniProgram',
    app_secret: '1a2b3c4d5e6f7g8h9i0j'
  })
});
const data = await response.json();
```

---

### 2. 查询微信应用

根据微信应用 ID 查询应用配置信息。

#### 请求

```
GET /api/v1/idp/wechat-apps/:app_id
```

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**URL 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `app_id` | string | ✓ | 微信应用 ID |

#### 响应

**成功响应（200 OK）：**
```json
{
  "id": "app-uuid-123",
  "app_id": "wx1234567890abcdef",
  "name": "我的小程序",
  "type": "MiniProgram",
  "status": "Active"
}
```

**错误响应：**

| 状态码 | 说明 |
|--------|------|
| 404 | 应用不存在 |
| 401 | 未认证 |
| 500 | 服务器错误 |

#### 示例

**cURL:**
```bash
curl -X GET https://api.example.com/api/v1/idp/wechat-apps/wx1234567890abcdef \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

### 3. 获取访问令牌

获取微信应用的 access_token，支持自动缓存和刷新。

#### 请求

```
GET /api/v1/idp/wechat-apps/:app_id/access-token
```

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**URL 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `app_id` | string | ✓ | 微信应用 ID |

#### 响应

**成功响应（200 OK）：**
```json
{
  "access_token": "68_A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6",
  "expires_in": 7200
}
```

**响应字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `access_token` | string | 微信访问令牌 |
| `expires_in` | int | 过期时间（秒），默认 7200 |

**说明：**
- 访问令牌会被缓存，重复调用会返回缓存的令牌
- 令牌即将过期时会自动刷新
- 支持分布式锁，防止并发刷新

#### 示例

**cURL:**
```bash
curl -X GET https://api.example.com/api/v1/idp/wechat-apps/wx1234567890abcdef/access-token \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

### 4. 刷新访问令牌

强制刷新微信应用的 access_token。

#### 请求

```
POST /api/v1/idp/wechat-apps/refresh-access-token
```

**Headers:**
```
Content-Type: application/json
Authorization: Bearer <jwt_token>
```

**Body:**
```json
{
  "app_id": "wx1234567890abcdef"
}
```

#### 响应

**成功响应（200 OK）：**
```json
{
  "access_token": "68_NEW_TOKEN_ABC123",
  "expires_in": 7200
}
```

**说明：**
- 会强制从微信服务器获取新的令牌
- 旧令牌会被立即失效
- 新令牌会被缓存

#### 示例

**cURL:**
```bash
curl -X POST https://api.example.com/api/v1/idp/wechat-apps/refresh-access-token \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d '{"app_id": "wx1234567890abcdef"}'
```

---

### 5. 轮换认证密钥

轮换微信应用的 AppSecret。

#### 请求

```
POST /api/v1/idp/wechat-apps/rotate-auth-secret
```

**Headers:**
```
Content-Type: application/json
Authorization: Bearer <jwt_token>
```

**Body:**
```json
{
  "app_id": "wx1234567890abcdef",
  "new_secret": "new_app_secret_1234567890"
}
```

#### 响应

**成功响应（200 OK）：**
```json
{
  "success": true,
  "message": "Auth secret rotated successfully"
}
```

**说明：**
- 新密钥会被加密存储
- 访问令牌缓存会被清除
- 建议定期轮换密钥以提高安全性

---

### 6. 轮换消息密钥

轮换微信应用的消息加解密密钥。

#### 请求

```
POST /api/v1/idp/wechat-apps/rotate-msg-secret
```

**Headers:**
```
Content-Type: application/json
Authorization: Bearer <jwt_token>
```

**Body:**
```json
{
  "app_id": "wx1234567890abcdef",
  "callback_token": "callback_token_123",
  "encoding_aes_key": "encoding_aes_key_456"
}
```

#### 响应

**成功响应（200 OK）：**
```json
{
  "success": true,
  "message": "Message secret rotated successfully"
}
```

---

### 7. 微信小程序登录

使用微信小程序登录码进行用户认证。

#### 请求

```
POST /api/v1/idp/wechat/login
```

**Headers:**
```
Content-Type: application/json
```

**Body:**
```json
{
  "app_id": "wx1234567890abcdef",
  "js_code": "071xYZ123456"
}
```

**参数说明：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `app_id` | string | ✓ | 微信应用 ID |
| `js_code` | string | ✓ | 微信登录码（通过 wx.login() 获取） |

#### 响应

**成功响应（200 OK）：**
```json
{
  "provider": "wechat_miniprogram",
  "app_id": "wx1234567890abcdef",
  "open_id": "oABC123456XYZ",
  "union_id": "uDEF789012UVW",
  "display_name": "微信用户",
  "avatar_url": "https://wx.qlogo.cn/...",
  "phone": null,
  "email": null,
  "expires_in": 7200,
  "session_key": "encrypted_session_key_base64",
  "version": 1
}
```

**响应字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `provider` | string | 身份提供商（wechat_miniprogram） |
| `app_id` | string | 微信应用 ID |
| `open_id` | string | 用户 OpenID |
| `union_id` | string | 用户 UnionID（可选） |
| `display_name` | string | 显示名称（可选） |
| `avatar_url` | string | 头像 URL（可选） |
| `phone` | string | 手机号（可选） |
| `email` | string | 邮箱（可选） |
| `expires_in` | int | 会话过期时间（秒） |
| `session_key` | string | Session Key（加密后） |
| `version` | int | 会话版本 |

**错误响应：**

| 状态码 | 说明 |
|--------|------|
| 400 | 参数错误 |
| 401 | 登录失败（code 无效或已过期） |
| 500 | 服务器错误 |

#### 小程序调用示例

```javascript
// 1. 获取登录码
const { code } = await wx.login();

// 2. 调用后端登录接口
const response = await wx.request({
  url: 'https://api.example.com/api/v1/idp/wechat/login',
  method: 'POST',
  data: {
    app_id: 'wx1234567890abcdef',
    js_code: code
  }
});

// 3. 保存会话信息
wx.setStorageSync('session_key', response.data.session_key);
wx.setStorageSync('open_id', response.data.open_id);
```

---

### 8. 解密用户手机号

使用 session_key 解密微信小程序获取的加密手机号。

#### 请求

```
POST /api/v1/idp/wechat/decrypt-phone
```

**Headers:**
```
Content-Type: application/json
Authorization: Bearer <jwt_token>
```

**Body:**
```json
{
  "app_id": "wx1234567890abcdef",
  "open_id": "oABC123456XYZ",
  "encrypted_data": "encrypted_phone_data_base64",
  "iv": "iv_base64"
}
```

**参数说明：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `app_id` | string | ✓ | 微信应用 ID |
| `open_id` | string | ✓ | 用户 OpenID |
| `encrypted_data` | string | ✓ | 加密数据（Base64） |
| `iv` | string | ✓ | 加密算法的初始向量（Base64） |

#### 响应

**成功响应（200 OK）：**
```json
{
  "phone": "13800138000"
}
```

**错误响应：**

| 状态码 | 说明 |
|--------|------|
| 400 | 参数错误 |
| 401 | 解密失败（session_key 无效） |
| 500 | 服务器错误 |

#### 小程序调用示例

```javascript
// 1. 获取手机号按钮授权
<button open-type="getPhoneNumber" bindgetphonenumber="getPhoneNumber">
  获取手机号
</button>

// 2. 处理授权事件
async function getPhoneNumber(e) {
  const { encryptedData, iv } = e.detail;
  const openId = wx.getStorageSync('open_id');
  
  const response = await wx.request({
    url: 'https://api.example.com/api/v1/idp/wechat/decrypt-phone',
    method: 'POST',
    header: {
      'Authorization': `Bearer ${getJwtToken()}`
    },
    data: {
      app_id: 'wx1234567890abcdef',
      open_id: openId,
      encrypted_data: encryptedData,
      iv: iv
    }
  });
  
  console.log('手机号:', response.data.phone);
}
```

---

### 9. 健康检查

检查 IDP 模块的运行状态。

#### 请求

```
GET /api/v1/idp/health
```

#### 响应

**成功响应（200 OK）：**
```json
{
  "status": "ok",
  "module": "idp"
}
```

---

## 错误码说明

### 通用错误码

| 错误码 | 说明 |
|--------|------|
| 100101 | 参数绑定失败 |
| 100201 | 认证失败 |
| 100301 | 授权失败 |
| 100401 | 资源不存在 |
| 100501 | 服务器内部错误 |

### IDP 特定错误码

| 错误码 | 说明 |
|--------|------|
| 200101 | 微信应用不存在 |
| 200102 | 微信应用已存在 |
| 200201 | 微信 Code 无效或已过期 |
| 200202 | Session Key 无效 |
| 200203 | 解密失败 |
| 200301 | 访问令牌获取失败 |

---

## 速率限制

为保护服务，所有 API 端点都有速率限制：

- **认证端点：** 每分钟 100 次请求
- **管理端点：** 每分钟 1000 次请求
- **令牌端点：** 每分钟 500 次请求

超出限制会返回 `429 Too Many Requests`。

---

## 最佳实践

### 1. 缓存访问令牌

客户端应该缓存访问令牌，避免频繁请求：

```javascript
let cachedToken = null;
let tokenExpiry = null;

async function getAccessToken(appId) {
  if (cachedToken && Date.now() < tokenExpiry) {
    return cachedToken;
  }
  
  const response = await fetch(`/api/v1/idp/wechat-apps/${appId}/access-token`);
  const data = await response.json();
  
  cachedToken = data.access_token;
  tokenExpiry = Date.now() + (data.expires_in - 300) * 1000; // 提前 5 分钟刷新
  
  return cachedToken;
}
```

### 2. 错误处理

统一处理 API 错误：

```javascript
async function callAPI(url, options) {
  try {
    const response = await fetch(url, options);
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(`API Error ${error.code}: ${error.message}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('API call failed:', error);
    throw error;
  }
}
```

### 3. 重试机制

对于临时性错误，实现重试机制：

```javascript
async function callAPIWithRetry(url, options, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await callAPI(url, options);
    } catch (error) {
      if (i === maxRetries - 1 || error.code < 500) {
        throw error;
      }
      await sleep(1000 * Math.pow(2, i)); // 指数退避
    }
  }
}
```

---

## 变更日志

### v1.0.0（2025-10-29）

- ✅ 初始版本发布
- ✅ 支持微信小程序登录
- ✅ 支持微信应用管理
- ✅ 支持访问令牌管理
- ✅ 支持手机号解密

---

## 联系我们

- **技术支持：** support@example.com
- **API 问题：** api@example.com
- **文档反馈：** docs@example.com

---

**最后更新：** 2025-10-29  
**版本：** v1.0.0
