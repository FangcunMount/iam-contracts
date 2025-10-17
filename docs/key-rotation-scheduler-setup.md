# 密钥轮换调度器配置指南

## 当前状态 ⚠️

**目前系统仍在使用 Ticker 调度器（旧方案）**

```go
// internal/apiserver/container/assembler/authn.go (第340行)
m.RotationScheduler = scheduler.NewKeyRotationScheduler(  // ❌ Ticker 方式
    m.KeyRotationApp,
    checkInterval,  // 1 小时
    logger,
)
```

## 如何切换到 Cron 调度器 ✅

### 方案一：直接硬编码切换（快速）

修改 `internal/apiserver/container/assembler/authn.go` 文件：

```go
// 找到第 335-344 行，替换为：

func (m *AuthnModule) initializeScheduler() {
	// checkInterval := 1 * time.Hour  // 注释掉 Ticker 配置

	logger := log.New(log.NewOptions())

	// 使用 Cron 调度器（推荐）
	m.RotationScheduler = scheduler.NewKeyRotationCronScheduler(
		m.KeyRotationApp,
		"0 2 * * *",  // 每天凌晨2点检查
		logger,
	)

	// 旧的 Ticker 调度器（已弃用）
	// m.RotationScheduler = scheduler.NewKeyRotationScheduler(
	// 	m.KeyRotationApp,
	// 	checkInterval,
	// 	logger,
	// )
}
```

### 方案二：通过配置文件切换（推荐生产环境）

#### 步骤 1：在配置文件中添加调度器配置

编辑 `configs/apiserver.yaml`，添加以下配置：

```yaml
# 密钥轮换调度器配置
key-rotation:
  scheduler:
    # 调度器类型: ticker 或 cron
    type: cron
    
    # Cron 表达式（仅当 type=cron 时有效）
    # "0 2 * * *" = 每天凌晨2点
    # "0 */6 * * *" = 每6小时
    # "@daily" = 每天午夜
    # "@every 24h" = 每24小时
    cron-spec: "0 2 * * *"
    
    # 检查间隔（仅当 type=ticker 时有效）
    check-interval: 1h
    
  # 轮换策略
  policy:
    rotation-interval: 720h  # 30天
    grace-period: 168h       # 7天
    max-keys: 3
```

#### 步骤 2：创建配置结构体

创建 `internal/apiserver/modules/authn/config/scheduler.go`：

```go
package config

import (
	"time"
)

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	Type          string        `mapstructure:"type"`           // ticker 或 cron
	CronSpec      string        `mapstructure:"cron-spec"`      // Cron 表达式
	CheckInterval time.Duration `mapstructure:"check-interval"` // Ticker 检查间隔
}

// RotationPolicyConfig 轮换策略配置
type RotationPolicyConfig struct {
	RotationInterval time.Duration `mapstructure:"rotation-interval"` // 轮换间隔
	GracePeriod      time.Duration `mapstructure:"grace-period"`      // 宽限期
	MaxKeys          int           `mapstructure:"max-keys"`          // 最大密钥数量
}

// KeyRotationConfig 密钥轮换配置
type KeyRotationConfig struct {
	Scheduler SchedulerConfig      `mapstructure:"scheduler"`
	Policy    RotationPolicyConfig `mapstructure:"policy"`
}

// DefaultKeyRotationConfig 默认配置
func DefaultKeyRotationConfig() KeyRotationConfig {
	return KeyRotationConfig{
		Scheduler: SchedulerConfig{
			Type:          "cron",       // 默认使用 Cron
			CronSpec:      "0 2 * * *",  // 每天凌晨2点
			CheckInterval: time.Hour,    // Ticker 模式下每小时检查
		},
		Policy: RotationPolicyConfig{
			RotationInterval: 30 * 24 * time.Hour, // 30天
			GracePeriod:      7 * 24 * time.Hour,  // 7天
			MaxKeys:          3,                    // 最多3个密钥
		},
	}
}
```

#### 步骤 3：修改 AuthnModule 初始化逻辑

修改 `internal/apiserver/container/assembler/authn.go`：

