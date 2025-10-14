# 认证模块快速参考

## 🚀 快速开始

### 1. 模块初始化

```go
// 创建并初始化认证模块
authModule := assembler.NewAuthModule()
err := authModule.Initialize(db, redisClient)
```

### 2. 路由注册

```go
// 注册认证路由
restful.Provide(restful.Dependencies{
    AuthHandler:    authModule.AuthHandler,
    AccountHandler: authModule.AccountHandler,
})
restful.Register(ginEngine)
```

---

## 📍 API 端点

### 认证相关

| 端点 | 方法 | 说明 |
|-----|------|-----|
| `/api/v1/auth/login` | POST | 统一登录（basic/wx:minip） |
| `/api/v1/auth/token` | POST | 刷新令牌 |
| `/api/v1/auth/logout` | POST | 登出 |
| `/api/v1/auth/verify` | POST | 验证令牌 |
| `/.well-known/jwks.json` | GET | 公钥集 |

---

## 💡 使用示例

### 密码登录

**请求:**

```json
POST /api/v1/auth/login
{
  "method": "basic",
  "credentials": {
    "username": "admin",
    "password": "password123"
  }
}
```

**响应:**

```json
{
  "accessToken": "eyJhbG...",
  "tokenType": "Bearer",
  "expiresIn": 900,
  "refreshToken": "550e8400-..."
}
```

### 微信登录

**请求:**

```json
POST /api/v1/auth/login
{
  "method": "wx:minip",
  "credentials": {
    "appId": "wx1234567890",
    "jsCode": "021xYz0w..."
  }
}
```

### 刷新令牌

**请求:**

```json
POST /api/v1/auth/token
{
  "refreshToken": "550e8400-..."
}
```

### 验证令牌

**请求:**

```http
POST /api/v1/auth/verify
Authorization: Bearer eyJhbG...
```

### 登出

**请求:**

```http
POST /api/v1/auth/logout
Authorization: Bearer eyJhbG...

{
  "refreshToken": "550e8400-..."
}
```

---

## 🔧 配置清单

### JWT 配置（TODO）

- `jwt.secret_key`: JWT 签名密钥
- `jwt.issuer`: 颁发者标识
- `jwt.access_ttl`: 访问令牌有效期（默认 15分钟）
- `jwt.refresh_ttl`: 刷新令牌有效期（默认 7天）

### 微信配置（TODO）

- `wechat.apps`: 微信应用列表（appId + appSecret）

---

## 📦 核心组件

### 依赖注入流程

```text
基础设施层组件 → 领域层组件 → 应用层组件 → 接口层组件
```

### 认证流程

```text
用户提交登录
  → AuthHandler.Login()
  → LoginService.LoginWithPassword/WeChat()
  → AuthenticationService.Authenticate()
  → BasicAuthenticator/WeChatAuthenticator
  → TokenService.IssueToken()
  → 返回 TokenPair
```

---

## ✅ 已实现功能

- ✅ 用户名密码认证（Bcrypt）
- ✅ 微信小程序认证
- ✅ JWT 访问令牌（15分钟）
- ✅ UUID 刷新令牌（7天，Redis）
- ✅ 令牌刷新（旋转策略）
- ✅ 令牌撤销（黑名单）
- ✅ 令牌验证
- ✅ 账户状态检查
- ✅ JWKS 端点

---

## 📝 TODO 列表

### 必须完成

- [ ] 从配置加载 JWT 密钥
- [ ] 从配置加载微信应用配置
- [ ] 编写集成测试
- [ ] 完善日志记录
- [ ] 错误消息国际化

### 可选增强

- [ ] 实现"撤销所有令牌"
- [ ] 多因素认证（TOTP）
- [ ] OAuth2 标准流程
- [ ] 审计日志
- [ ] 速率限制
- [ ] 设备管理

---

## 🐛 故障排查

### 编译错误

```bash
# 检查所有层是否正确编译
go build ./internal/apiserver/modules/authn/...
```

### 依赖检查

```bash
# 确保导入了正确的包
go mod tidy
```

### 运行时问题

- 检查 DB 连接是否正常
- 检查 Redis 连接是否正常
- 查看日志获取详细错误信息

---

**文档版本**: v1.0.0  
**最后更新**: 2025年10月14日
