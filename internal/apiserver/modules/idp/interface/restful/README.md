# Interface 层（RESTful API）

## 概述

Interface 层是 IDP 模块的 HTTP 适配器层，提供 RESTful API 接口供外部客户端（如前端应用、移动端等）调用。本层遵循**六边形架构**（Hexagonal Architecture）的原则，作为驱动适配器（Driving Adapter）与外部世界交互。

## 架构原则

### 1. **依赖方向**

```
外部客户端 → Interface 层 → Application 层 → Domain 层
```

- ✅ **Interface 层依赖 Application 层的服务接口**
- ❌ **Interface 层不能直接依赖 Domain 层或 Infrastructure 层**
- ✅ **使用 Request/Response DTOs 进行数据传输**

### 2. **职责划分**

| 层次 | 职责 | 示例 |
|------|------|------|
| **Interface** | HTTP 协议适配、参数验证、响应格式化 | Gin Handler、路由注册 |
| **Application** | 业务流程编排、事务管理、DTO 转换 | 应用服务 |
| **Domain** | 业务规则、领域逻辑 | 领域服务、聚合根 |
| **Infrastructure** | 技术实现、外部系统集成 | MySQL、Redis、微信 API |

### 3. **RESTful 设计原则**

- **资源导向**：URL 表示资源，HTTP 方法表示操作
- **统一接口**：使用标准 HTTP 状态码
- **无状态**：每个请求包含完整的信息
- **可缓存性**：支持 HTTP 缓存机制

## 目录结构

```
interface/restful/
├── router.go              # 路由注册（主入口）
├── handler/               # HTTP 处理器
│   ├── base.go           # 基础处理器（统一响应和绑定）
│   ├── wechatapp.go      # 微信应用管理处理器
│   └── wechatauth.go     # 微信认证处理器
├── request/              # 请求 DTOs
│   └── request.go        # 请求结构定义
└── response/             # 响应 DTOs
    └── response.go       # 响应结构定义
```

## API 端点总览

### 基础路径

所有 IDP 模块的 API 都在 `/api/v1/idp` 路径下。

### 微信应用管理（WechatApp）

| HTTP 方法 | 路径 | 说明 | 认证 |
|----------|------|------|------|
| `POST` | `/wechat-apps` | 创建微信应用 | 需要 |
| `GET` | `/wechat-apps/:app_id` | 查询微信应用 | 需要 |
| `GET` | `/wechat-apps/:app_id/access-token` | 获取访问令牌 | 需要 |
| `POST` | `/wechat-apps/rotate-auth-secret` | 轮换认证密钥 | 需要 |
| `POST` | `/wechat-apps/rotate-msg-secret` | 轮换消息密钥 | 需要 |
| `POST` | `/wechat-apps/refresh-access-token` | 刷新访问令牌 | 需要 |

### 微信认证（WechatAuth）

| HTTP 方法 | 路径 | 说明 | 认证 |
|----------|------|------|------|
| `POST` | `/wechat/login` | 微信小程序登录 | 不需要 |
| `POST` | `/wechat/decrypt-phone` | 解密用户手机号 | 需要 |

### 健康检查

| HTTP 方法 | 路径 | 说明 | 认证 |
|----------|------|------|------|
| `GET` | `/health` | 模块健康检查 | 不需要 |

## API 详细说明

### 1. 创建微信应用

**请求：**

```http
POST /api/v1/idp/wechat-apps
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
  "app_id": "wx1234567890abcdef",
  "name": "我的小程序",
  "type": "MiniProgram",
  "app_secret": "1a2b3c4d5e6f7g8h9i0j"
}
```

**响应：**

```json
{
  "id": "app-uuid-123",
  "app_id": "wx1234567890abcdef",
  "name": "我的小程序",
  "type": "MiniProgram",
  "status": "Active"
}
```

**状态码：**
- `201 Created` - 创建成功
- `400 Bad Request` - 请求参数错误
- `500 Internal Server Error` - 服务器内部错误

---

### 2. 查询微信应用

**请求：**

```http
GET /api/v1/idp/wechat-apps/wx1234567890abcdef
Authorization: Bearer <jwt_token>
```

**响应：**

```json
{
  "id": "app-uuid-123",
  "app_id": "wx1234567890abcdef",
  "name": "我的小程序",
  "type": "MiniProgram",
  "status": "Active"
}
```

