# Log Package

一个功能强大的日志包，支持多种日志格式和输出方式。

## 特性

- 支持多种日志格式（JSON、Console）
- 支持多种输出方式（文件、控制台、网络）
- 支持日志级别控制
- 支持结构化日志
- 支持日志轮转
- 支持多种日志库（Zap、Logrus、Klog）

## 快速开始

### 基本使用

```go
package main

import (
    "github.com/FangcunMount/iam-contracts/pkg/log"
)

func main() {
    // 初始化日志
    log.Init(&log.Options{
        Level:      "info",
        Format:     "console",
        OutputPaths: []string{"stdout"},
    })
    defer log.Flush()

    // 使用日志
    log.Info("Hello, World!")
    log.Error("Something went wrong")
}
```

### 配置选项

```go
log.Init(&log.Options{
    Level:      "debug",           // 日志级别
    Format:     "json",            // 日志格式
    OutputPaths: []string{         // 输出路径
        "stdout",
        "/var/log/app.log",
    },
    ErrorOutputPaths: []string{    // 错误输出路径
        "stderr",
        "/var/log/app-error.log",
    },
    MaxSize:    100,               // 单个日志文件最大大小（MB）
    MaxAge:     30,                // 保留旧日志文件的最大天数
    MaxBackups: 10,                // 保留旧日志文件的最大数量
    Compress:   true,              // 是否压缩旧日志文件
})
```

## 日志级别

- `debug`: 调试信息
- `info`: 一般信息
- `warn`: 警告信息
- `error`: 错误信息
- `fatal`: 致命错误（会调用os.Exit(1)）
- `panic`: 恐慌错误（会调用panic）

## 结构化日志

```go
log.Info("User login",
    "user_id", 123,
    "ip", "192.168.1.1",
    "user_agent", "Mozilla/5.0...",
)
```

## 日志轮转

当日志文件达到指定大小时，会自动进行轮转：

```go
log.Init(&log.Options{
    OutputPaths: []string{"/var/log/app.log"},
    MaxSize:    100,    // 100MB
    MaxAge:     30,     // 30天
    MaxBackups: 10,     // 保留10个旧文件
    Compress:   true,   // 压缩旧文件
})
```

## 多种日志库支持

### Zap

```go
import "github.com/FangcunMount/iam-contracts/pkg/log"

log.Init(&log.Options{
    Level:      "info",
    Format:     "json",
    OutputPaths: []string{"stdout"},
})
```

### Logrus

```go
import "github.com/FangcunMount/iam-contracts/pkg/log/logrus"

logger := logrus.New()
logger.SetFormatter(&logrus.JSONFormatter{})
logger.Info("Hello, World!")
```

### Klog

```go
import "github.com/FangcunMount/iam-contracts/pkg/log/klog"

klog.InitFlags(nil)
klog.Info("Hello, World!")
klog.Flush()
```

## 开发工具

### 开发环境日志

```go
log.Init(&log.Options{
    Level:      "debug",
    Format:     "console",
    OutputPaths: []string{"stdout"},
    Development: true,  // 开发模式
})
```

### 测试环境日志

```go
log.Init(&log.Options{
    Level:      "info",
    Format:     "json",
    OutputPaths: []string{"/var/log/test.log"},
    Development: false,
})
```

## 最佳实践

1. **选择合适的日志级别**：不要在生产环境使用debug级别
2. **使用结构化日志**：便于日志分析和搜索
3. **配置日志轮转**：避免日志文件过大
4. **分离错误日志**：将错误日志输出到单独的文件
5. **使用有意义的日志消息**：便于问题排查

## 许可证

MIT License
