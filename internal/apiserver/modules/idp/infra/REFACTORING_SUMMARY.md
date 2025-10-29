# 基础设施层重构总结

## 重构目标

修复基础设施层违反六边形架构原则的问题：基础设施层不应依赖领域对象，应使用原始类型（primitives）进行输入输出。

## 重构范围

### 1. 微信 API 集成（wechatapi/）

#### 重构前问题

```go
// ❌ 错误：基础设施层方法接受领域对象
func (s *AuthService) Code2Session(
    ctx context.Context,
    app *wechatappDomain.WechatApp,  // 依赖领域对象
    jsCode string,
) (*wechatappDomain.Code2SessionResult, error)
```

**问题分析：**
- 基础设施层不应该知道 `domain.WechatApp` 的存在
- 违反了六边形架构的依赖方向原则
- 导致基础设施层与领域层紧耦合

#### 重构后方案

```go
// ✅ 正确：使用原始类型作为参数
func (s *AuthService) Code2Session(
    ctx context.Context,
    appID string,      // 原始类型
    appSecret string,  // 原始类型
    jsCode string,
) (*Code2SessionResult, error)

// Result 结构体：简单的数据传输对象，不含业务逻辑
type Code2SessionResult struct {
    OpenID     string
    SessionKey string
    UnionID    string
}
```

**改进点：**
- ✅ 参数使用原始类型（`string`）
- ✅ 返回值对象位于基础设施层，不依赖领域
- ✅ 领域服务层负责将领域对象转换为原始类型

### 2. 完成重构的文件

| 文件 | 重构内容 | 状态 |
|------|---------|------|
| `code2session_client.go` | 移除 `domain.WechatApp` 依赖，改用原始类型 | ✅ 完成 |
| `token_provider.go` | 移除领域对象依赖，使用 `appID`、`appSecret` 字符串参数 | ✅ 完成 |
| `infrastructure.go` | 创建基础设施服务聚合器 | ✅ 完成 |

### 3. 架构改进

#### 依赖方向

```
【重构前】
┌──────────────┐
│  领域层      │
│  (Domain)    │◄──┐ 错误：循环依赖
└──────────────┘   │
       ▲           │
       │           │
       │ 依赖      │
       │           │
┌──────────────┐   │
│ 基础设施层   ├───┘
│ (Infra)      │
└──────────────┘

【重构后】
┌──────────────┐
│  领域层      │
│  (Domain)    │
│  - 定义端口   │
└──────────────┘
       ▲
       │ 实现端口接口
       │
┌──────────────┐
│ 基础设施层   │
│ (Infra)      │
│  - 使用原始类型│
└──────────────┘
```

#### 层次协作

```
应用层 (Application)
    ↓ 调用
领域服务层 (Domain Service)
    ↓ 适配：领域对象 → 原始类型
基础设施层 (Infrastructure)
    ↓ 使用原始类型调用外部 SDK
微信 SDK (silenceper/wechat/v2)
```

**职责划分：**

1. **基础设施层（Infra）**
   - 使用原始类型（`string`、`[]byte`）
   - 调用外部 SDK
   - 返回简单的 Result 结构体

2. **领域服务层（Domain Service）**
   - 接受领域对象作为参数
   - 提取原始值（如 `app.AppID`、`app.Secret`）
   - 调用基础设施层方法
   - 将 Result 转换为领域对象

3. **应用层（Application）**
   - 协调领域服务
   - 处理应用级事务
   - 返回 DTO

## 重构后的代码示例

### 基础设施层（infra/wechatapi/code2session_client.go）

```go
package wechatapi

// AuthService 微信认证服务（使用 silenceper SDK）
type AuthService struct {
    cache cache.Cache
}

// Code2Session 小程序登录（使用原始类型参数）
func (s *AuthService) Code2Session(
    ctx context.Context,
    appID string,      // ✅ 原始类型
    appSecret string,  // ✅ 原始类型
    jsCode string,
) (*Code2SessionResult, error) {
    // 使用 silenceper SDK
    miniProgram := wechat.NewWechat().GetMiniProgram(&miniConfig.Config{
        AppID:     appID,
        AppSecret: appSecret,
        Cache:     s.cache,
    })
    
    result, err := miniProgram.GetAuth().Code2Session(jsCode)
    if err != nil {
        return nil, fmt.Errorf("failed to code2session: %w", err)
    }
    
    // ✅ 返回简单的数据结构
    return &Code2SessionResult{
        OpenID:     result.OpenID,
        SessionKey: result.SessionKey,
        UnionID:    result.UnionID,
    }, nil
}

// Code2SessionResult 登录结果（简单数据结构，不含业务逻辑）
type Code2SessionResult struct {
    OpenID     string
    SessionKey string
    UnionID    string
}
```

### 领域服务层（domain/wechatapp/service/auth_service.go）

