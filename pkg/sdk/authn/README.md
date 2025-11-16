# AuthN SDK

IAM 认证服务的 Go 客户端 SDK，提供 JWT 验证和 gRPC 认证服务调用功能。

## 功能特性

- ✅ **本地 JWT 验证**：使用 JWKS 在本地验证 JWT 签名
- ✅ **远程验证**：可选的通过 gRPC 调用 IAM 服务进行验证
- ✅ **JWKS 自动管理**：自动下载、缓存和刷新 JWKS
- ✅ **ETag 支持**：使用 HTTP ETag 优化 JWKS 缓存
- ✅ **多种密钥类型**：支持 RSA 和 EC 密钥
- ✅ **时钟偏差容忍**：自动处理时间不同步问题
- ✅ **Audience/Issuer 验证**：配置白名单验证 JWT 声明
- ✅ **结构化日志**：完整的操作日志记录
- ✅ **线程安全**：所有操作都是并发安全的

## 安装

```bash
go get github.com/FangcunMount/iam-contracts/pkg/sdk/authn
```

## 快速开始

### 基础用法：仅本地验证

```go
package main

import (
    "context"
    "fmt"
    "log"

    authnsdk "github.com/FangcunMount/iam-contracts/pkg/sdk/authn"
)

func main() {
    // 1. 配置 SDK
    cfg := authnsdk.Config{
        JWKSURL: "https://iam.example.com/.well-known/jwks.json",
        AllowedAudience: []string{"my-app"},
        AllowedIssuer:   "https://iam.example.com",
    }

    // 2. 创建验证器（不传 client 则仅本地验证）
    verifier, err := authnsdk.NewVerifier(cfg, nil)
    if err != nil {
        log.Fatal(err)
    }

    // 3. 验证 JWT
    ctx := context.Background()
    token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
    
    resp, err := verifier.Verify(ctx, token, nil)
    if err != nil {
        log.Fatalf("验证失败: %v", err)
    }

    // 4. 使用验证结果
    fmt.Printf("用户 ID: %s\n", resp.Claims.UserId)
    fmt.Printf("租户 ID: %s\n", resp.Claims.TenantId)
    fmt.Printf("过期时间: %v\n", resp.Claims.ExpiresAt.AsTime())
}
```

### 高级用法：本地 + 远程验证

```go
package main

import (
    "context"
    "fmt"
    "log"

    authnsdk "github.com/FangcunMount/iam-contracts/pkg/sdk/authn"
)

func main() {
    ctx := context.Background()

    // 1. 创建 gRPC 客户端
    client, err := authnsdk.NewClient(ctx, "iam.example.com:8081")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 2. 配置 SDK
    cfg := authnsdk.Config{
        GRPCEndpoint: "iam.example.com:8081",
        JWKSURL:      "https://iam.example.com/.well-known/jwks.json",
        
        // JWT 验证配置
        AllowedAudience: []string{"my-app", "admin-panel"},
        AllowedIssuer:   "https://iam.example.com",
        
        // JWKS 缓存配置
        JWKSRefreshInterval: 5 * time.Minute,  // 5 分钟刷新一次
        JWKSRequestTimeout:  3 * time.Second,  // 请求超时 3 秒
        JWKSCacheTTL:        10 * time.Minute, // 缓存 10 分钟
        
        // 时钟偏差容忍
        ClockSkew: time.Minute, // 容忍 1 分钟的时间差
        
        // 远程验证（可选）
        ForceRemoteVerification: false, // 仅本地验证失败时调用远程
    }

    // 3. 创建验证器
    verifier, err := authnsdk.NewVerifier(cfg, client)
    if err != nil {
        log.Fatal(err)
    }

    // 4. 验证 JWT
    token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
    
    resp, err := verifier.Verify(ctx, token, nil)
    if err != nil {
        log.Fatalf("验证失败: %v", err)
    }

    // 5. 使用验证结果
    fmt.Printf("验证状态: %v\n", resp.Status)
    fmt.Printf("用户 ID: %s\n", resp.Claims.UserId)
    fmt.Printf("账号 ID: %s\n", resp.Claims.AccountId)
    fmt.Printf("租户 ID: %s\n", resp.Claims.TenantId)
    fmt.Printf("Subject: %s\n", resp.Claims.Subject)
    fmt.Printf("签发时间: %v\n", resp.Claims.IssuedAt.AsTime())
    fmt.Printf("过期时间: %v\n", resp.Claims.ExpiresAt.AsTime())
    
    // 6. 访问自定义属性
    for key, value := range resp.Claims.Attributes {
        fmt.Printf("属性 %s: %s\n", key, value)
    }
}
```

