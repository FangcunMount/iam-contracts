# 密钥轮换调度器：Ticker vs Cron 对比

## 📊 当前实现：Ticker 轮询方式

### 架构

```go
// KeyRotationScheduler 使用 time.Ticker
type KeyRotationScheduler struct {
    checkInterval time.Duration  // 默认 1 小时
}

func (s *KeyRotationScheduler) run() {
    ticker := time.NewTicker(s.checkInterval)
    // 每 1 小时触发一次
    for {
        case <-ticker.C:
            s.checkAndRotate()  // 调用 ShouldRotate() 检查是否需要轮换
    }
}
```

### 工作流程

```
启动 ──> 每1小时触发 ──> ShouldRotate() ──> 检查密钥年龄是否 >= 30天
                           │
                           ├─ 是 ──> RotateKey() ──> 执行轮换
                           │
                           └─ 否 ──> 跳过（29天23小时的检查都会跳过）
```

### 优缺点

**✅ 优点**：
1. **实现简单** - 只需要标准库 `time.Ticker`
2. **容错性好** - 轮询机制，即使错过某次检查，下次仍会触发
3. **无外部依赖** - 不需要额外的包

**❌ 缺点**：
1. **资源浪费** - 每小时都要执行检查逻辑（29天23小时的检查都是无效的）
   ```
   30天 × 24小时 = 720次检查
   有效检查：1次
   无效检查：719次（99.86%）
   ```

2. **不够精确** - 可能在密钥到期后最多 1 小时才轮换
   ```
   密钥到期时间：2025-10-18 14:35:00
   检查时间：    14:00, 15:00, 16:00...
   实际轮换：    15:00（延迟25分钟）
   ```

3. **不适合复杂调度** - 无法实现"每月1号凌晨2点"这样的需求

4. **内存占用** - Ticker 持续在后台运行

---

## 🚀 推荐方案：Cron 调度方式

### 架构

```go
// KeyRotationCronScheduler 使用 robfig/cron/v3
type KeyRotationCronScheduler struct {
    cronSpec string         // Cron 表达式，如 "0 2 * * *"
    cron     *cron.Cron     // Cron 调度器
}

func (s *KeyRotationCronScheduler) Start(ctx context.Context) error {
    s.cron = cron.New()
    s.cron.AddFunc(s.cronSpec, func() {
        s.checkAndRotate()
    })
    s.cron.Start()
}
```

### 工作流程

```
启动 ──> 注册 Cron 任务 ──> 等待触发时间
                              │
                              v
                         每天凌晨2点 ──> ShouldRotate() ──> 检查并轮换
```

### Cron 表达式示例

```go
// 推荐配置：每天凌晨2点检查
cronSpec: "0 2 * * *"
// ┌─────────── 分钟 (0-59)
// │ ┌────────── 小时 (0-23)
// │ │ ┌───────── 日 (1-31)
// │ │ │ ┌──────── 月 (1-12)
// │ │ │ │ ┌─────── 星期 (0-6, 0=周日)
// │ │ │ │ │
// * * * * *

// 其他示例：
"0 2 * * *"      // 每天凌晨2点
"0 2 */3 * *"    // 每3天凌晨2点
"0 2 1 * *"      // 每月1号凌晨2点
"@daily"         // 每天午夜（00:00）
"@weekly"        // 每周日午夜
"@monthly"       // 每月1号午夜
"@every 24h"     // 每24小时（推荐用于密钥轮换）
"@every 720h"    // 每30天（30×24小时）
```

### 优缺点

**✅ 优点**：
1. **资源节省** - 只在需要时触发，不做无效检查
   ```
   30天期间的检查次数：30次（每天1次）
   vs Ticker：720次（每小时1次）
   资源节省：95.8%
   ```

2. **精确调度** - 可以在指定时间点触发（如凌晨2点低峰期）
   ```
   指定时间：每天凌晨2点
   实际触发：精确到秒
   ```

