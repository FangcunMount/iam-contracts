# 日志使用快速参考

## 1. HTTP 日志

```go
import "github.com/FangcunMount/component-base/pkg/log"

// INFO 级别
log.HTTP("请求成功",
    log.String("method", "POST"),
    log.String("path", "/api/orders"),
    log.Int("status", 200),
    log.Float64("duration_ms", 45.6),
)

// DEBUG 级别
log.HTTPDebug("请求详情", log.String("body", bodyStr))

// WARN 级别
log.HTTPWarn("响应慢", log.Float64("duration_ms", 1523.4))

// ERROR 级别
log.HTTPError("请求失败",
    log.Int("status", 500),
    log.String("error", err.Error()),
)
```

## 2. SQL 日志

```go
// INFO 级别
log.SQL("查询用户",
    log.String("query", "SELECT * FROM users WHERE id = ?"),
    log.Int("rows", 1),
    log.Float64("duration_ms", 12.3),
)

// DEBUG 级别
log.SQLDebug("SQL 执行计划", log.String("explain", "..."))

// WARN 级别（慢查询）
log.SQLWarn("慢查询",
    log.String("query", "SELECT * FROM orders..."),
    log.Float64("duration_ms", 2500.0),
)

// ERROR 级别
log.SQLError("查询失败",
    log.String("query", "INSERT INTO..."),
    log.String("error", err.Error()),
)
```

## 3. gRPC 日志

```go
// INFO 级别
log.GRPC("调用成功",
    log.String("service", "UserService"),
    log.String("method", "GetUser"),
    log.String("code", "OK"),
    log.Float64("duration_ms", 23.5),
)

// DEBUG 级别
log.GRPCDebug("请求详情", log.String("request", "..."))

// WARN 级别
log.GRPCWarn("响应超时", log.Float64("duration_ms", 4500.0))

// ERROR 级别
log.GRPCError("调用失败",
    log.String("code", "UNAVAILABLE"),
    log.String("error", err.Error()),
)
```

## 4. Redis 日志

```go
// INFO 级别
log.Redis("缓存命中",
    log.String("command", "GET"),
    log.String("key", "user:10086"),
    log.Bool("hit", true),
)

// DEBUG 级别
log.RedisDebug("连接池状态", log.Int("active", 10))

// WARN 级别
log.RedisWarn("缓存未命中", log.String("key", "..."))

// ERROR 级别
log.RedisError("连接失败",
    log.String("host", "redis.example.com:6379"),
    log.String("error", err.Error()),
)
```

## 5. 链路追踪

### 创建追踪上下文

```go
import (
    "context"
    "github.com/FangcunMount/component-base/pkg/log"
    "github.com/FangcunMount/component-base/pkg/util/idutil"
)

// 生成追踪 ID
traceID := idutil.NewTraceID()    // 32 字符
spanID := idutil.NewSpanID()      // 16 字符
requestID := idutil.NewRequestID() // UUID

// 注入到 context
ctx := log.WithTraceContext(context.Background(), traceID, spanID, requestID)
```

### 使用带追踪的日志

```go
// 使用 Context 方法（推荐）
log.InfoContext(ctx, "处理订单",
    log.String("order_id", "ORD-123"),
)

// 或手动添加追踪字段
fields := log.TraceFields(ctx)
log.Info("处理订单",
    append(fields, log.String("order_id", "ORD-123"))...,
)
```

### 创建子 Span

```go
// 创建子操作的 span
childSpanID := idutil.NewSpanID()
childCtx := log.WithSpanID(ctx, childSpanID)

log.InfoContext(childCtx, "调用支付服务")
```

### 获取追踪信息

```go
traceID := log.TraceID(ctx)
spanID := log.SpanID(ctx)
requestID := log.RequestID(ctx)
```

### HTTP 传递追踪信息

```go
// 客户端：设置请求头
req, _ := http.NewRequest("POST", url, body)
req.Header.Set("X-Trace-Id", log.TraceID(ctx))
req.Header.Set("X-Span-Id", idutil.NewSpanID())
req.Header.Set("X-Request-ID", log.RequestID(ctx))

// 服务端：自动从 Tracing 中间件获取
// 已经在 genericapiserver.go 中配置
```

### gRPC 传递追踪信息

```go
import "google.golang.org/grpc/metadata"

// 客户端：设置 metadata
md := metadata.Pairs(
    "x-trace-id", log.TraceID(ctx),
    "x-span-id", idutil.NewSpanID(),
    "x-request-id", log.RequestID(ctx),
)
ctx = metadata.NewOutgoingContext(ctx, md)

// 服务端：从 metadata 获取
md, ok := metadata.FromIncomingContext(ctx)
if ok {
    traceID := md.Get("x-trace-id")[0]
    // ...
}
```