### 强制远程验证

```go
// 方式 1: 全局配置强制远程验证
cfg := authnsdk.Config{
    JWKSURL:                 "https://iam.example.com/.well-known/jwks.json",
    ForceRemoteVerification: true, // 始终调用远程验证
}

// 方式 2: 单次调用强制远程验证
resp, err := verifier.Verify(ctx, token, &authnsdk.VerifyOptions{
    ForceRemote: true, // 仅此次调用使用远程验证
})
```

## 配置说明

### Config 结构

```go
type Config struct {
    // gRPC 服务地址（可选，仅远程验证需要）
    GRPCEndpoint string

    // JWKS 端点 URL（必需）
    JWKSURL string

    // JWKS 缓存刷新间隔（默认：5 分钟）
    JWKSRefreshInterval time.Duration

    // JWKS HTTP 请求超时（默认：3 秒）
    JWKSRequestTimeout time.Duration

    // JWKS 缓存 TTL（默认：10 分钟）
    JWKSCacheTTL time.Duration

    // 允许的 JWT 受众列表（可选）
    AllowedAudience []string

    // 允许的 JWT 签发者（可选）
    AllowedIssuer string

    // 时钟偏差容忍度（默认：60 秒）
    ClockSkew time.Duration

    // 强制远程验证（默认：false）
    ForceRemoteVerification bool
}
```

### 默认值

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `JWKSRefreshInterval` | 5 分钟 | JWKS 主动刷新间隔 |
| `JWKSRequestTimeout` | 3 秒 | HTTP 请求超时时间 |
| `JWKSCacheTTL` | 10 分钟 | 缓存最大生存时间 |
| `ClockSkew` | 60 秒 | 时钟偏差容忍度 |
| `ForceRemoteVerification` | false | 是否强制远程验证 |

## 验证流程

### 本地验证流程

1. **解析 JWT**：提取 header 和 claims
2. **查找密钥**：根据 kid 从 JWKS 查找公钥
3. **验证签名**：使用公钥验证 JWT 签名
4. **验证时间**：检查 exp（过期时间）和 nbf（生效时间）
5. **验证声明**：检查 audience 和 issuer
6. **返回结果**：返回解析后的 claims

### 远程验证流程

1. **本地验证**：先执行完整的本地验证
2. **调用 gRPC**：调用 IAM 的 VerifyToken RPC
3. **返回结果**：使用远程验证的结果

## JWKS 管理

### 自动刷新策略

1. **主动刷新**：根据 `JWKSRefreshInterval` 定期刷新
2. **懒加载**：首次使用时自动下载
3. **缓存未命中**：密钥未找到时触发刷新
4. **ETag 优化**：使用 HTTP ETag 避免重复下载

### 密钥查找逻辑

```
1. 从缓存查找 kid
   ├─ 找到 → 返回密钥
   └─ 未找到 → 刷新 JWKS → 重试查找
```

### 支持的密钥类型

| 类型 | 算法 | 说明 |
|------|------|------|
| RSA | RS256, RS384, RS512 | RSA 签名 |
| EC | ES256, ES384, ES512 | 椭圆曲线签名 |

## 错误处理

### 常见错误

