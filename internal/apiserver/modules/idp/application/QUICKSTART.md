# IDP 模块应用服务层 - 快速开始

## 简介

IDP（Identity Provider）模块的应用服务层提供了微信应用管理和微信认证的业务用例接口。

## 安装

应用服务层依赖领域层，确保领域层已完成：

```bash
# 检查领域层
ls internal/apiserver/modules/idp/domain/
# 应该看到：wechatapp/ wechatsession/
```

## 快速使用

### 1. 创建依赖实例

```go
import (
    "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application"
    wechatappservice "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp/service"
    wechatsessionservice "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession/service"
)

// 创建领域服务实例（这些通常由基础设施层提供）
creator := wechatappservice.NewWechatAppCreator(querier)
querier := wechatappservice.NewWechatAppQuerier(repo)
rotater := wechatappservice.NewCredentialRotater()
tokenCacher := wechatappservice.NewAccessTokenCacher()
authenticator := wechatsessionservice.NewAuthenticator()

// 准备依赖
deps := application.ApplicationServicesDependencies{
    WechatAppRepo:       repo,           // 由基础设施层提供
    WechatAppCreator:    creator,
    WechatAppQuerier:    querier,
    CredentialRotater:   rotater,
    AccessTokenCacher:   tokenCacher,
    AppTokenProvider:    provider,       // 由基础设施层提供
    AccessTokenCache:    cache,          // 由基础设施层提供
    WechatAuthenticator: authenticator,
}

// 创建所有应用服务
appServices := application.NewApplicationServices(deps)
```

### 2. 使用应用服务

#### 2.1 创建微信应用

```go
import (
    "context"
    "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application/wechatapp"
    domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp"
)

ctx := context.Background()

// 创建微信小程序
dto := wechatapp.CreateWechatAppDTO{
    AppID:     "wx1234567890abcdef",
    Name:      "我的小程序",
    Type:      domain.MiniProgram,
    AppSecret: "your-app-secret-here",
}

result, err := appServices.WechatApp.CreateApp(ctx, dto)
if err != nil {
    log.Fatalf("创建失败: %v", err)
}

fmt.Printf("创建成功: %s (ID: %s)\n", result.Name, result.ID)
```

#### 2.2 获取访问令牌

```go
// 获取访问令牌（自动缓存，过期自动刷新）
token, err := appServices.WechatAppToken.GetAccessToken(ctx, "wx1234567890abcdef")
if err != nil {
    log.Fatalf("获取失败: %v", err)
}

fmt.Printf("访问令牌: %s\n", token)
```

#### 2.3 微信登录

```go
import (
    "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application/wechatsession"
)

// 微信小程序登录
loginDTO := wechatsession.LoginWithCodeDTO{
    AppID:  "wx1234567890abcdef",
    JSCode: "071AbcDef123456789", // 前端 wx.login() 获取
}

result, err := appServices.WechatAuth.LoginWithCode(ctx, loginDTO)
if err != nil {
    log.Fatalf("登录失败: %v", err)
}

fmt.Printf("OpenID: %s\n", result.OpenID)
if result.UnionID != nil {
    fmt.Printf("UnionID: %s\n", *result.UnionID)
}
```

#### 2.4 轮换密钥

```go
// 轮换 AppSecret（定期或应急）
err := appServices.WechatAppCredential.RotateAuthSecret(
    ctx,
    "wx1234567890abcdef",
    "new-app-secret-32-chars-here",
)
if err != nil {
    log.Fatalf("轮换失败: %v", err)
}

fmt.Println("密钥轮换成功")
```

## 应用服务列表

### 微信应用管理

| 服务 | 方法 | 说明 |
|------|------|------|
| **WechatAppApplicationService** | `CreateApp()` | 创建微信应用 |
| | `GetApp()` | 查询微信应用 |
| **WechatAppCredentialApplicationService** | `RotateAuthSecret()` | 轮换认证密钥 |
| | `RotateMsgSecret()` | 轮换消息加解密密钥 |
| **WechatAppTokenApplicationService** | `GetAccessToken()` | 获取访问令牌（缓存） |
| | `RefreshAccessToken()` | 强制刷新访问令牌 |

### 微信认证

| 服务 | 方法 | 说明 |
|------|------|------|
| **WechatAuthApplicationService** | `LoginWithCode()` | 微信登录 |
| | `DecryptUserPhone()` | 解密手机号 |

## 完整示例

### 示例 1: 微信小程序登录流程

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application"
    "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application/wechatsession"
)

func main() {
    // 1. 初始化应用服务（实际项目中由 DI 容器提供）
    appServices := initApplicationServices()

    ctx := context.Background()

    // 2. 微信登录
    loginDTO := wechatsession.LoginWithCodeDTO{
        AppID:  "wx1234567890abcdef",
        JSCode: "071AbcDef123456789",
    }

    loginResult, err := appServices.WechatAuth.LoginWithCode(ctx, loginDTO)
    if err != nil {
        log.Fatalf("登录失败: %v", err)
    }

    fmt.Printf("登录成功:\n")
    fmt.Printf("  OpenID: %s\n", loginResult.OpenID)
    if loginResult.UnionID != nil {
        fmt.Printf("  UnionID: %s\n", *loginResult.UnionID)
    }

    // 3. 后续业务逻辑
    // - 根据 OpenID 查找或创建用户（调用 UC 模块）
    // - 生成业务系统的 JWT Token（调用 AUTHN 模块）
    // - 返回给前端
}

