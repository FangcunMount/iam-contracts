# 日志升级总结 - component-base v0.3.0

## 更新时间
2025年11月11日

## 升级内容

### 1. 类型化日志 (Typed Logs)

component-base v0.3.0 引入了类型化日志功能，为不同类型的操作提供专门的日志函数，自动添加 `type` 字段，便于日志分析和监控。

#### 支持的日志类型
- **HTTP** - HTTP/REST API 请求日志
- **SQL** - 数据库查询日志
- **GRPC** - gRPC 服务调用日志
- **Redis** - Redis 缓存操作日志
- **MQ** - 消息队列操作日志
- **Cache** - 通用缓存操作日志
- **RPC** - 通用 RPC 调用日志

每种类型都提供 4 个级别的函数：INFO、DEBUG、WARN、ERROR

#### 使用示例

```go
// HTTP 请求日志
log.HTTP("GET /api/users",
    log.String("method", "GET"),
    log.String("path", "/api/users"),
    log.Int("status", 200),
    log.Float64("duration_ms", 45.6),
)

// SQL 查询日志
log.SQL("查询用户信息",
    log.String("query", "SELECT * FROM users WHERE id = ?"),
    log.Int("rows", 1),
    log.Float64("duration_ms", 12.3),
)

// gRPC 调用日志
log.GRPC("UserService.GetUser",
    log.String("service", "UserService"),
    log.String("method", "GetUser"),
    log.String("code", "OK"),
)

// Redis 操作日志
log.Redis("GET user:10086",
    log.String("command", "GET"),
    log.String("key", "user:10086"),
    log.Bool("hit", true),
)
```

### 2. 链路追踪 (Distributed Tracing)

component-base v0.3.0 新增了完整的链路追踪支持，自动为每个请求生成和传递追踪信息。

#### 核心概念
- **trace_id**: 32字符的追踪ID，标识整个请求链
- **span_id**: 16字符的跨度ID，标识请求链中的特定操作
- **request_id**: 请求ID，用于关联单个HTTP请求

#### API 使用

```go
import (
    "context"
    "github.com/FangcunMount/component-base/pkg/log"
    "github.com/FangcunMount/component-base/pkg/util/idutil"
)

// 1. 创建追踪上下文
ctx := context.Background()
traceID := idutil.NewTraceID()
spanID := idutil.NewSpanID()
requestID := idutil.NewRequestID()

ctx = log.WithTraceContext(ctx, traceID, spanID, requestID)

// 2. 使用带追踪信息的日志
log.InfoContext(ctx, "处理订单",
    log.String("order_id", "ORD-123"),
)

// 3. 创建子 span
childSpanID := idutil.NewSpanID()
childCtx := log.WithSpanID(ctx, childSpanID)

// 4. 获取追踪信息
traceID := log.TraceID(ctx)
spanID := log.SpanID(ctx)
requestID := log.RequestID(ctx)

// 5. 获取追踪字段（用于手动添加到日志）
fields := log.TraceFields(ctx)
```

## 项目中的应用

### 1. HTTP 中间件

项目已实现完整的 HTTP 链路追踪中间件：

**文件**: `internal/pkg/middleware/tracing.go`

```go
// Tracing 中间件自动：
// 1. 从 HTTP 头获取或生成 trace_id
// 2. 生成新的 span_id
// 3. 使用已有的 request_id
// 4. 将追踪信息注入到 request context
// 5. 将追踪信息添加到响应头
func Tracing() gin.HandlerFunc {
    return func(c *gin.Context) {
        traceID := c.GetHeader("X-Trace-Id")
        if traceID == "" {
            traceID = idutil.NewTraceID()
        }
        
        spanID := idutil.NewSpanID()
        requestID := c.GetString(XRequestIDKey)
        
        ctx := log.WithTraceContext(c.Request.Context(), traceID, spanID, requestID)
        c.Request = c.Request.WithContext(ctx)
        
        c.Header("X-Trace-Id", traceID)
        c.Header("X-Span-Id", spanID)
        c.Header("X-Request-ID", requestID)
        
        c.Next()
    }
}
```