```go
func (m *AuthnModule) initializeScheduler() {
	logger := log.New(log.NewOptions())

	// 读取配置（如果配置文件中没有，使用默认值）
	schedulerType := viper.GetString("key-rotation.scheduler.type")
	if schedulerType == "" {
		schedulerType = "cron" // 默认使用 Cron
	}

	switch schedulerType {
	case "cron":
		// 使用 Cron 调度器
		cronSpec := viper.GetString("key-rotation.scheduler.cron-spec")
		if cronSpec == "" {
			cronSpec = "0 2 * * *" // 默认每天凌晨2点
		}

		m.RotationScheduler = scheduler.NewKeyRotationCronScheduler(
			m.KeyRotationApp,
			cronSpec,
			logger,
		)
		logger.Infow("Key rotation scheduler initialized",
			"type", "cron",
			"cronSpec", cronSpec,
		)

	case "ticker":
		// 使用 Ticker 调度器（向后兼容）
		checkInterval := viper.GetDuration("key-rotation.scheduler.check-interval")
		if checkInterval == 0 {
			checkInterval = 1 * time.Hour // 默认每小时
		}

		m.RotationScheduler = scheduler.NewKeyRotationScheduler(
			m.KeyRotationApp,
			checkInterval,
			logger,
		)
		logger.Infow("Key rotation scheduler initialized",
			"type", "ticker",
			"checkInterval", checkInterval,
		)

	default:
		logger.Errorw("Unknown scheduler type, falling back to cron",
			"type", schedulerType,
		)
		m.RotationScheduler = scheduler.NewKeyRotationCronScheduler(
			m.KeyRotationApp,
			"0 2 * * *",
			logger,
		)
	}
}
```

## 验证切换是否成功

### 1. 编译检查

```bash
cd /Users/yangshujie/workspace/golang/src/github.com/fangcun-mount/iam-contracts
go build ./internal/apiserver/container/assembler/
```

### 2. 运行 API Server 并查看日志

```bash
# 启动服务
./iam-apiserver --config=configs/apiserver.yaml

# 预期日志输出：
# INFO  Key rotation scheduler initialized  {"type": "cron", "cronSpec": "0 2 * * *"}
# INFO  Key rotation cron scheduler started  {"cronSpec": "0 2 * * *", "nextRun": "2025-10-19 02:00:00"}
```

### 3. 检查调度器类型

可以添加一个健康检查接口：

```go
// internal/apiserver/modules/authn/interface/restful/handler/jwks.go

// GetSchedulerStatus 获取调度器状态
func (h *JWKSHandler) GetSchedulerStatus(c *gin.Context) {
	isRunning := h.authnModule.RotationScheduler.IsRunning()
	
	// 尝试获取下次运行时间（仅 Cron 调度器支持）
	var nextRun string
	if cronScheduler, ok := h.authnModule.RotationScheduler.(interface{ GetNextRunTime() string }); ok {
		nextRun = cronScheduler.GetNextRunTime()
	}

	core.WriteResponse(c, nil, map[string]interface{}{
		"running": isRunning,
		"nextRun": nextRun,
	})
}
```

注册路由：

```go
// internal/apiserver/routers.go
authnGroup.GET("/jwks/rotation/status", jwksHandler.GetSchedulerStatus)
```

测试：

```bash
curl http://localhost:8080/v1/admin/jwks/rotation/status

# 预期响应（Cron 调度器）：
{
  "running": true,
  "nextRun": "2025-10-19 02:00:00"
}

# 预期响应（Ticker 调度器）：
{
  "running": true,
  "nextRun": ""
}
```

## 快速切换命令

### 切换到 Cron（推荐）

```bash
# 编辑文件
vi internal/apiserver/container/assembler/authn.go

# 找到第 340 行，替换为：
# m.RotationScheduler = scheduler.NewKeyRotationCronScheduler(
#     m.KeyRotationApp,
#     "0 2 * * *",
#     logger,
# )

# 编译并重启服务
go build -o iam-apiserver cmd/apiserver/apiserver.go
./iam-apiserver --config=configs/apiserver.yaml
```