**状态码：**
- `200 OK` - 查询成功
- `404 Not Found` - 应用不存在
- `500 Internal Server Error` - 服务器内部错误

---

### 3. 获取访问令牌

**请求：**

```http
GET /api/v1/idp/wechat-apps/wx1234567890abcdef/access-token
Authorization: Bearer <jwt_token>
```

**响应：**

```json
{
  "access_token": "68_A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6",
  "expires_in": 7200
}
```

**状态码：**
- `200 OK` - 获取成功
- `404 Not Found` - 应用不存在
- `500 Internal Server Error` - 服务器内部错误

---

### 4. 微信小程序登录

**请求：**

```http
POST /api/v1/idp/wechat/login
Content-Type: application/json

{
  "app_id": "wx1234567890abcdef",
  "js_code": "071xYZ123456"
}
```

**响应：**

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

**状态码：**
- `200 OK` - 登录成功
- `400 Bad Request` - 请求参数错误
- `401 Unauthorized` - 登录失败（code 无效或已过期）
- `500 Internal Server Error` - 服务器内部错误

---

### 5. 解密用户手机号

**请求：**

```http
POST /api/v1/idp/wechat/decrypt-phone
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
  "app_id": "wx1234567890abcdef",
  "open_id": "oABC123456XYZ",
  "encrypted_data": "encrypted_phone_data_base64",
  "iv": "iv_base64"
}
```

**响应：**

```json
{
  "phone": "13800138000"
}
```

**状态码：**
- `200 OK` - 解密成功
- `400 Bad Request` - 请求参数错误
- `401 Unauthorized` - 解密失败（session_key 无效）
- `500 Internal Server Error` - 服务器内部错误

---

### 6. 轮换认证密钥

**请求：**

```http
POST /api/v1/idp/wechat-apps/rotate-auth-secret
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
  "app_id": "wx1234567890abcdef",
  "new_secret": "new_app_secret_1234567890"
}
```

**响应：**

```json
{
  "success": true,
  "message": "Auth secret rotated successfully"
}
```

**状态码：**
- `200 OK` - 轮换成功
- `400 Bad Request` - 请求参数错误
- `404 Not Found` - 应用不存在
- `500 Internal Server Error` - 服务器内部错误

---

### 7. 刷新访问令牌

**请求：**

```http
POST /api/v1/idp/wechat-apps/refresh-access-token
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
  "app_id": "wx1234567890abcdef"
}
```

**响应：**

```json
{
  "access_token": "68_NEW_TOKEN_ABC123",
  "expires_in": 7200
}
```

**状态码：**
- `200 OK` - 刷新成功
- `400 Bad Request` - 请求参数错误
- `404 Not Found` - 应用不存在
- `500 Internal Server Error` - 服务器内部错误

## 错误响应格式

所有错误响应都遵循统一格式：

```json
{
  "code": 400001,
  "message": "参数验证失败: app_id 不能为空"
}
```

**常见错误码：**

| 错误码 | 说明 |
|--------|------|
| `100101` | 参数绑定失败 |
| `100201` | 认证失败 |
| `100301` | 授权失败 |
| `100401` | 资源不存在 |
| `100501` | 服务器内部错误 |

## Handler 实现

### 1. BaseHandler

提供统一的响应和参数绑定能力：

```go
type BaseHandler struct{}

// Success 写出成功响应
func (h *BaseHandler) Success(c *gin.Context, data interface{})

// Error 写出错误响应
func (h *BaseHandler) Error(c *gin.Context, err error)

// BindJSON 绑定 JSON 请求体
func (h *BaseHandler) BindJSON(c *gin.Context, obj interface{}) error

// BindURI 绑定 URI 参数
func (h *BaseHandler) BindURI(c *gin.Context, obj interface{}) error
```

### 2. WechatAppHandler

微信应用管理处理器：

```go
type WechatAppHandler struct {
    *BaseHandler
    appService        wechatapp.WechatAppApplicationService
    credentialService wechatapp.WechatAppCredentialApplicationService
    tokenService      wechatapp.WechatAppTokenApplicationService
}
```