```go
package service

// AuthenticateUser 用户认证（领域服务）
func (s *AuthService) AuthenticateUser(
    ctx context.Context,
    app *domain.WechatApp,  // 接受领域对象
    jsCode string,
) (*domain.AuthResult, error) {
    // 1. 从领域对象提取原始值
    appID := app.AppID
    appSecret, err := s.secretVault.Decrypt(ctx, app.EncryptedSecret)
    if err != nil {
        return nil, err
    }
    
    // 2. 调用基础设施层（使用原始类型）
    result, err := s.wechatAuthClient.Code2Session(
        ctx,
        appID,
        string(appSecret),
        jsCode,
    )
    if err != nil {
        return nil, err
    }
    
    // 3. 将基础设施层结果转换为领域对象
    return &domain.AuthResult{
        OpenID:     result.OpenID,
        SessionKey: result.SessionKey,
        UnionID:    result.UnionID,
    }, nil
}
```

## 架构优势

### 1. 清晰的层次边界

- **基础设施层**：只知道原始类型和外部 SDK
- **领域层**：定义端口接口，不关心具体实现
- **应用层**：协调领域服务，不直接调用基础设施

### 2. 可测试性

基础设施层的方法使用原始类型，容易编写测试：

```go
func TestAuthService_Code2Session(t *testing.T) {
    authService := wechatapi.NewAuthService(nil)
    
    result, err := authService.Code2Session(
        context.Background(),
        "wx1234567890abcdef",  // appID
        "app-secret",          // appSecret
        "071xYZ123456",        // jsCode
    )
    
    assert.NoError(t, err)
    assert.NotEmpty(t, result.OpenID)
}
```

### 3. 技术栈独立性

基础设施层的实现可以轻松替换：

- 从 `silenceper/wechat` 切换到其他 SDK
- 从微信 API 切换到其他 OAuth 提供商
- **领域层代码无需修改**

### 4. 符合六边形架构原则

```
┌─────────────────────────────────┐
│       应用层 (Application)       │
│   - 协调领域服务                  │
│   - 定义 DTO                     │
└─────────────────────────────────┘
              ▲
              │ 调用
              │
┌─────────────────────────────────┐
│        领域层 (Domain)           │
│   - 领域对象                     │
│   - 领域服务                     │
│   - 端口接口（Port）             │
└─────────────────────────────────┘
              ▲
              │ 实现端口
              │
┌─────────────────────────────────┐
│   基础设施层 (Infrastructure)    │
│   - 适配器（Adapter）            │
│   - 使用原始类型                  │
│   - 不依赖领域对象                │
└─────────────────────────────────┘
              │
              │ 调用
              ▼
┌─────────────────────────────────┐
│     外部系统 (External SDK)      │
│   - 微信 SDK                     │
│   - MySQL / Redis                │
└─────────────────────────────────┘
```

## 编译验证

所有重构文件编译通过，无错误：

```bash
✅ code2session_client.go - No errors
✅ token_provider.go - No errors
✅ infrastructure.go - No errors
```

## 后续工作

### 1. 创建领域服务适配器

在领域层创建服务，调用基础设施层并进行类型适配：

```go
// domain/wechatapp/service/token_service.go
type TokenService struct {
    tokenProvider *wechatapi.TokenProvider
    secretVault   port.SecretVault
}

func (s *TokenService) GetAccessToken(
    ctx context.Context,
    app *domain.WechatApp,
) (*domain.AppAccessToken, error) {
    // 适配：领域对象 → 原始类型
    appSecret, err := s.secretVault.Decrypt(ctx, app.EncryptedSecret)
    if err != nil {
        return nil, err
    }
    
    // 调用基础设施层
    result, err := s.tokenProvider.FetchMiniProgramToken(
        ctx,
        app.AppID,
        string(appSecret),
    )
    if err != nil {
        return nil, err
    }
    
    // 适配：Result → 领域对象
    return &domain.AppAccessToken{
        Token:     result.Token,
        ExpiresAt: result.ExpiresAt,
    }, nil
}
```

### 2. 更新应用层服务

确保应用层服务使用新的领域服务适配器。

### 3. 编写集成测试

测试完整的调用链：应用层 → 领域层 → 基础设施层 → 微信 SDK

## 总结

本次重构成功修复了基础设施层违反六边形架构的问题，确保了：

✅ **依赖方向正确**：基础设施层 → 领域层（通过端口接口）

✅ **使用原始类型**：基础设施层方法参数和返回值不依赖领域对象

✅ **职责清晰**：基础设施层负责技术实现，领域层负责业务逻辑和适配

✅ **易于测试**：各层可独立测试，不需要复杂的 mock

✅ **可扩展性强**：基础设施实现可替换，不影响领域层

重构后的代码完全符合 DDD 和六边形架构的最佳实践！