func initApplicationServices() *application.ApplicationServices {
    // 实际项目中，这些依赖由基础设施层提供
    // 这里只是示例
    deps := application.ApplicationServicesDependencies{
        // ... 注入各种依赖
    }
    return application.NewApplicationServices(deps)
}
```

### 示例 2: 定期轮换密钥（定时任务）

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application"
)

func main() {
    appServices := initApplicationServices()
    ctx := context.Background()

    // 定时任务：每 90 天轮换一次 AppSecret
    ticker := time.NewTicker(90 * 24 * time.Hour)
    defer ticker.Stop()

    for range ticker.C {
        // 生成新的 AppSecret
        newSecret := generateSecureAppSecret()

        // 轮换密钥
        err := appServices.WechatAppCredential.RotateAuthSecret(
            ctx,
            "wx1234567890abcdef",
            newSecret,
        )
        if err != nil {
            log.Printf("轮换失败: %v", err)
            continue
        }

        fmt.Println("定期轮换 AppSecret 成功")

        // 通知相关人员
        // notifyAdmins("AppSecret has been rotated")
    }
}

func generateSecureAppSecret() string {
    // 实现：使用加密安全的随机数生成器
    return "new-secure-secret"
}

func initApplicationServices() *application.ApplicationServices {
    // ...
    return nil
}
```

### 示例 3: 访问令牌性能优化（缓存）

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
)

func main() {
    appServices := initApplicationServices()
    ctx := context.Background()
    appID := "wx1234567890abcdef"

    // 并发获取访问令牌（测试缓存和单飞机制）
    var wg sync.WaitGroup
    start := time.Now()

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()

            token, err := appServices.WechatAppToken.GetAccessToken(ctx, appID)
            if err != nil {
                fmt.Printf("协程 %d: 获取失败 %v\n", n, err)
                return
            }

            fmt.Printf("协程 %d: 获取成功 %s...\n", n, token[:20])
        }(i)
    }

    wg.Wait()
    duration := time.Since(start)

    // 由于缓存和单飞机制，100 个并发请求应该非常快完成
    fmt.Printf("100 个并发请求完成，耗时: %v\n", duration)
    // 预期输出：耗时不到 1 秒（只调用一次微信 API）
}

func initApplicationServices() *application.ApplicationServices {
    // ...
    return nil
}
```

## 错误处理

应用服务层会包装领域层错误，提供更友好的错误信息：

```go
result, err := appServices.WechatApp.CreateApp(ctx, dto)
if err != nil {
    // 错误信息示例：
    // - "failed to create wechat app: appID cannot be empty"
    // - "failed to create wechat app: wechat app with the given appID already exists"
    // - "failed to set auth secret: invalid app secret"
    log.Printf("创建失败: %v", err)
    return
}
```

建议在接口层进一步处理错误，转换为 HTTP 状态码和错误响应。

## 测试

### 单元测试（使用 Mock）

```go
package wechatapp_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"

    "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application/wechatapp"
    domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp"
)

// Mock 实现
type MockCreator struct {
    mock.Mock
}

func (m *MockCreator) Create(ctx context.Context, appID, name string, appType domain.AppType) (*domain.WechatApp, error) {
    args := m.Called(ctx, appID, name, appType)
    return args.Get(0).(*domain.WechatApp), args.Error(1)
}

// 测试用例
func TestCreateApp(t *testing.T) {
    // 准备 Mock
    mockCreator := new(MockCreator)
    mockQuerier := new(MockQuerier)
    mockRepo := new(MockRepository)
    mockRotater := new(MockRotater)

    // 设置期望
    expectedApp := &domain.WechatApp{
        AppID: "wx123",
        Name:  "Test App",
        Type:  domain.MiniProgram,
    }
    mockCreator.On("Create", mock.Anything, "wx123", "Test App", domain.MiniProgram).
        Return(expectedApp, nil)
    mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

    // 创建应用服务
    service := wechatapp.NewWechatAppApplicationService(
        mockRepo, mockCreator, mockQuerier, mockRotater,
    )

    // 执行测试
    dto := wechatapp.CreateWechatAppDTO{
        AppID: "wx123",
        Name:  "Test App",
        Type:  domain.MiniProgram,
    }
    result, err := service.CreateApp(context.Background(), dto)

    // 验证结果
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "wx123", result.AppID)
    assert.Equal(t, "Test App", result.Name)

    // 验证 Mock 调用
    mockCreator.AssertExpectations(t)
    mockRepo.AssertExpectations(t)
}
```

## 常见问题

### Q1: 如何获取应用服务实例？

A: 应用服务应该通过依赖注入容器获取，不要手动 new。推荐使用 Wire 或类似工具。

### Q2: 应用服务可以直接访问数据库吗？

A: 不可以。应用服务应该通过仓储接口（Repository）访问数据，仓储的具体实现由基础设施层提供。

### Q3: 应用服务方法应该返回领域对象还是 DTO？

A: 应该返回 DTO。DTO 用于隔离应用层和接口层，避免暴露领域对象。

### Q4: 如何处理事务？

A: 应用服务方法是事务的自然边界。可以使用工作单元（UoW）模式或在仓储层管理事务。

### Q5: 多个应用服务可以相互调用吗？

A: 不推荐。应用服务应该是独立的用例，如果需要复用逻辑，应该提取到领域服务层。

## 下一步

1. **实现基础设施层** - 提供仓储、缓存、外部 API 等具体实现
2. **实现接口层** - 暴露 HTTP/gRPC API
3. **配置依赖注入** - 使用 Wire 或类似工具组装各层
4. **编写测试** - 单元测试、集成测试、E2E 测试

## 参考文档

- [应用服务层详细设计文档](./README.md)
- [使用示例](./examples_test.go)
- [完成总结](./SUMMARY.md)

## 联系

如有问题，请联系开发团队。