**提供的方法：**
- `CreateWechatApp()` - 创建微信应用
- `GetWechatApp()` - 查询微信应用
- `GetAccessToken()` - 获取访问令牌
- `RefreshAccessToken()` - 刷新访问令牌
- `RotateAuthSecret()` - 轮换认证密钥
- `RotateMsgSecret()` - 轮换消息密钥

### 3. WechatAuthHandler

微信认证处理器：

```go
type WechatAuthHandler struct {
    *BaseHandler
    authService wechatsession.WechatAuthApplicationService
}
```

**提供的方法：**
- `LoginWithCode()` - 微信登录
- `DecryptUserPhone()` - 解密手机号

## 路由注册

### 注册流程

```go
package restful

// Dependencies IDP 模块的依赖
type Dependencies struct {
    WechatAppHandler  *handler.WechatAppHandler
    WechatAuthHandler *handler.WechatAuthHandler
}

// Provide 存储依赖供 Register 使用
func Provide(d Dependencies) {
    deps = d
}

// Register 注册 IDP 模块的所有路由
func Register(engine *gin.Engine) {
    // 注册路由...
}
```

### 集成到主路由

在 `internal/apiserver/routers.go` 中注册：

```go
import (
    idphttp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/interface/restful"
)

func (r *Router) RegisterRoutes(engine *gin.Engine) {
    // ... 其他模块注册 ...
    
    // 注册 IDP 模块路由
    idphttp.Register(engine)
}
```

## 请求/响应 DTOs

### Request DTOs

定义在 `request/request.go`：

- `CreateWechatAppRequest` - 创建微信应用
- `GetWechatAppRequest` - 查询微信应用
- `LoginWithCodeRequest` - 微信登录
- `DecryptPhoneRequest` - 解密手机号
- `RotateAuthSecretRequest` - 轮换认证密钥
- `RefreshAccessTokenRequest` - 刷新访问令牌

**特点：**
- 使用 `binding` tag 进行参数验证
- 支持 JSON、URI、Query 参数绑定

### Response DTOs

定义在 `response/response.go`：

- `WechatAppResponse` - 微信应用响应
- `LoginResponse` - 登录响应
- `AccessTokenResponse` - 访问令牌响应
- `DecryptPhoneResponse` - 解密手机号响应
- `ErrorResponse` - 错误响应

**特点：**
- 使用 `json` tag 定义 JSON 字段名
- 支持 `omitempty` 处理可选字段

## 使用示例

### 客户端调用示例（JavaScript）