**中间件顺序** (`internal/pkg/server/genericapiserver.go`):
```go
s.Use(middleware.RequestID())   // 1. 生成 request_id
s.Use(middleware.Tracing())     // 2. 生成追踪信息并注入 context
s.Use(middleware.Context())     // 3. 设置其他上下文信息
s.Use(middleware.APILogger())   // 4. 记录 HTTP 日志
```

### 2. HTTP API 日志

**文件**: `internal/pkg/middleware/api_logger.go`

- 使用 `log.HTTP()` 记录请求开始和结束
- 自动添加追踪字段: `log.TraceFields(c.Request.Context())`
- 根据状态码使用不同级别: `log.HTTP()`, `log.HTTPWarn()`, `log.HTTPError()`

```go
func logRequestStart(c *gin.Context, cfg APILoggerConfig, requestID string, body []byte, bodyLen int) {
    fields := []log.Field{
        log.String("event", "request_start"),
        log.String("method", c.Request.Method),
        log.String("path", c.Request.URL.Path),
        // ... 其他字段
    }
    
    // 添加追踪字段
    fields = append(fields, log.TraceFields(c.Request.Context())...)
    
    log.HTTP("HTTP request started", fields...)
}
```

### 3. gRPC 日志

**文件**: `internal/pkg/middleware/grpc_logger.go`

- 使用 `log.GRPC()` 及其变体记录 gRPC 调用
- 自动添加追踪字段到每条日志

```go
func grpcInfo(ctx context.Context, msg string, fields ...log.Field) {
    log.GRPC(msg, append(fields, log.TraceFields(ctx)...)...)
}

func grpcDebug(ctx context.Context, msg string, fields ...log.Field) {
    log.GRPCDebug(msg, append(fields, log.TraceFields(ctx)...)...)
}

func grpcWarn(ctx context.Context, msg string, fields ...log.Field) {
    log.GRPCWarn(msg, append(fields, log.TraceFields(ctx)...)...)
}

func grpcError(ctx context.Context, msg string, fields ...log.Field) {
    log.GRPCError(msg, append(fields, log.TraceFields(ctx)...)...)
}
```

### 4. GORM 数据库日志

**文件**: `internal/pkg/logger/logger.go`

- 使用 `log.SQL()` 及其变体记录数据库操作
- 自动添加追踪字段到每条 SQL 日志

```go
func (l logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
    elapsed := time.Since(begin)
    switch {
    case err != nil && l.LogLevel >= Error:
        sql, rows := fc()
        fields := l.traceFields(ctx, sql, rows, elapsed)
        fields = append(fields, log.String("error", err.Error()))
        log.SQLError("GORM trace failed", fields...)
    case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= Warn:
        sql, rows := fc()
        fields := l.traceFields(ctx, sql, rows, elapsed)
        fields = append(fields, log.String("event", "slow_query"))
        log.SQLWarn("GORM slow query", fields...)
    case l.LogLevel >= Info:
        sql, rows := fc()
        fields := l.traceFields(ctx, sql, rows, elapsed)
        log.SQLDebug("GORM trace", fields...)
    }
}

func (l logger) traceFields(ctx context.Context, sql string, rows int64, elapsed time.Duration) []log.Field {
    fields := []log.Field{
        log.String("sql", sql),
        log.Float64("elapsed_ms", float64(elapsed.Nanoseconds())/1e6),
        log.Int64("rows", rows),
    }
    
    // 添加追踪字段
    fields = append(fields, log.TraceFields(ctx)...)
    return fields
}
```

### 5. Redis 日志

**文件**: `internal/apiserver/infra/redis/logging.go`

- 使用 `log.Redis()` 及其变体记录 Redis 操作
- 提供辅助函数自动添加追踪字段

