# 快速开始

## 🎯 30 秒搞懂

### 金字塔概览

```text
                        👤 你的应用
                            ↓
              ┌─────────────────────────┐
              │     sdk.Client          │  ← 统一入口
              │  (一个连接，全部服务)     │
              └─────────────────────────┘
                    ↓   ↓   ↓
        ┌───────────┼───┼───────────┐
        ↓           ↓   ↓           ↓
    Auth()    Identity()    Guardianship()
    认证服务    身份服务      监护关系服务
        ↓           ↓               ↓
    验证Token   用户/角色管理    监护关系查询
```

### 核心概念

| 概念 | 说明 | 3 秒记忆 |
|------|------|---------|
| **Client** | 统一客户端 | 一个连接，访问所有服务 |
| **Config** | 配置对象 | 地址 + TLS + 重试 |
| **Auth** | 认证服务 | 验证/刷新/撤销 Token |
| **Identity** | 身份服务 | 用户/角色/部门管理 |
| **Guardianship** | 监护关系 | 家长-孩子关系查询 |

### 3 行代码开始

```go
// 1️⃣ 创建客户端
client, _ := sdk.NewClient(ctx, &sdk.Config{Endpoint: "localhost:8081"})

// 2️⃣ 使用服务
user, _ := client.Identity().GetUser(ctx, "user-123")

// 3️⃣ 完成！
log.Printf("用户: %s", user.GetProfile().GetDisplayName())
```

### 使用流程

```text
┌─────────────────────────────────────────────────────────┐
│ 1. 配置                                                  │
│    Config{Endpoint, TLS, Retry, ...}                    │
└────────────┬────────────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────────────┐
│ 2. 创建客户端                                             │
│    client := sdk.NewClient(ctx, config)                 │
└────────────┬────────────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────────────┐
│ 3. 调用服务                                              │
│    client.Auth().VerifyToken(...)                       │
│    client.Identity().GetUser(...)                       │
│    client.Guardianship().IsGuardian(...)                │
└─────────────────────────────────────────────────────────┘
```

---

## 📦 安装

```bash
go get github.com/FangcunMount/iam-contracts/pkg/sdk
```

## 最简示例

```go
package main

import (
    "context"
    "log"
    
    sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"
)

func main() {
    ctx := context.Background()
    
    // 创建客户端
    client, err := sdk.NewClient(ctx, &sdk.Config{
        Endpoint: "localhost:8081",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // 验证 Token
    result, err := client.Auth().VerifyToken(ctx, &authnv1.VerifyTokenRequest{
        AccessToken: "your-token-here",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Token verified: %+v", result)
}
```

## 从环境变量加载配置

```bash
# 设置环境变量
export IAM_ENDPOINT="iam.example.com:8081"
export IAM_TLS_ENABLED="true"
export IAM_TLS_CA_CERT="/etc/iam/certs/ca.crt"
export IAM_TIMEOUT="30s"
```

```go
package main

import (
    "context"
    "log"
    
    sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"
)

func main() {
    // 从环境变量加载
    cfg, err := sdk.ConfigFromEnv()
    if err != nil {
        log.Fatal(err)
    }
    
    client, err := sdk.NewClient(context.Background(), cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // 使用客户端...
}
```

## 常见配置场景

### 1. 开发环境（无 TLS）

```go
client, _ := sdk.NewClient(ctx, &sdk.Config{
    Endpoint: "localhost:8081",
    TLS: &sdk.TLSConfig{
        Enabled: false,
    },
})
```

### 2. 测试环境（TLS，跳过验证）

```go
client, _ := sdk.NewClient(ctx, &sdk.Config{
    Endpoint: "iam-test.example.com:8081",
    TLS: &sdk.TLSConfig{
        Enabled:            true,
        InsecureSkipVerify: true, // 仅用于测试！
    },
})
```

### 3. 生产环境（mTLS）

```go
client, _ := sdk.NewClient(ctx, &sdk.Config{
    Endpoint: "iam.example.com:8081",
    TLS: &sdk.TLSConfig{
        Enabled:    true,
        CACert:     "/etc/iam/certs/ca.crt",
        ClientCert: "/etc/iam/certs/client.crt",
        ClientKey:  "/etc/iam/certs/client.key",
        ServerName: "iam.example.com",
        MinVersion: tls.VersionTLS13,
    },
    Timeout:     30 * time.Second,
    DialTimeout: 10 * time.Second,
})
```

## 基础操作示例

### 认证服务

```go
// 验证 Token
resp, err := client.Auth().VerifyToken(ctx, &authnv1.VerifyTokenRequest{
    AccessToken: token,
})

// 刷新 Token
resp, err := client.Auth().RefreshToken(ctx, &authnv1.RefreshTokenRequest{
    RefreshToken: refreshToken,
})

// 撤销 Token
_, err := client.Auth().RevokeToken(ctx, &authnv1.RevokeTokenRequest{
    Token: token,
})
```

### 身份服务

```go
// 获取用户
user, err := client.Identity().GetUser(ctx, "user-id-123")

// 创建用户
user, err := client.Identity().CreateUser(ctx, &identityv1.CreateUserRequest{
    User: &identityv1.User{
        Profile: &identityv1.UserProfile{
            DisplayName: "张三",
            Email:       "zhangsan@example.com",
        },
    },
})

// 批量获取用户
users, err := client.Identity().BatchGetUsers(ctx, []string{"user-1", "user-2"})
```

### 监护关系服务

```go
// 检查监护关系
isGuardian, err := client.Guardianship().IsGuardian(ctx, "parent-id", "child-id")

// 列举被监护人
children, err := client.Guardianship().ListChildren(ctx, "parent-id")
```

## 错误处理

```go
import (
    sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
)

user, err := client.Identity().GetUser(ctx, "user-123")
if err != nil {
    switch {
    case errors.IsNotFound(err):
        log.Println("用户不存在")
    case errors.IsUnauthorized(err):
        log.Println("未认证，请重新登录")
    case errors.IsPermissionDenied(err):
        log.Println("权限不足")
    case errors.IsServiceUnavailable(err):
        log.Println("服务暂时不可用，请稍后重试")
    default:
        log.Printf("未知错误: %v", err)
    }
    return
}

log.Printf("用户: %s", user.GetProfile().GetDisplayName())
```

## 下一步

- [配置详解](./02-configuration.md) - 了解所有配置选项
- [JWT 验证](./03-jwt-verification.md) - 本地 JWT 验证
- [服务间认证](./04-service-auth.md) - 自动化服务间 Token 管理
- [可观测性](./05-observability.md) - Metrics 和 Tracing
- [错误处理](./06-error-handling.md) - 统一错误处理最佳实践