```go
resp, err := verifier.Verify(ctx, token, nil)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "token is expired"):
        // Token 已过期
        log.Println("Token 已过期，请重新登录")
        
    case strings.Contains(err.Error(), "kid") && strings.Contains(err.Error(), "not found"):
        // 密钥未找到
        log.Println("签名密钥未找到，可能是旧的 Token")
        
    case strings.Contains(err.Error(), "signature"):
        // 签名验证失败
        log.Println("Token 签名无效")
        
    case strings.Contains(err.Error(), "audience"):
        // Audience 不匹配
        log.Println("Token 的 audience 不匹配")
        
    case strings.Contains(err.Error(), "issuer"):
        // Issuer 不匹配
        log.Println("Token 的 issuer 不匹配")
        
    default:
        log.Printf("验证失败: %v", err)
    }
    return
}
```

### 最佳实践

1. **错误日志**：记录验证失败的详细信息
2. **重试逻辑**：网络错误时可以重试
3. **降级策略**：JWKS 服务不可用时的降级处理
4. **监控告警**：监控验证失败率和 JWKS 刷新失败

## 日志

SDK 使用项目统一的日志库，所有日志都带有 `[AuthN SDK]` 前缀。

### 日志级别

- **Info**：重要操作（连接、初始化、验证成功）
- **Debug**：详细信息（Token 解析、密钥查找、缓存操作）
- **Warn**：警告信息（验证失败、密钥未找到）
- **Error**：错误信息（连接失败、HTTP 请求失败）

### 日志示例

```
INFO  [AuthN SDK] Connecting to IAM authn gRPC endpoint: localhost:8081
INFO  [AuthN SDK] Successfully connected to IAM authn gRPC endpoint
INFO  [AuthN SDK] Initializing JWKS manager with URL: https://iam.example.com/.well-known/jwks.json, refresh interval: 5m0s
DEBUG [AuthN SDK] Refreshing JWKS from https://iam.example.com/.well-known/jwks.json
INFO  [AuthN SDK] Successfully refreshed JWKS, loaded 2 keys
DEBUG [AuthN SDK] Starting token verification
DEBUG [AuthN SDK] Local verification successful, subject: user123, user_id: 456
INFO  [AuthN SDK] Token verification completed successfully
```

详细的日志说明请参考 [LOGGING.md](./LOGGING.md)。

## 性能优化

### JWKS 缓存

- ✅ 内存缓存，访问速度快
- ✅ 支持 HTTP ETag，减少网络传输
- ✅ 自动过期刷新，无需手动管理
- ✅ 读写锁保护，支持高并发

### 验证性能

- 本地验证延迟：< 1ms（缓存命中）
- 远程验证延迟：取决于 gRPC 网络延迟
- JWKS 刷新延迟：< 100ms（ETag 命中）

### 并发安全

所有操作都是线程安全的，可以在多个 goroutine 中并发调用：

```go
verifier, _ := authnsdk.NewVerifier(cfg, client)

// 并发验证
for i := 0; i < 100; i++ {
    go func() {
        resp, err := verifier.Verify(ctx, token, nil)
        // 处理结果...
    }()
}
```

## 使用场景

### 1. API 网关

在 API 网关中验证客户端请求的 JWT：

```go
func authMiddleware(verifier *authnsdk.Verifier) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 提取 Authorization header
        auth := c.GetHeader("Authorization")
        if !strings.HasPrefix(auth, "Bearer ") {
            c.AbortWithStatusJSON(401, gin.H{"error": "missing token"})
            return
        }
        
        token := strings.TrimPrefix(auth, "Bearer ")
        
        // 验证 token
        resp, err := verifier.Verify(c.Request.Context(), token, nil)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
            return
        }
        
        // 将用户信息存入上下文
        c.Set("user_id", resp.Claims.UserId)
        c.Set("tenant_id", resp.Claims.TenantId)
        
        c.Next()
    }
}
```

### 2. 微服务认证

在微服务间传递和验证 JWT：

