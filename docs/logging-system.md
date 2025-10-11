# 日志模块设计

## 🎯 设计理念

日志系统是框架的重要组成部分，基于Zap高性能日志库构建，提供结构化日志记录、多级别日志、日志轮转等功能。系统支持多种输出格式和输出目标，满足不同环境下的日志需求。

## 🏗️ 架构设计

```text
┌─────────────────────────────────────────────────────────────┐
│                    Logging System                           │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Log Interface                            │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │   Debug     │  │    Info     │  │    Error    │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Output Targets                           │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │   Console   │  │    File     │  │   Network   │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Log Formats                              │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │   Console   │  │     JSON    │  │   Custom    │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## 📦 核心组件

### 配置选项

```go
// Options 日志配置选项
type Options struct {
    Level             string   `json:"level" mapstructure:"level"`
    Format            string   `json:"format" mapstructure:"format"`
    EnableColor       bool     `json:"enable-color" mapstructure:"enable-color"`
    DisableCaller     bool     `json:"disable-caller" mapstructure:"disable-caller"`
    DisableStacktrace bool     `json:"disable-stacktrace" mapstructure:"disable-stacktrace"`
    Development       bool     `json:"development" mapstructure:"development"`
    Name              string   `json:"name" mapstructure:"name"`
    OutputPaths       []string `json:"output-paths" mapstructure:"output-paths"`
    ErrorOutputPaths  []string `json:"error-output-paths" mapstructure:"error-output-paths"`
    MaxSize           int      `json:"max-size" mapstructure:"max-size"`
    MaxAge            int      `json:"max-age" mapstructure:"max-age"`
    MaxBackups        int      `json:"max-backups" mapstructure:"max-backups"`
    Compress          bool     `json:"compress" mapstructure:"compress"`
}
```

### 默认配置

```go
// NewOptions 创建默认配置
func NewOptions() *Options {
    return &Options{
        Level:             "info",
        Format:            "console",
        EnableColor:       false,
        DisableCaller:     false,
        DisableStacktrace: false,
        Development:       false,
        Name:              "",
        OutputPaths:       []string{"stdout"},
        ErrorOutputPaths:  []string{"stderr"},
        MaxSize:           100,    // 100MB
        MaxAge:            30,     // 30天
        MaxBackups:        10,     // 10个备份
        Compress:          true,   // 压缩
    }
}
```

## 🔧 使用方法

### 基本初始化

```go
// 使用默认配置
log.Init(nil)
defer log.Flush()

// 使用自定义配置
opts := &log.Options{
    Level:      "debug",
    Format:     "json",
    OutputPaths: []string{"stdout", "/var/log/app.log"},
}
log.Init(opts)
defer log.Flush()
```

### 基本日志记录

```go
// 不同级别的日志
log.Debug("Debug message")
log.Info("Info message")
log.Warn("Warning message")
log.Error("Error message")
log.Fatal("Fatal message")  // 会调用os.Exit(1)
log.Panic("Panic message")  // 会调用panic

// 格式化日志
log.Infof("User %s logged in", username)
log.Errorf("Failed to connect to %s: %v", host, err)

// 结构化日志（推荐）
log.Infow("User login",
    "user_id", 123,
    "ip", "192.168.1.1",
    "user_agent", "Mozilla/5.0...",
)

log.Errorw("Database error",
    "table", "users",
    "operation", "insert",
    "error", err,
)
```

### 字段化日志

```go
// 使用类型化字段（性能更好）
log.Info("Request processed",
    log.String("method", "POST"),
    log.String("path", "/api/users"),
    log.Int("status", 200),
    log.Duration("latency", time.Millisecond*15),
    log.Int64("user_id", 12345),
    log.Bool("cached", true),
    log.Any("headers", headers),
)

// 常用字段类型
log.String("key", "value")          // 字符串
log.Int("count", 10)                // 整数
log.Float64("score", 95.5)          // 浮点数
log.Bool("success", true)           // 布尔值
log.Duration("latency", duration)   // 时间间隔
log.Time("timestamp", time.Now())   // 时间
log.Err(err)                        // 错误
log.Any("data", complexObject)      // 任意类型
```

## 🔄 高级功能

### 上下文日志

```go
// 将logger存入context
ctx := log.WithContext(context.Background())

// 从context获取logger
logger := log.FromContext(ctx)
logger.Info("Operation completed")

// 在HTTP处理器中使用
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    logger := log.FromContext(ctx)
    logger.Info("Handling request", 
        log.String("method", r.Method),
        log.String("path", r.URL.Path),
    )
}
```

### 具名Logger

```go
// 创建具名logger
userLogger := log.WithName("user-service")
userLogger.Info("User service started")

// 创建带默认字段的logger
requestLogger := log.WithValues(
    "request_id", "req-123",
    "user_id", 456,
)
requestLogger.Info("Processing request")
requestLogger.Error("Request failed")
```

### 级别控制

```go
// 检查级别是否启用
if log.V(log.DebugLevel).Enabled() {
    log.V(log.DebugLevel).Info("Expensive debug operation")
}