## 6. 在中间件中使用

### HTTP 中间件

```go
func MyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := c.Request.Context()
        
        // 使用类型化日志 + 追踪
        log.HTTP("中间件执行",
            append(log.TraceFields(ctx),
                log.String("middleware", "MyMiddleware"),
            )...,
        )
        
        c.Next()
    }
}
```

### gRPC 拦截器

```go
func MyInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // 使用类型化日志 + 追踪
        log.GRPC("拦截器执行",
            append(log.TraceFields(ctx),
                log.String("method", info.FullMethod),
            )...,
        )
        
        return handler(ctx, req)
    }
}
```

## 7. 在数据库操作中使用

```go
// GORM 已自动配置，只需传递 context
db.WithContext(ctx).Where("id = ?", id).First(&user)

// 日志会自动包含 trace_id、span_id、request_id
```

## 8. 在 Redis 操作中使用

```go
// 使用辅助函数（已在 infra/redis/logging.go 中定义）
redisInfo(ctx, "缓存操作",
    log.String("command", "SET"),
    log.String("key", key),
)

// 或直接使用
log.Redis("缓存操作",
    append(log.TraceFields(ctx),
        log.String("command", "SET"),
        log.String("key", key),
    )...,
)
```

## 9. 常用日志字段

### HTTP 日志
- method: 请求方法
- path: 请求路径
- status: 状态码
- duration_ms: 响应时间（毫秒）
- client_ip: 客户端IP
- user_agent: 用户代理

### SQL 日志
- query: SQL 语句
- rows: 影响行数
- duration_ms: 执行时间（毫秒）
- error: 错误信息

### gRPC 日志
- service: 服务名
- method: 方法名
- code: 状态码
- duration_ms: 调用时间（毫秒）

### Redis 日志
- command: Redis 命令
- key: 键名
- hit: 是否命中（布尔值）
- ttl: 过期时间

## 10. 注意事项

### ✅ 正确用法

```go
// 1. 使用类型化日志
log.HTTP("请求处理", log.String("method", "GET"))

// 2. 传递 context
log.InfoContext(ctx, "处理订单")

// 3. 使用 log.String("error", err.Error())
log.SQLError("查询失败", log.String("error", err.Error()))

// 4. 添加追踪字段
log.Info("操作", append(log.TraceFields(ctx), fields...)...)
```

### ❌ 错误用法

```go
// 1. 不使用类型化日志
log.Info("请求处理", log.String("type", "HTTP"))  // 手动添加 type

// 2. 不传递 context
log.Info("处理订单")  // 缺少追踪信息

// 3. 错误的 error 用法
log.SQLError("查询失败", log.Error(err))  // 编译错误

// 4. 忘记添加追踪字段
log.Info("操作", fields...)  // 缺少追踪信息
```

## 11. 日志级别选择

- **DEBUG**: 调试信息，详细的执行细节
- **INFO**: 正常操作，重要的业务事件
- **WARN**: 警告，可能的问题但不影响运行
- **ERROR**: 错误，需要关注的问题

```go
// DEBUG: 详细信息
log.HTTPDebug("请求头详情", log.Any("headers", headers))

// INFO: 正常操作
log.HTTP("请求成功", log.Int("status", 200))

// WARN: 警告
log.HTTPWarn("响应慢", log.Float64("duration_ms", 1500))

// ERROR: 错误
log.HTTPError("请求失败", log.String("error", err.Error()))
```

## 12. 快速查看日志

```bash
# 只查看 HTTP 日志
grep '"type":"HTTP"' logs/app.log

# 只查看 SQL 日志
grep '"type":"SQL"' logs/app.log

# 查看特定 trace_id 的所有日志
grep '"trace_id":"xxx"' logs/app.log

# 查看错误日志
grep '"level":"ERROR"' logs/app.log

# 查看慢查询
grep '"type":"SQL"' logs/app.log | grep '"duration_ms"' | awk '$NF > 1000'
```

## 13. 中间件架构

### 13.1 中间件分层

项目采用分层的中间件架构，各层职责清晰：

```
基础设施层 → 可观测性层 → 上下文层 → 业务层
```

### 13.2 当前中间件配置

