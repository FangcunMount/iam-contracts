# 认证策略完整实现总结

## 📋 目录结构

```text
authentication/
├── port/
│   └── driven.go              # Driven 端口定义（领域需求）
├── service/
│   ├── factory.go            # 策略工厂（依赖注入）
│   ├── password.go           # 密码认证策略 ✅
│   ├── phone-otp.go          # 手机验证码认证策略 ✅
│   ├── wechat-mini.go        # 微信小程序认证策略 ✅
│   └── wechat-com.go         # 企业微信认证策略 ✅
├── authenticater.go          # 认证器（待实现）
├── decision.go               # 认证判决
├── factory.go                # 策略注册器
├── input.go                  # 认证输入
├── types.go                  # 场景、AMR等类型
└── USAGE_EXAMPLE.md          # 使用示例
```

## 🎯 核心设计思想

### 1. 六边形架构（Ports & Adapters）

```text
┌─────────────────────────────────────────────────────────────┐
│                      应用层 (Application)                    │
│                    Authenticator Service                     │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────┴──────────────────────────────────────┐
│                   领域层 (Domain)                            │
│  ┌────────────────────────────────────────────────────────┐ │
│  │         AuthStrategy Interface (策略模式)              │ │
│  │  ├── PasswordAuthStrategy                              │ │
│  │  ├── PhoneOTPAuthStrategy                              │ │
│  │  ├── OAuthWechatMinipAuthStrategy                      │ │
│  │  └── OAuthWeChatComAuthStrategy                        │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │            Driven Ports (端口定义)                      │ │
│  │  • CredentialRepository     (凭据仓储)                 │ │
│  │  • AccountRepository        (账户仓储)                 │ │
│  │  • PasswordHasher          (密码哈希)                  │ │
│  │  • OTPVerifier             (OTP验证)                   │ │
│  │  • IdentityProvider        (IdP交互)                   │ │
│  └────────────────────────────────────────────────────────┘ │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────┴──────────────────────────────────────┐
│                基础设施层 (Infrastructure)                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ MySQL Adapter│  │Redis Adapter │  │ HTTP Adapter │      │
│  │ (Repository) │  │ (OTPVerifier)│  │(IdP Provider)│      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### 2. 策略模式实现

每种认证方式都是一个独立的策略：

```go
type AuthStrategy interface {
    Kind() Scenario
    Authenticate(ctx context.Context, in AuthInput) (AuthDecision, error)
}
```

**4种策略实现：**

- ✅ `PasswordAuthStrategy` - 用户名+密码
- ✅ `PhoneOTPAuthStrategy` - 手机验证码
- ✅ `OAuthWechatMinipAuthStrategy` - 微信小程序
- ✅ `OAuthWeChatComAuthStrategy` - 企业微信

### 3. Driven 端口设计（核心改进）

**改进前（❌）：**

```go
type PasswordDeps interface {
    FindAccountByUsername(...)
    FindPasswordCredential(...)
    VerifyPHC(...)  // 混合了数据查询和算法逻辑
    NeedRehash(...)
    Pepper()
}
// 每个策略一个Deps接口，造成端口碎片化
```

**改进后（✅）：**

```go
// 按职责分离，而非按认证方式分离
type CredentialRepository interface {
    FindPasswordCredential(...)
    FindPhoneOTPCredential(...)
    FindOAuthCredential(...)  // 统一OAuth凭据查询
}

type AccountRepository interface {
    FindAccountByUsername(...)
    GetAccountStatus(...)
}

type PasswordHasher interface {
    Verify(...)
    NeedRehash(...)
    Hash(...)
    Pepper()
}

type OTPVerifier interface {
    VerifyAndConsume(...)
}

type IdentityProvider interface {
    ExchangeWxMinipCode(...)
    ExchangeWecomCode(...)
}
```

## 🔧 使用示例

### 示例1：密码认证完整流程

```go
// 1. 创建策略（通过工厂注入依赖）
strategy := service.NewPasswordAuthStrategy(
    credRepo,    // MySQL实现
    accountRepo, // MySQL实现
    hasher,      // Argon2实现
)