```javascript
// 微信小程序登录
async function wechatLogin() {
  // 1. 获取微信登录码
  const { code } = await wx.login();
  
  // 2. 调用后端登录接口
  const response = await fetch('/api/v1/idp/wechat/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      app_id: 'wx1234567890abcdef',
      js_code: code
    })
  });
  
  const result = await response.json();
  
  // 3. 保存 session_key 和 open_id
  wx.setStorageSync('session_key', result.session_key);
  wx.setStorageSync('open_id', result.open_id);
  
  return result;
}

// 解密手机号
async function decryptPhone(encryptedData, iv) {
  const appId = 'wx1234567890abcdef';
  const openId = wx.getStorageSync('open_id');
  
  const response = await fetch('/api/v1/idp/wechat/decrypt-phone', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${getJwtToken()}`
    },
    body: JSON.stringify({
      app_id: appId,
      open_id: openId,
      encrypted_data: encryptedData,
      iv: iv
    })
  });
  
  const result = await response.json();
  return result.phone;
}
```

### 服务端调用示例（Go）

```go
// 管理员创建微信应用
func createWechatApp(client *http.Client, token string) error {
    req := &request.CreateWechatAppRequest{
        AppID:     "wx1234567890abcdef",
        Name:      "我的小程序",
        Type:      "MiniProgram",
        AppSecret: "1a2b3c4d5e6f7g8h9i0j",
    }
    
    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequest("POST", 
        "http://api.example.com/api/v1/idp/wechat-apps", 
        bytes.NewReader(body))
    
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+token)
    
    resp, err := client.Do(httpReq)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusCreated {
        return fmt.Errorf("failed to create app: %d", resp.StatusCode)
    }
    
    return nil
}
```

## 测试建议

### 单元测试

对每个 Handler 编写单元测试：

```go
func TestWechatAppHandler_CreateWechatApp(t *testing.T) {
    // 准备 mock 应用服务
    mockAppService := &mockWechatAppApplicationService{}
    handler := NewWechatAppHandler(mockAppService, nil, nil)
    
    // 准备测试请求
    req := &request.CreateWechatAppRequest{
        AppID: "wx123",
        Name:  "Test App",
        Type:  "MiniProgram",
    }
    
    // 执行测试
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    // ... 设置请求体 ...
    
    handler.CreateWechatApp(c)
    
    // 验证响应
    assert.Equal(t, http.StatusCreated, w.Code)
}
```

### 集成测试

在真实环境中测试 API 端点：

```go
func TestWechatLoginIntegration(t *testing.T) {
    // 启动测试服务器
    router := setupRouter()
    ts := httptest.NewServer(router)
    defer ts.Close()
    
    // 调用 API
    resp, err := http.Post(
        ts.URL+"/api/v1/idp/wechat/login",
        "application/json",
        strings.NewReader(`{"app_id":"wx123","js_code":"test"}`),
    )
    
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

## 最佳实践

### 1. 参数验证

使用 `binding` tag 进行声明式验证：

```go
type CreateWechatAppRequest struct {
    AppID string `json:"app_id" binding:"required,min=18,max=32"`
    Name  string `json:"name" binding:"required,max=100"`
}
```

### 2. 错误处理

统一使用 `BaseHandler.Error()` 处理错误：

```go
func (h *WechatAppHandler) CreateWechatApp(c *gin.Context) {
    // ...
    result, err := h.appService.CreateApp(ctx, dto)
    if err != nil {
        h.Error(c, err)  // 统一错误处理
        return
    }
    // ...
}
```

### 3. 上下文传递

使用 Gin Context 传递请求上下文：

```go
result, err := h.authService.LoginWithCode(
    c.Request.Context(),  // 传递 HTTP 上下文
    dto,
)
```

### 4. 响应格式化

使用 DTO 进行响应格式化：

```go
resp := &response.WechatAppResponse{
    ID:     result.ID,
    AppID:  result.AppID,
    Name:   result.Name,
    Type:   string(result.Type),
    Status: string(result.Status),
}
h.Success(c, resp)
```

## 安全考虑

### 1. 认证授权

大部分接口需要 JWT 认证：

```go
// 在 router.go 中使用中间件
wechatApps.Use(authMiddleware.AuthRequired())
```

### 2. 参数验证

使用 Gin 的 binding 验证：

```go
binding:"required,min=1,max=100"
```

### 3. 限流

建议使用限流中间件：

```go
wechatAuth.Use(ratelimit.Middleware(100, time.Minute))
```

### 4. HTTPS

生产环境必须使用 HTTPS。

## 性能优化

### 1. 缓存

对访问令牌等频繁访问的数据使用缓存：

```go
token, err := h.tokenService.GetAccessToken(ctx, appID)  // 自动使用缓存
```

### 2. 连接池

使用 HTTP/2 和连接池优化性能。

### 3. 异步处理

对非关键路径使用异步处理：

```go
go h.logAccessToken(appID, token)  // 异步记录日志
```

## 监控和日志

### 1. 访问日志

使用 Gin 的日志中间件：

```go
engine.Use(gin.Logger())
```

### 2. 错误日志

记录错误详情：

```go
log.Errorw("failed to create wechat app", 
    "app_id", req.AppID, 
    "error", err)
```

### 3. 性能指标

记录 API 响应时间等指标。

## 相关文档

- [IDP 模块总览](../../README.md)
- [应用层文档](../../application/README.md)
- [领域层文档](../../domain/README.md)
- [基础设施层文档](../../infra/README.md)

## 贡献指南

### 添加新的 API 端点

1. 在 `request/request.go` 定义请求 DTO
2. 在 `response/response.go` 定义响应 DTO
3. 在对应的 Handler 中实现处理方法
4. 在 `router.go` 中注册路由
5. 编写测试用例

### 代码规范

- Handler 方法使用大写开头（可导出）
- 使用 `BaseHandler` 提供的统一方法
- 错误处理使用统一格式
- 添加 Swagger 注释

---

**最后更新：** 2025-10-29  
**状态：** ✅ Interface 层实现完成，所有文件编译通过  
**架构合规性：** ✅ 符合六边形架构原则