```go
// internal/pkg/server/genericapiserver.go
func (s *GenericAPIServer) InstallMiddlewares() {
    // ===== 1. 基础设施层 =====
    s.Use(middleware.Cors())    // CORS 跨域
    s.Use(middleware.Secure)    // 安全头
    s.Use(middleware.Options)   // OPTIONS 请求处理
    
    // ===== 2. 可观测性层 =====
    s.Use(middleware.Tracing()) // 链路追踪（包含 trace_id, span_id, request_id）
    s.Use(middleware.APILogger()) // API 日志（类型化日志 + 追踪）
    
    // ===== 3. 上下文层 =====
    s.Use(middleware.Context()) // 上下文信息
    
    // ===== 4. 业务层（动态加载） =====
    // JWT 认证、权限验证等
}
```

### 13.3 Tracing 中间件

**功能**: 生成分布式追踪信息

```go
// 自动生成或从请求头获取
- trace_id:   32字符十六进制 (X-Trace-Id)
- span_id:    16字符十六进制
- request_id: UUID 格式 (X-Request-ID)

// 注入到 context，供后续中间件和业务代码使用
ctx := log.WithTraceContext(c.Request.Context(), traceID, spanID, requestID)

// 设置响应头
c.Header("X-Trace-Id", traceID)
c.Header("X-Request-ID", requestID)
```

### 13.4 APILogger 中间件

**功能**: 记录 HTTP 请求/响应日志

```go
// 自动使用类型化日志
log.HTTP("请求开始", ...)
log.HTTPWarn("响应慢", ...)
log.HTTPError("请求失败", ...)

// 自动包含追踪信息
log.TraceFields(ctx) // 包含 trace_id, span_id, request_id

// 自动记录
- 请求方法、路径、查询参数
- 请求头（可配置脱敏）
- 请求体（可配置脱敏和大小限制）
- 响应状态码
- 响应体（可配置）
- 处理耗时
```

### 13.5 已废弃的中间件

以下中间件已被更优方案替代，不建议使用：

| 废弃中间件 | 替代方案 | 原因 |
|-----------|---------|------|
| `RequestID()` | `Tracing()` | Tracing 已包含 request_id 生成 |
| `Logger()` | `APILogger()` | APILogger 使用类型化日志 |
| `EnhancedLogger()` | `APILogger()` | APILogger 自动添加追踪信息 |

**迁移建议**:
```go
// ❌ 旧方式
s.Use(middleware.RequestID())
s.Use(middleware.Logger())

// ✅ 新方式
s.Use(middleware.Tracing())   // 包含 request_id
s.Use(middleware.APILogger()) // 类型化日志
```

### 13.6 中间件执行顺序

中间件按以下顺序执行（从上到下）：

1. **Cors** - 处理跨域请求
2. **Secure** - 添加安全头
3. **Options** - 处理 OPTIONS 预检请求
4. **Tracing** - 生成追踪 ID，注入到 context
5. **APILogger** - 记录请求日志（使用 context 中的追踪信息）
6. **Context** - 设置业务上下文
7. **业务中间件** - JWT 认证、权限验证等

**重要**: Tracing 必须在 APILogger 之前执行，这样日志才能包含追踪信息。

### 13.7 如何添加自定义中间件

```go
// 1. 实现中间件
func MyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := c.Request.Context()
        
        // 前置处理
        log.InfoContext(ctx, "middleware start")
        
        c.Next()
        
        // 后置处理
        log.InfoContext(ctx, "middleware end")
    }
}

// 2. 注册到 defaultMiddlewares()
func defaultMiddlewares() map[string]gin.HandlerFunc {
    return map[string]gin.HandlerFunc{
        "my_middleware": MyMiddleware(),
    }
}

// 3. 在配置文件中启用
server:
  middlewares:
    - "my_middleware"
```

## 14. 性能优化建议

### 14.1 避免过度日志

```go
// ❌ 不要在循环中记录详细日志
for _, item := range items {
    log.Debug("处理项", log.Any("item", item))  // 可能产生大量日志
}

// ✅ 批量记录或只记录摘要
log.Info("批量处理", log.Int("count", len(items)))
```

### 14.2 使用合适的日志级别

```go
// 生产环境建议使用 INFO 级别
log:
  level: "info"  # 生产环境

// 开发环境可以使用 DEBUG
log:
  level: "debug"  # 开发环境
```

### 14.3 限制日志大小

```go
// 配置请求体大小限制
api_logger:
  max_request_body_size: 1024   # 1KB
  max_response_body_size: 2048  # 2KB
```

## 15. 参考资料

- [component-base v0.3.0 文档](https://github.com/FangcunMount/component-base)
- [中间件架构设计](./MIDDLEWARE_ARCHITECTURE.md)
- [日志升级总结](./LOG_UPGRADE_SUMMARY.md)