3. **灵活配置** - 支持复杂的调度需求
   ```go
   // 业务低峰期轮换
   "0 2 * * *"  // 凌晨2点
   
   // 每月定期维护
   "0 2 1 * *"  // 每月1号凌晨2点
   
   // 适应不同安全策略
   "@every 168h"  // 每周轮换一次
   "@every 720h"  // 每月轮换一次
   ```

4. **生产级稳定性** - `robfig/cron` 是成熟的第三方库，被广泛使用

5. **更好的可观测性** - 可以查询下次执行时间
   ```go
   scheduler.GetNextRunTime()  // "2025-10-19 02:00:00"
   ```

**❌ 缺点**：
1. **外部依赖** - 需要引入 `github.com/robfig/cron/v3`
2. **学习成本** - 需要理解 Cron 表达式语法

---

## 🔄 性能对比

### 资源消耗对比（30天周期）

| 指标 | Ticker 方式 | Cron 方式 | 改善 |
|------|------------|----------|------|
| 检查次数 | 720次 | 30次 | -95.8% |
| 无效检查 | 719次 | 29次 | -95.9% |
| CPU 唤醒 | 720次 | 30次 | -95.8% |
| 日志量 | 720条 | 30条 | -95.8% |
| 内存占用 | 持续占用 | 按需占用 | 更低 |

### 时间精度对比

| 场景 | Ticker 方式 | Cron 方式 |
|------|------------|----------|
| 密钥到期时间 | 2025-10-18 14:35:00 | 2025-10-18 14:35:00 |
| 检查间隔 | 每1小时 | 每天凌晨2点 |
| 实际轮换时间 | 15:00（延迟25分钟） | 次日02:00（延迟11小时25分钟） |
| 说明 | 更及时，但浪费资源 | 延迟可控，资源节省 |

**注意**：对于密钥轮换场景，延迟11小时是可接受的，因为：
1. 密钥有 7 天宽限期，旧密钥仍可验证 JWT
2. 在业务低峰期（凌晨）轮换更安全
3. 节省 95.8% 的资源消耗

---

## 💡 推荐配置

### 场景 1：标准生产环境（推荐）

```go
// 使用 Cron 调度器，每天凌晨2点检查
scheduler := NewKeyRotationCronScheduler(
    rotationApp,
    "0 2 * * *",  // 每天凌晨2点
    logger,
)

// 密钥轮换策略
policy := jwks.RotationPolicy{
    RotationInterval: 30 * 24 * time.Hour,  // 30天轮换
    GracePeriod:      7 * 24 * time.Hour,   // 7天宽限期
    MaxKeysInJWKS:    3,                     // 最多3个密钥
}
```

**优势**：
- ✅ 资源消耗最低（每天1次检查）
- ✅ 在业务低峰期执行，影响最小
- ✅ 7天宽限期足够应对延迟

### 场景 2：高频轮换（高安全要求）

```go
// 使用 Cron 调度器，每6小时检查一次
scheduler := NewKeyRotationCronScheduler(
    rotationApp,
    "0 */6 * * *",  // 每6小时（0点、6点、12点、18点）
    logger,
)

// 密钥轮换策略
policy := jwks.RotationPolicy{
    RotationInterval: 7 * 24 * time.Hour,   // 7天轮换
    GracePeriod:      1 * 24 * time.Hour,   // 1天宽限期
    MaxKeysInJWKS:    3,
}
```

**优势**：
- ✅ 轮换更及时（最多延迟6小时）
- ✅ 适合高安全要求场景
- ✅ 仍比 Ticker 节省 75% 资源

### 场景 3：开发/测试环境

```go
// 使用 Ticker 调度器，快速测试
scheduler := NewKeyRotationScheduler(
    rotationApp,
    5 * time.Minute,  // 每5分钟检查
    logger,
)

// 密钥轮换策略
policy := jwks.RotationPolicy{
    RotationInterval: 15 * time.Minute,  // 15分钟轮换
    GracePeriod:      5 * time.Minute,   // 5分钟宽限期
    MaxKeysInJWKS:    3,
}
```

**优势**：
- ✅ 快速验证轮换逻辑
- ✅ 实现简单，无需 Cron 语法
- ❌ 不推荐用于生产环境

---

## 🛠️ 使用示例