// 条件日志
if log.V(1).Enabled() {
    log.V(1).Info("Verbose logging enabled")
}
```

## 📊 日志轮转

### 配置轮转

```go
opts := &log.Options{
    OutputPaths: []string{"/var/log/app.log"},
    MaxSize:     100,    // 100MB
    MaxAge:      30,     // 30天
    MaxBackups:  10,     // 保留10个旧文件
    Compress:    true,   // 压缩旧文件
}
log.Init(opts)
```

### 轮转行为

- 当日志文件达到`MaxSize`时，会自动轮转
- 旧文件会被重命名为`app.log.2024-01-01.1.gz`
- 超过`MaxAge`天的文件会被自动删除
- 最多保留`MaxBackups`个旧文件
- 如果启用`Compress`，旧文件会被压缩

## 🎨 输出格式

### Console格式

```go
opts := &log.Options{
    Format:      "console",
    EnableColor: true,
    OutputPaths: []string{"stdout"},
}
```

输出示例：

```text
2024-01-01T12:00:00.000Z    INFO    user-service/main.go:25    User service started
2024-01-01T12:00:01.000Z    INFO    user-service/handler.go:15    User login    {"user_id": 123, "ip": "192.168.1.1"}
```

### JSON格式

```go
opts := &log.Options{
    Format:      "json",
    OutputPaths: []string{"stdout"},
}
```

输出示例：

```json
{
  "level": "info",
  "ts": 1704110400.000,
  "caller": "user-service/main.go:25",
  "msg": "User service started",
  "service": "user-service"
}
{
  "level": "info",
  "ts": 1704110401.000,
  "caller": "user-service/handler.go:15",
  "msg": "User login",
  "user_id": 123,
  "ip": "192.168.1.1"
}
```

## 🔧 中间件集成

### HTTP日志中间件

```go
func LoggingMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        // 创建请求专用logger
        requestLogger := log.WithValues(
            "request_id", generateRequestID(),
            "method", c.Request.Method,
            "path", c.Request.URL.Path,
            "remote_addr", c.ClientIP(),
        )
        
        // 将logger存入context
        ctx := requestLogger.WithContext(c.Request.Context())
        c.Request = c.Request.WithContext(ctx)
        
        requestLogger.Info("Request started")
        
        // 处理请求
        c.Next()
        
        // 记录响应
        requestLogger.Info("Request completed",
            log.Int("status", c.Writer.Status()),
            log.Duration("latency", time.Since(start)),
        )
    }
}
```

### 错误日志中间件

```go
func ErrorLoggingMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        
        // 检查是否有错误
        if len(c.Errors) > 0 {
            logger := log.FromContext(c.Request.Context())
            for _, err := range c.Errors {
                logger.Errorw("Request error",
                    "error", err.Error(),
                    "type", err.Type,
                    "meta", err.Meta,
                )
            }
        }
    }
}
```

## 🧪 测试策略

### 单元测试

```go
func TestLogger(t *testing.T) {
    // 创建测试配置
    opts := &log.Options{
        Level:      "debug",
        Format:     "console",
        OutputPaths: []string{"stdout"},
    }
    log.Init(opts)
    defer log.Flush()
    
    // 测试日志记录
    log.Info("Test message")
    log.Error("Test error")
}
```

### 性能测试

```go
func BenchmarkLogger(b *testing.B) {
    opts := &log.Options{
        Level:      "info",
        Format:     "json",
        OutputPaths: []string{"/dev/null"},
    }
    log.Init(opts)
    defer log.Flush()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        log.Info("Benchmark message",
            log.Int("iteration", i),
            log.String("test", "performance"),
        )
    }
}
```

## 🎯 最佳实践

### 1. 日志级别使用

- **Debug**: 详细的调试信息，仅在开发环境使用
- **Info**: 一般信息，记录重要的业务事件
- **Warn**: 警告信息，不影响系统运行但需要注意
- **Error**: 错误信息，影响功能但系统可以继续运行
- **Fatal**: 致命错误，系统无法继续运行
- **Panic**: 严重错误，会触发panic

### 2. 结构化日志

```go
// 推荐：使用结构化字段
log.Info("User operation", 
    log.String("action", "login"),
    log.Int("user_id", 123),
    log.Duration("latency", time.Millisecond*15),
)

// 不推荐：使用字符串拼接
log.Info("User " + username + " logged in with ID " + strconv.Itoa(userID))
```

### 3. 错误日志

```go
// 推荐：记录错误详情
log.Errorw("Database operation failed",
    "table", "users",
    "operation", "insert",
    "user_id", userID,
    "error", err,
)

// 不推荐：只记录错误消息
log.Error("Database error: " + err.Error())
```

### 4. 性能考虑

```go
// 推荐：条件日志
if log.V(log.DebugLevel).Enabled() {
    log.V(log.DebugLevel).Info("Expensive debug operation",
        log.Any("data", expensiveData),
    )
}

// 不推荐：无条件记录
log.Debug("Expensive debug operation", log.Any("data", expensiveData))
```

### 5. 上下文传递

```go
// 推荐：在context中传递logger
func processRequest(ctx context.Context, data interface{}) error {
    logger := log.FromContext(ctx)
    logger.Info("Processing request", log.Any("data", data))
    // ...
}

// 使用
ctx := log.WithContext(context.Background())
err := processRequest(ctx, data)
```

## 📈 监控和告警

### 日志聚合

- 使用ELK Stack (Elasticsearch, Logstash, Kibana)
- 使用Fluentd进行日志收集
- 使用Prometheus + Grafana进行监控

### 告警规则

```yaml
# Prometheus告警规则示例
groups:
  - name: application_alerts
    rules:
      - alert: HighErrorRate
        expr: rate(log_errors_total[5m]) > 0.1
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors per second"
```

### 日志分析

```sql
-- 分析错误日志
SELECT 
    DATE(timestamp) as date,
    COUNT(*) as error_count,
    error_type
FROM logs 
WHERE level = 'error' 
GROUP BY DATE(timestamp), error_type
ORDER BY date DESC;
```