// 2. 准备输入
input := domain.AuthInput{
    TenantID: nil,
    Username: "alice@example.com",
    Password: "SecureP@ssw0rd",
    RemoteIP: "192.168.1.100",
    UserAgent: "Mozilla/5.0...",
}

// 3. 执行认证
decision, err := strategy.Authenticate(ctx, input)
if err != nil {
    // 系统异常（数据库错误等）
    log.Error("auth error", err)
    return
}

// 4. 处理判决
if !decision.OK {
    // 业务失败
    switch decision.ErrCode {
    case domain.ErrInvalidCredential:
        // 记录失败次数，可能锁定账户
        recordFailure(decision.CredentialID)
    case domain.ErrLocked:
        return "账户已锁定"
    case domain.ErrDisabled:
        return "账户已禁用"
    }
    return
}

// 5. 认证成功
principal := decision.Principal
// principal.AccountID = 100
// principal.UserID = 200
// principal.AMR = ["pwd"]

// 6. 可选：处理密码rehash
if decision.ShouldRotate {
    updatePasswordHash(decision.CredentialID, decision.NewMaterial)
}

// 7. 签发Token
token := issueJWT(principal)
```

### 示例2：手机验证码认证

```go
strategy := service.NewPhoneOTPAuthStrategy(
    credRepo,
    accountRepo,
    otpVerifier, // Redis实现
)

input := domain.AuthInput{
    PhoneE164: "+8613800138000",
    OTP:       "123456",
    RemoteIP:  "192.168.1.100",
}

decision, err := strategy.Authenticate(ctx, input)
// 处理逻辑类似...
```

### 示例3：微信小程序认证

```go
strategy := service.NewOAuthWechatMinipAuthStrategy(
    credRepo,
    accountRepo,
    idp, // 微信HTTP客户端实现
)

input := domain.AuthInput{
    WxAppID:  "wx1234567890abcdef",
    WxJsCode: "061AbcDef2gHIjk0lmNOp3qRStU1AbcD2fGhIJ",
}

decision, err := strategy.Authenticate(ctx, input)
// decision.Principal.Claims["wx_openid"] = "oXyz..."
// decision.Principal.Claims["wx_unionid"] = "uAbc..."
```

### 示例4：企业微信认证

```go
strategy := service.NewOAuthWeChatComAuthStrategy(
    credRepo,
    accountRepo,
    idp,
)

input := domain.AuthInput{
    WecomCorpID: "ww1234567890abcdef",
    WecomCode:   "CODE123456",
    WecomState:  "STATE_XYZ",
}

decision, err := strategy.Authenticate(ctx, input)
// decision.Principal.Claims["wecom_user_id"] = "zhangsan"
```

## 🏭 工厂模式使用

```go
// 创建工厂（一次性初始化）
factory := service.NewStrategyFactory(
    credRepo,
    accountRepo,
    hasher,
    otpVerifier,
    idp,
)

// 根据场景动态创建策略
scenario := domain.AuthPassword // 从请求中解析
strategy := factory.CreateStrategy(scenario)