```go
func callServiceWithAuth(ctx context.Context, token string) error {
    // 验证上游传来的 token
    resp, err := verifier.Verify(ctx, token, nil)
    if err != nil {
        return fmt.Errorf("认证失败: %w", err)
    }
    
    // 使用验证后的信息调用下游服务
    req, _ := http.NewRequestWithContext(ctx, "GET", "http://service-b/api", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("X-User-ID", resp.Claims.UserId)
    
    // 发送请求...
    return nil
}
```

### 3. gRPC 拦截器

在 gRPC 服务中验证客户端的 JWT：

```go
func authInterceptor(verifier *authnsdk.Verifier) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // 提取 metadata 中的 token
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "missing metadata")
        }
        
        tokens := md.Get("authorization")
        if len(tokens) == 0 {
            return nil, status.Error(codes.Unauthenticated, "missing token")
        }
        
        token := strings.TrimPrefix(tokens[0], "Bearer ")
        
        // 验证 token
        resp, err := verifier.Verify(ctx, token, nil)
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }
        
        // 将用户信息注入上下文
        ctx = context.WithValue(ctx, "user_id", resp.Claims.UserId)
        ctx = context.WithValue(ctx, "tenant_id", resp.Claims.TenantId)
        
        return handler(ctx, req)
    }
}
```

## 测试

### 单元测试

```go
func TestVerifier(t *testing.T) {
    // Mock JWKS 服务器
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "keys": []map[string]interface{}{
                {
                    "kty": "RSA",
                    "kid": "test-key",
                    "n":   "...",
                    "e":   "AQAB",
                },
            },
        })
    }))
    defer server.Close()
    
    cfg := authnsdk.Config{
        JWKSURL: server.URL,
    }
    
    verifier, err := authnsdk.NewVerifier(cfg, nil)
    assert.NoError(t, err)
    
    // 测试验证...
}
```

### 集成测试

完整的集成测试示例请参考 `examples/` 目录。

## 示例代码

更多示例代码请查看：

- [examples/basic](./examples/basic/) - 基础用法示例
- [examples/middleware](./examples/middleware/) - 中间件集成示例
- [examples/grpc](./examples/grpc/) - gRPC 拦截器示例

## 故障排查

### 问题：JWKS 下载失败

**症状**：日志显示 "JWKS fetch failed"

**解决方案**：

1. 检查 JWKS URL 是否正确
2. 检查网络连接
3. 检查防火墙规则
4. 增加 `JWKSRequestTimeout`

### 问题：密钥未找到

**症状**：错误信息 "kid xxx not found in jwks"

**解决方案**：

1. 检查 Token 是否来自正确的 IAM 实例
2. 等待 JWKS 缓存刷新（默认 5 分钟）
3. 手动触发 JWKS 刷新（重启应用）

### 问题：Token 验证失败

**症状**：签名验证失败

**解决方案**：

1. 检查 Token 是否被篡改
2. 检查 Token 是否过期
3. 检查 JWKS 是否是最新的
4. 检查系统时间是否正确

### 问题：性能问题

**症状**：验证速度慢

**解决方案**：

1. 检查 JWKS 缓存是否生效
2. 减少 `JWKSRefreshInterval`
3. 使用本地验证，避免远程调用
4. 检查网络延迟

## 相关文档

- [日志说明](./LOGGING.md) - 详细的日志输出说明
- [API 文档](https://pkg.go.dev/github.com/FangcunMount/iam-contracts/pkg/sdk/authn) - GoDoc API 文档
- [IAM 系统文档](../../../docs/) - IAM 系统整体文档

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](../../../LICENSE) 文件。

## 贡献

欢迎提交 Issue 和 Pull Request！

## 更新日志

### v1.0.0 (2024-01-16)

- ✅ 初始版本发布
- ✅ 支持本地 JWT 验证
- ✅ 支持远程 gRPC 验证
- ✅ JWKS 自动管理
- ✅ 完整的日志记录
- ✅ 中文注释和文档
