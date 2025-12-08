# IAM SDK 文档索引

完整的 IAM SDK 使用文档。

## 📚 文档列表

### 入门指南

1. **[快速开始](./01-quick-start.md)**
   - 安装和基础示例
   - 环境变量配置
   - 常见配置场景
   - 基础操作示例

2. **[配置详解](./02-configuration.md)**
   - 完整配置参数说明
   - TLS/mTLS 配置
   - 重试和超时配置
   - JWKS 配置
   - 熔断器配置
   - 环境变量映射
   - YAML 配置文件

### 核心功能

1. **[JWT 本地验证](./03-jwt-verification.md)**
   - TokenVerifier 使用
   - JWKS Manager 配置
   - 验证策略（Strategy 模式）
   - JWKS 职责链（Chain of Responsibility）
   - 性能优化
   - 监控和统计

2. **[服务间认证](./04-service-auth.md)**
   - ServiceAuthHelper 基础用法
   - 自动 Token 刷新
   - Jitter 和退避策略
   - 熔断保护
   - 状态监控
   - 生产环境最佳实践

3. **[可观测性](./05-observability.md)**
   - Prometheus Metrics 集成
   - OpenTelemetry Tracing
   - 请求 ID 传播
   - 自定义 Metrics 和 Tracing

4. **[错误处理](./06-error-handling.md)**
   - 统一错误类型
   - 错误分类和匹配
   - 错误处理链
   - 最佳实践

### 高级主题

1. **[方法级重试配置](./07-advanced-retry.md)**
   - 按方法定制重试策略
   - 自定义可重试错误码
   - 重试判断函数
   - 预定义策略模板

2. **[传输层配置](./08-transport.md)**
   - gRPC 连接管理
   - 拦截器链
   - 负载均衡
   - Keepalive 配置

3. **[设计模式](./09-design-patterns.md)**
   - Chain of Responsibility（JWKS）
   - Strategy（TokenVerifier）
   - Builder（配置构建）
   - Observer（回调钩子）

### API 参考

1. **[认证服务 API](./api/auth.md)**
    - VerifyToken
    - RefreshToken
    - RevokeToken
    - IssueServiceToken
    - GetJWKS

2. **[身份服务 API](./api/identity.md)**
    - GetUser / ListUsers / BatchGetUsers
    - CreateUser / UpdateUser / DeleteUser
    - GetRole / ListRoles
    - GetDepartment / ListDepartments

3. **[监护关系 API](./api/guardianship.md)**
    - IsGuardian
    - ListChildren / ListGuardians
    - AddGuardianship / RemoveGuardianship

## 🎯 快速导航

### 我想

- **快速开始使用 SDK** → [快速开始](./01-quick-start.md)
- **配置 mTLS** → [配置详解 - TLS 配置](./02-configuration.md#tls-配置)
- **本地验证 JWT** → [JWT 本地验证](./03-jwt-verification.md)
- **实现服务间认证** → [服务间认证](./04-service-auth.md)
- **添加 Metrics 监控** → [可观测性](./05-observability.md)
- **处理特定错误** → [错误处理](./06-error-handling.md)
- **定制重试策略** → [方法级重试配置](./07-advanced-retry.md)
- **了解设计原理** → [设计模式](./09-design-patterns.md)

### 场景索引

#### 开发环境

```go
// 最简配置
client, _ := sdk.NewClient(ctx, &sdk.Config{
    Endpoint: "localhost:8081",
})
```

→ [快速开始](./01-quick-start.md#最简示例)

#### 测试环境

```go
// TLS 但跳过验证
client, _ := sdk.NewClient(ctx, &sdk.Config{
    Endpoint: "iam-test.example.com:8081",
    TLS: &sdk.TLSConfig{
        Enabled:            true,
        InsecureSkipVerify: true,
    },
})
```

→ [配置详解](./02-configuration.md#示例单向-tls)

#### 生产环境

```go
// mTLS + 重试 + 熔断 + 监控
client, _ := sdk.NewClient(ctx, cfg, 
    sdk.WithUnaryInterceptors(
        observability.MetricsUnaryInterceptor(metrics),
        observability.CircuitBreakerInterceptor(cb),
    ),
)
```

→ [README - 生产环境完整示例](../README.md#生产环境完整示例)

## 📖 文档约定

### 代码示例

所有代码示例都假设已导入：

```go
import (
    "context"
    "log"
    "time"
    
    sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/auth"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/config"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/observability"
)
```

### 配置示例

- ✅ 生产推荐配置
- ⚠️ 需要根据实际情况调整
- ❌ 仅用于测试/开发

### 符号说明

- 📌 重要提示
- 💡 最佳实践
- ⚡ 性能优化
- 🔒 安全建议
- 🐛 常见陷阱

## 🤝 贡献

发现文档问题？欢迎提交 Issue 或 PR：

- 文档源码：`pkg/sdk/docs/`
- 示例代码：`pkg/sdk/_examples/`

## 📝 更新日志

- **2025-12-08**: 初始文档
  - 快速开始
  - 配置详解
  - JWT 验证
  - 服务间认证
  - 错误处理
  - 可观测性

## 📧 联系方式

- GitHub Issues: <https://github.com/FangcunMount/iam-contracts/issues>
- 邮箱: <support@example.com>