// 执行认证
decision, err := strategy.Authenticate(ctx, input)
```

## 📊 认证判决处理

```go
type AuthDecision struct {
    OK           bool           // 是否认证成功
    ErrCode      ErrCode        // 业务错误码（OK=false时）
    Principal    *Principal     // 认证主体（OK=true时）
    CredentialID int64          // 凭据ID（用于审计）
    
    // 可选：密码rehash
    ShouldRotate bool
    NewMaterial  []byte
    NewAlgo      *string
}
```

**错误码映射：**

```go
const (
    ErrInvalidCredential  = "invalid_credential"   // 凭据无效
    ErrOTPMissingOrExpiry = "otp_invalid_or_expired" // OTP无效
    ErrStateMismatch      = "state_mismatch"       // state不匹配
    ErrIDPExchangeFailed  = "idp_exchange_failed"  // IdP交互失败
    ErrNoBinding          = "no_binding"           // 未绑定账户
    ErrLocked             = "locked"               // 账户锁定
    ErrDisabled           = "disabled"             // 账户禁用
)
```

## 🧪 测试示例

```go
func TestPasswordAuthStrategy(t *testing.T) {
    // Mock所有依赖
    credRepo := &MockCredentialRepo{
        passwordHash: "$argon2id$v=19$...",
    }
    accountRepo := &MockAccountRepo{
        accountID: 100,
        userID:    200,
        enabled:   true,
        locked:    false,
    }
    hasher := &MockHasher{
        pepper: "test_pepper",
    }
    
    strategy := service.NewPasswordAuthStrategy(
        credRepo,
        accountRepo,
        hasher,
    )
    
    // 测试成功场景
    decision, err := strategy.Authenticate(ctx, domain.AuthInput{
        Username: "alice",
        Password: "correct",
    })
    
    assert.NoError(t, err)
    assert.True(t, decision.OK)
    assert.Equal(t, 100, decision.Principal.AccountID)
    assert.Contains(t, decision.Principal.AMR, "pwd")
}
```

## 🔄 扩展新认证方式

假设要添加"GitHub OAuth"认证：

### 1. 在 types.go 添加场景

```go
const (
    AuthGitHub Scenario = "oauth_github"
)

const (
    AMRGitHub AMR = "github"
)
```

### 2. 在 IdentityProvider 添加方法

```go
type IdentityProvider interface {
    // ...existing methods...
    ExchangeGitHubCode(ctx context.Context, code string) (githubID, email string, err error)
}
```

### 3. 实现策略

```go
// service/github.go
type OAuthGitHubAuthStrategy struct {
    scenario    domain.Scenario
    credRepo    port.CredentialRepository
    accountRepo port.AccountRepository
    idp         port.IdentityProvider
}

func (g *OAuthGitHubAuthStrategy) Authenticate(ctx context.Context, in domain.AuthInput) (domain.AuthDecision, error) {
    // 1. 调用GitHub API
    githubID, email, err := g.idp.ExchangeGitHubCode(ctx, in.GitHubCode)
    
    // 2. 查找绑定（复用统一接口）
    accountID, userID, credID, err := g.credRepo.FindOAuthCredential(
        ctx, "github", "default", githubID,
    )
    
    // 3. 检查账户状态
    enabled, locked, _ := g.accountRepo.GetAccountStatus(ctx, accountID)
    
    // 4. 返回判决
    // ...
}
```

### 4. 注册到工厂

```go
func (f *StrategyFactory) CreateStrategy(scenario domain.Scenario) domain.AuthStrategy {
    switch scenario {
    // ...existing cases...
    case domain.AuthGitHub:
        return NewOAuthGitHubAuthStrategy(f.credRepo, f.accountRepo, f.idp)
    }
}
```

**无需修改任何端口定义！** ✅

## 📈 设计优势总结

| 方面 | 改进前 | 改进后 |
|------|--------|--------|
| **端口数量** | 每种认证一个Deps接口（4个） | 5个按职责划分的端口 |
| **扩展性** | 新增认证需要新增Deps | 复用现有端口 |
| **测试性** | 需要mock整个Deps | 只mock需要的端口 |
| **职责** | 混合数据查询和算法 | 职责单一 |
| **领域语言** | 暴露技术细节（PHC） | 使用领域概念 |
| **适配器实现** | 碎片化 | 集中实现 |

## 🎁 关键收益

1. ✅ **清晰的边界**：领域层不知道MySQL/Redis/HTTP的存在
2. ✅ **易于替换**：可以轻松切换Argon2→Bcrypt、MySQL→PostgreSQL
3. ✅ **单一职责**：每个端口只关注一个领域概念
4. ✅ **易于测试**：可以独立mock每个端口
5. ✅ **可扩展**：新增认证方式不需要修改端口定义
6. ✅ **符合DDD**：端口定义表达领域需求，而非技术实现

这就是**六边形架构**的精髓：**领域层定义自己需要什么，基础设施层去实现**！
