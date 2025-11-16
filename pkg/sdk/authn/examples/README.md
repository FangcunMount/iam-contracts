# AuthN SDK 示例代码

本目录包含 AuthN SDK 的各种使用示例。每个示例都在独立的子目录中，可以单独编译和运行。

## 目录结构

```text
examples/
├── README.md          # 本文件
├── basic/             # 基础用法示例
│   └── main.go
├── gin/               # Gin 框架集成示例
│   └── main.go
└── grpc/              # gRPC 拦截器示例
    └── main.go
```

## 示例列表

### 1. basic/ - 基础用法示例

演示 SDK 的基本功能：

- **示例 1**: 最简单的本地验证
- **示例 2**: 带 Audience 和 Issuer 验证
- **示例 3**: 自定义 JWKS 缓存配置
- **示例 4**: 本地 + 远程验证
- **示例 5**: 强制远程验证
- **示例 6**: 访问自定义属性
- **示例 7**: 错误处理
- **示例 8**: 并发验证

编译和运行：

```bash
cd basic
go build
./basic
```

或者直接运行：

```bash
cd basic
go run main.go
```

### 2. gin/ - Gin 框架中间件

演示如何在 Gin Web 框架中集成 SDK：

- JWT 认证中间件
- 可选认证中间件
- 租户验证中间件
- 从上下文获取用户信息
- 路由保护

编译和运行：

```bash
cd gin
go build
./gin
```

或者：

```bash
cd gin
go run main.go
```

测试端点：

```bash
# 公开端点
curl http://localhost:8080/public

# 受保护端点（需要 token）
curl -H "Authorization: Bearer <your-token>" http://localhost:8080/api/protected

# 用户信息
curl -H "Authorization: Bearer <your-token>" http://localhost:8080/api/user/info
```

### 3. grpc/ - gRPC 拦截器

演示如何在 gRPC 服务中集成 SDK：

- 一元 RPC 认证拦截器
- 流式 RPC 认证拦截器
- 租户验证拦截器
- 日志拦截器
- 拦截器链配置
- 公开方法白名单

编译和运行：

```bash
cd grpc
go build
./grpc
```

或者：

```bash
cd grpc
go run main.go
```

测试调用：

```bash
grpcurl -H 'authorization: Bearer <your-token>' \
  localhost:50051 \
  api.Service/Method
```

## 前置要求

### 依赖包

```bash
# Gin 框架（用于 HTTP 示例）
go get -u github.com/gin-gonic/gin

# gRPC（用于 gRPC 示例）
go get -u google.golang.org/grpc
```

### IAM 服务

这些示例需要运行中的 IAM 服务：

1. **JWKS 端点**：`https://iam.example.com/.well-known/jwks.json`
2. **gRPC 端点**（可选）：`iam.example.com:8081`

### 获取 Token

使用 IAM 服务获取有效的 JWT token：

```bash
# 使用 IAM CLI 或 API 登录获取 token
iam-cli login --username user@example.com --password xxx
```

## 配置说明

修改示例代码中的配置以匹配你的环境：

```go
cfg := authnsdk.Config{
    // 修改为你的 JWKS URL
    JWKSURL: "https://your-iam-domain/.well-known/jwks.json",
    
    // 修改为你的 audience 和 issuer
    AllowedAudience: []string{"your-app"},
    AllowedIssuer:   "https://your-iam-domain",
    
    // 可选：gRPC 端点（用于远程验证）
    GRPCEndpoint: "your-iam-domain:8081",
}
```

## 集成到你的项目

### 步骤 1: 安装 SDK

```bash
go get github.com/FangcunMount/iam-contracts/pkg/sdk/authn
```

### 步骤 2: 初始化验证器

```go
import authnsdk "github.com/FangcunMount/iam-contracts/pkg/sdk/authn"

func initVerifier() (*authnsdk.Verifier, error) {
    cfg := authnsdk.Config{
        JWKSURL:         "https://iam.example.com/.well-known/jwks.json",
        AllowedAudience: []string{"my-app"},
        AllowedIssuer:   "https://iam.example.com",
    }
    
    return authnsdk.NewVerifier(cfg, nil)
}
```

### 步骤 3: 集成到你的框架

根据你使用的框架选择相应的示例：

- **Gin / Echo / Fiber**: 参考 `gin_middleware.go`
- **gRPC**: 参考 `grpc_interceptor.go`
- **其他框架**: 参考 `basic_usage.go` 了解核心 API

## 常见问题

### Q: 如何在本地开发时跳过认证？

A: 使用环境变量或配置文件控制：

```go
if os.Getenv("SKIP_AUTH") == "true" {
    // 跳过认证中间件
    return
}
```

### Q: 如何处理 Token 刷新？

A: SDK 只负责验证 Token，刷新逻辑应该在客户端实现。可以通过检查 `exp` 声明提前刷新。

### Q: 性能如何优化？

A:

1. 使用本地验证（不配置 gRPC client）
2. 调整 JWKS 缓存间隔
3. 使用连接池
4. 考虑使用缓存层

### Q: 如何调试验证失败？

A: 启用 Debug 日志级别查看详细信息：

```go
// 在应用配置中设置
log:
  level: debug
```

## 更多资源

- [SDK 文档](../README.md)
- [日志说明](../LOGGING.md)
- [IAM 系统文档](../../../../docs/)

## 反馈

如有问题或建议，请提交 Issue 或 Pull Request。