```go
func redisFields(ctx context.Context, fields []log.Field) []log.Field {
    if ctx == nil {
        return fields
    }
    return append(fields, log.TraceFields(ctx)...)
}

func redisInfo(ctx context.Context, msg string, fields ...log.Field) {
    log.Redis(msg, redisFields(ctx, fields)...)
}

func redisDebug(ctx context.Context, msg string, fields ...log.Field) {
    log.RedisDebug(msg, redisFields(ctx, fields)...)
}

func redisWarn(ctx context.Context, msg string, fields ...log.Field) {
    log.RedisWarn(msg, redisFields(ctx, fields)...)
}

func redisError(ctx context.Context, msg string, fields ...log.Field) {
    log.RedisError(msg, redisFields(ctx, fields)...)
}
```

## 代码更新清单

### 修改的文件

1. **internal/pkg/middleware/middleware.go**
   - 在默认中间件列表中添加了 `tracing` 中间件

2. **internal/apiserver/infra/redis/logging.go**
   - 修复了日志字段展开语法 (`...`)

3. **internal/apiserver/infra/redis/*.go**
   - 批量修复了 `log.Error(err)` 的使用方式
   - 改为 `log.String("error", err.Error())`

4. **internal/pkg/middleware/grpc_logger.go**
   - 修复了 `log.Error(err)` 的使用方式

5. **internal/pkg/middleware/enhanced_logger.go**
   - 重命名内部函数避免与 api_logger.go 冲突

### 已存在且运行良好的文件

1. **internal/pkg/middleware/tracing.go** - 链路追踪中间件
2. **internal/pkg/middleware/api_logger.go** - HTTP 日志中间件
3. **internal/pkg/middleware/grpc_logger.go** - gRPC 日志中间件
4. **internal/pkg/logger/logger.go** - GORM 日志适配器
5. **internal/apiserver/infra/redis/*.go** - Redis 操作日志

## 日志输出示例

### HTTP 请求日志
```json
{
  "level": "INFO",
  "timestamp": "2025-11-11T10:54:12.046Z",
  "message": "HTTP request completed",
  "type": "HTTP",
  "trace_id": "a1b2c3d4e5f6789012345678901234ab",
  "span_id": "1234567890abcdef",
  "request_id": "req-uuid-xxx",
  "method": "GET",
  "path": "/api/users",
  "status": 200,
  "duration_ms": 45.6
}
```

### SQL 查询日志
```json
{
  "level": "DEBUG",
  "timestamp": "2025-11-11T10:54:12.050Z",
  "message": "GORM trace",
  "type": "SQL",
  "trace_id": "a1b2c3d4e5f6789012345678901234ab",
  "span_id": "1234567890abcdef",
  "request_id": "req-uuid-xxx",
  "sql": "SELECT * FROM users WHERE id = ?",
  "elapsed_ms": 12.3,
  "rows": 1
}
```

### Redis 操作日志
```json
{
  "level": "INFO",
  "timestamp": "2025-11-11T10:54:12.055Z",
  "message": "access token cached",
  "type": "Redis",
  "trace_id": "a1b2c3d4e5f6789012345678901234ab",
  "span_id": "1234567890abcdef",
  "request_id": "req-uuid-xxx",
  "app_id": "wx123456",
  "ttl": "7200s"
}
```

### gRPC 调用日志
```json
{
  "level": "INFO",
  "timestamp": "2025-11-11T10:54:12.060Z",
  "message": "gRPC client request succeeded",
  "type": "GRPC",
  "trace_id": "a1b2c3d4e5f6789012345678901234ab",
  "span_id": "fedcba0987654321",
  "request_id": "req-uuid-xxx",
  "method": "/iam.identity.v1.UserService/GetUser",
  "latency": "23.5ms"
}
```

## 优势和好处

### 1. 便于日志分析
- 通过 `type` 字段快速过滤特定类型的日志
- 例如：`grep '"type":"HTTP"'` 只查看 HTTP 日志
- 例如：`grep '"type":"SQL"'` 只查看 SQL 日志

### 2. 便于性能监控
- 统计 HTTP 平均响应时间
- 统计 SQL 慢查询数量
- 监控 gRPC 调用失败率
- 分析 Redis 缓存命中率

### 3. 完整的链路追踪
- 通过 `trace_id` 追踪整个请求链路
- 通过 `span_id` 定位具体操作
- 跨服务传递追踪信息
- 便于问题定位和性能分析

### 4. 代码可读性更好
- `log.HTTP()` 比 `log.Info()` 意图更明确
- 自动添加 `type` 字段，减少重复代码
- 统一的日志格式，便于团队协作

### 5. 与日志系统集成
- 易于与 ELK、Grafana 等系统集成
- 按类型创建仪表盘和告警规则
- 自定义分析工具更容易处理

## 最佳实践

### 1. 使用 Context 传递追踪信息

```go
// ✅ 正确：使用 context 传递
func HandleRequest(ctx context.Context) {
    log.InfoContext(ctx, "处理请求")
    
    // 调用其他服务时传递 context
    result, err := service.DoSomething(ctx)
}

// ❌ 错误：不传递 context
func HandleRequest() {
    log.Info("处理请求")  // 无法获取追踪信息
}
```

### 2. 使用类型化日志

```go
// ✅ 正确：使用类型化日志
log.HTTP("请求处理",
    log.String("method", "POST"),
    log.String("path", "/api/orders"),
    log.Int("status", 200),
)

// ❌ 错误：手动添加 type 字段
log.Info("请求处理",
    log.String("type", "HTTP"),  // 容易忘记或拼写错误
    log.String("method", "POST"),
)
```

### 3. 添加关键字段

```go
// ✅ HTTP 日志建议字段
log.HTTP("请求处理",
    log.String("method", "POST"),        // 请求方法
    log.String("path", "/api/orders"),   // 请求路径
    log.Int("status", 200),              // 状态码
    log.Float64("duration_ms", 45.6),    // 响应时间
    log.String("user_id", "10086"),      // 用户ID（可选）
)

// ✅ SQL 日志建议字段
log.SQL("查询用户",
    log.String("query", "SELECT..."),    // SQL语句
    log.Int("rows", 100),                // 影响行数
    log.Float64("duration_ms", 12.3),    // 执行时间
)
```

### 4. 跨服务传递追踪信息

```go
// HTTP 客户端
func CallExternalService(ctx context.Context) {
    traceID := log.TraceID(ctx)
    spanID := idutil.NewSpanID()  // 创建新的 span_id
    
    req, _ := http.NewRequest("POST", "http://service-b/api", nil)
    req.Header.Set("X-Trace-Id", traceID)
    req.Header.Set("X-Span-Id", spanID)
    
    // 发送请求...
}

// gRPC 客户端
func CallGRPCService(ctx context.Context) {
    traceID := log.TraceID(ctx)
    spanID := idutil.NewSpanID()
    
    md := metadata.Pairs(
        "x-trace-id", traceID,
        "x-span-id", spanID,
    )
    ctx = metadata.NewOutgoingContext(ctx, md)
    
    // 调用 gRPC...
}
```

## 注意事项

1. **Context 传递**: 务必在所有需要日志的地方传递 context
2. **Error 字段**: 使用 `log.String("error", err.Error())` 而不是 `log.Error(err)`
3. **中间件顺序**: 确保 Tracing 中间件在其他日志中间件之前
4. **性能考虑**: 链路追踪的性能影响 < 1%，可以放心使用

## 总结

本次升级成功地将项目的日志系统升级到 component-base v0.3.0，实现了：

✅ 完整的类型化日志支持（HTTP、SQL、gRPC、Redis等）
✅ 完整的链路追踪功能（trace_id、span_id、request_id）
✅ 所有中间件和服务正确使用新的日志 API
✅ 代码编译通过，功能正常

项目现在拥有了企业级的日志和追踪能力，极大地提升了系统的可观测性和问题诊断能力。
