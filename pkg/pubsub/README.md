# PubSub Package

一个轻量级的发布-订阅模式实现，支持多种消息队列后端。

## 特性

- 支持多种消息队列后端（Redis、内存）
- 异步消息处理
- 消息持久化
- 错误处理和重试机制
- 优雅关闭
- 监控和指标

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "time"

    "github.com/fangcun-mount/iam-contracts/pkg/pubsub"
)

func main() {
    // 创建发布者
    publisher := pubsub.NewPublisher()
    defer publisher.Close()

    // 创建订阅者
    subscriber := pubsub.NewSubscriber()
    defer subscriber.Close()

    // 订阅主题
    subscriber.Subscribe("user.created", func(msg *pubsub.Message) {
        fmt.Printf("收到消息: %s\n", msg.Data)
    })

    // 发布消息
    publisher.Publish("user.created", "用户已创建")
}
```

### Redis后端

```go
import (
    "github.com/fangcun-mount/iam-contracts/pkg/pubsub"
)

// 创建Redis发布者
publisher := pubsub.NewRedisPublisher(&pubsub.RedisConfig{
    Addr: "localhost:6379",
})

// 创建Redis订阅者
subscriber := pubsub.NewRedisSubscriber(&pubsub.RedisConfig{
    Addr: "localhost:6379",
})
```

### 消息处理

```go
// 定义消息处理器
type UserHandler struct{}

func (h *UserHandler) HandleUserCreated(msg *pubsub.Message) {
    var user User
    json.Unmarshal(msg.Data, &user)
    
    // 处理用户创建逻辑
    fmt.Printf("处理用户创建: %s\n", user.Name)
}

// 注册处理器
subscriber.Subscribe("user.created", handler.HandleUserCreated)
```

### 错误处理

```go
subscriber.Subscribe("user.created", func(msg *pubsub.Message) {
    defer func() {
        if r := recover(); r != nil {
            log.Errorf("消息处理panic: %v", r)
        }
    }()
    
    // 处理消息
    if err := processMessage(msg); err != nil {
        log.Errorf("消息处理失败: %v", err)
        // 可以选择重试或丢弃消息
    }
})
```

### 优雅关闭

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// 优雅关闭
if err := subscriber.Shutdown(ctx); err != nil {
    log.Errorf("关闭订阅者失败: %v", err)
}
```

## 配置选项

### Redis配置

```go
config := &pubsub.RedisConfig{
    Addr:         "localhost:6379",
    Password:     "",
    DB:           0,
    PoolSize:     10,
    MinIdleConns: 5,
    MaxRetries:   3,
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
}
```

### 订阅者配置

```go
config := &pubsub.SubscriberConfig{
    BufferSize:    1000,
    WorkerCount:   10,
    RetryAttempts: 3,
    RetryDelay:    1 * time.Second,
}
```

## 最佳实践

1. **消息幂等性**：确保消息处理是幂等的
2. **错误处理**：妥善处理消息处理过程中的错误
3. **监控**：监控消息队列的状态和性能
4. **优雅关闭**：在应用关闭时优雅地关闭消息队列
5. **消息大小**：控制消息大小，避免过大的消息

## 许可证

MIT License