### 方案 A：使用 Cron 调度器（推荐）

```go
package main

import (
    "context"
    "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/scheduler"
    "github.com/fangcun-mount/iam-contracts/pkg/log"
)

func main() {
    // 创建应用服务
    rotationApp := jwks.NewKeyRotationAppService(keyRotationSvc, logger)
    
    // 创建 Cron 调度器
    cronScheduler := scheduler.NewKeyRotationCronScheduler(
        rotationApp,
        "0 2 * * *",  // 每天凌晨2点
        logger,
    )
    
    // 启动调度器
    ctx := context.Background()
    if err := cronScheduler.Start(ctx); err != nil {
        log.Fatalf("Failed to start cron scheduler: %v", err)
    }
    
    log.Infow("Cron scheduler started",
        "nextRun", cronScheduler.GetNextRunTime(),  // "2025-10-19 02:00:00"
    )
    
    // 手动触发（如果需要）
    if err := cronScheduler.TriggerNow(ctx); err != nil {
        log.Errorf("Manual trigger failed: %v", err)
    }
    
    // 优雅关闭
    defer cronScheduler.Stop()
}
```

### 方案 B：使用 Ticker 调度器

```go
package main

import (
    "context"
    "time"
    "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/scheduler"
)

func main() {
    // 创建应用服务
    rotationApp := jwks.NewKeyRotationAppService(keyRotationSvc, logger)
    
    // 创建 Ticker 调度器
    tickerScheduler := scheduler.NewKeyRotationScheduler(
        rotationApp,
        1 * time.Hour,  // 每小时检查
        logger,
    )
    
    // 启动调度器
    ctx := context.Background()
    if err := tickerScheduler.Start(ctx); err != nil {
        log.Fatalf("Failed to start ticker scheduler: %v", err)
    }
    
    log.Info("Ticker scheduler started")
    
    // 优雅关闭
    defer tickerScheduler.Stop()
}
```

---

## 📈 迁移建议

### 第一阶段：保留 Ticker，添加 Cron（当前状态）

```go
// 两种调度器并存，可配置选择
type SchedulerType string

const (
    SchedulerTypeTicker SchedulerType = "ticker"
    SchedulerTypeCron   SchedulerType = "cron"
)

// 在配置文件中选择
config:
  scheduler:
    type: "cron"            # 或 "ticker"
    cronSpec: "0 2 * * *"   # 仅 type=cron 时有效
    checkInterval: "1h"     # 仅 type=ticker 时有效
```

### 第二阶段：逐步切换到 Cron

1. **灰度测试**（1-2周）
   - 在测试环境使用 Cron 调度器
   - 验证稳定性和资源消耗

2. **生产环境切换**（1周）
   - 在生产环境启用 Cron 调度器
   - 监控运行状态和资源指标

3. **清理旧代码**（可选）
   - 如果 Cron 调度器运行稳定，可考虑移除 Ticker 调度器
   - 简化代码维护

---

## 🎯 总结

| 维度 | Ticker 方式 | Cron 方式 | 推荐 |
|------|------------|----------|------|
| **资源消耗** | 高（每小时720次） | 低（每天30次） | ✅ Cron |
| **时间精度** | 高（1小时内） | 中（24小时内） | Ticker |
| **实现复杂度** | 简单（标准库） | 中等（第三方库） | Ticker |
| **灵活性** | 低（固定间隔） | 高（复杂表达式） | ✅ Cron |
| **生产适用性** | 低（资源浪费） | 高（稳定高效） | ✅ Cron |
| **可维护性** | 中（简单但浪费） | 高（清晰的调度逻辑） | ✅ Cron |

**最终推荐**：
- ✅ **生产环境使用 Cron 调度器** - 每天凌晨2点检查，资源节省95.8%
- ✅ **保留 Ticker 调度器作为备选** - 用于开发测试或特殊场景
- ✅ **通过配置文件切换** - 灵活适应不同需求

**关键设计原则**：
- 轮换周期（30天）是业务策略
- 检查频率（每天1次）是技术优化
- 宽限期（7天）确保服务连续性
- 在低峰期（凌晨2点）执行确保稳定性