### 切换到 Ticker（回退）

```bash
# 恢复原配置
m.RotationScheduler = scheduler.NewKeyRotationScheduler(
    m.KeyRotationApp,
    1 * time.Hour,
    logger,
)
```

## 配置示例

### 生产环境（推荐）

```yaml
key-rotation:
  scheduler:
    type: cron
    cron-spec: "0 2 * * *"  # 每天凌晨2点
  policy:
    rotation-interval: 720h  # 30天
    grace-period: 168h       # 7天
    max-keys: 3
```

**预期行为**：
- 每天凌晨2点检查一次
- 30天轮换一次密钥
- 7天宽限期保证新旧 JWT 共存
- 资源节省 95.8%

### 高安全要求

```yaml
key-rotation:
  scheduler:
    type: cron
    cron-spec: "0 */6 * * *"  # 每6小时
  policy:
    rotation-interval: 168h  # 7天
    grace-period: 24h        # 1天
    max-keys: 3
```

**预期行为**：
- 每6小时检查一次（0点、6点、12点、18点）
- 7天轮换一次密钥
- 1天宽限期
- 资源节省 75%

### 开发/测试环境

```yaml
key-rotation:
  scheduler:
    type: ticker
    check-interval: 5m  # 每5分钟
  policy:
    rotation-interval: 15m  # 15分钟
    grace-period: 5m        # 5分钟
    max-keys: 3
```

**预期行为**：
- 快速验证轮换逻辑
- 仅用于开发测试
- 不推荐生产环境

## 监控和告警

### 关键指标

1. **调度器运行状态**
   ```bash
   curl /v1/admin/jwks/rotation/status
   ```

2. **下次轮换时间**（Cron 独有）
   ```go
   scheduler.GetNextRunTime()  // "2025-10-19 02:00:00"
   ```

3. **轮换历史**
   - 查看日志中的 "Automatic key rotation completed successfully"
   - 记录轮换次数、失败次数

### 日志示例

**Cron 调度器启动**：
```
INFO  Key rotation cron scheduler started  
      {"cronSpec": "0 2 * * *", "nextRun": "2025-10-19 02:00:00"}
```

**Ticker 调度器启动**：
```
INFO  Key rotation scheduler started  
      {"checkInterval": "1h0m0s"}
```

**密钥轮换执行**：
```
INFO  Automatic key rotation completed successfully  
      {"kid": "key-xxx", "algorithm": "RS256", "status": "active"}
```

## 故障排查

### 问题：调度器未启动

**症状**：
```
WARN  Key rotation scheduler is not running
```

**解决**：
```bash
# 检查 StartSchedulers() 是否被调用
grep -r "StartSchedulers" internal/apiserver/
```

### 问题：Cron 表达式错误

**症状**：
```
ERROR  Failed to add cron job  {"error": "Invalid cron spec", "cronSpec": "invalid"}
```

**解决**：
```bash
# 验证 Cron 表达式
# 使用在线工具: https://crontab.guru/
# 或者测试代码：
go run -c '
import "github.com/robfig/cron/v3"
parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
_, err := parser.Parse("0 2 * * *")
println(err)
'
```

### 问题：密钥未按时轮换

**检查步骤**：
1. 确认调度器正在运行
2. 检查密钥年龄是否达到轮换间隔
3. 查看日志是否有错误信息
4. 手动触发测试：
   ```bash
   curl -X POST /v1/admin/jwks/rotation/trigger
   ```

## 总结

| 状态 | 说明 |
|------|------|
| ✅ Cron 调度器已实现 | `KeyRotationCronScheduler` 已创建 |
| ✅ 依赖已安装 | `github.com/robfig/cron/v3` 已添加 |
| ⚠️ 配置未切换 | 仍在使用 Ticker 调度器 |
| 📝 待完成 | 修改 `authn.go` 切换到 Cron |

**推荐操作**：
1. 使用**方案一（直接硬编码切换）**快速验证
2. 验证通过后，实施**方案二（配置文件切换）**用于生产环境
3. 添加监控接口和告警
