# 密钥轮换自动化

## 概述

密钥轮换自动化功能实现了JWT签名密钥的定时自动轮换，确保系统安全性和密钥的定期更新。

## 架构设计

### 1. 领域服务层

#### KeyRotation Service (`domain/jwks/service/key_rotation.go`)

**职责**：

- 执行密钥轮换逻辑
- 判断是否需要轮换
- 管理轮换策略
- 清理过期密钥

**核心方法**：

```go
// RotateKey 执行密钥轮换
// 流程：
// 1. 将当前所有 Active 密钥转为 Grace 状态
// 2. 生成新密钥（Active 状态）
// 3. 清理超过 MaxKeys 的密钥（将最老的 Grace 密钥转为 Retired）
// 4. 清理过期的 Retired 密钥
func (s *KeyRotation) RotateKey(ctx context.Context) (*jwks.Key, error)

// ShouldRotate 判断是否需要轮换
// 根据 RotationPolicy 判断当前 Active 密钥是否已到轮换时间
func (s *KeyRotation) ShouldRotate(ctx context.Context) (bool, error)

// GetRotationPolicy 获取当前轮换策略
func (s *KeyRotation) GetRotationPolicy() jwks.RotationPolicy

// UpdateRotationPolicy 更新轮换策略
func (s *KeyRotation) UpdateRotationPolicy(ctx context.Context, policy jwks.RotationPolicy) error

// GetRotationStatus 获取轮换状态
func (s *KeyRotation) GetRotationStatus(ctx context.Context) (*driving.RotationStatus, error)
```

### 2. 基础设施层

#### KeyRotationScheduler (`infra/scheduler/key_rotation_scheduler.go`)

**职责**：

- 定时检查并执行密钥轮换
- 管理调度器生命周期
- 支持手动触发轮换

**核心方法**：

```go
// Start 启动调度器
func (s *KeyRotationScheduler) Start(ctx context.Context) error

// Stop 停止调度器
func (s *KeyRotationScheduler) Stop() error

// IsRunning 返回调度器是否正在运行
func (s *KeyRotationScheduler) IsRunning() bool

// TriggerNow 立即触发一次密钥轮换检查
func (s *KeyRotationScheduler) TriggerNow(ctx context.Context) error
```

**工作原理**：

- 使用 `time.Ticker` 定期触发检查（默认每小时）
- 每次触发时调用 `ShouldRotate()` 判断是否需要轮换
- 如果需要，自动调用 `RotateKey()` 执行轮换
- 支持优雅关闭（等待正在执行的轮换完成）

### 3. 应用层

#### KeyRotationAppService (`application/jwks/key_rotation.go`)

**职责**：

- 协调领域服务和调度器
- 提供应用级API
- 日志记录和错误处理

### 4. 容器集成

#### AuthnModule (`container/assembler/authn.go`)

**集成点**：

```go
type AuthnModule struct {
    // ... 其他服务 ...
    
    KeyRotationApp *jwksApp.KeyRotationAppService
    
    RotationScheduler interface {
        Start(ctx context.Context) error
        Stop() error
        IsRunning() bool
        TriggerNow(ctx context.Context) error
    }
}

// StartSchedulers 启动调度器
func (m *AuthnModule) StartSchedulers(ctx context.Context) error

// StopSchedulers 停止调度器
func (m *AuthnModule) StopSchedulers() error

// Cleanup 清理模块资源（会自动停止调度器）
func (m *AuthnModule) Cleanup() error
```

## 轮换策略

### RotationPolicy 配置

```go
type RotationPolicy struct {
    RotationInterval time.Duration // 轮换间隔（默认30天）
    GracePeriod      time.Duration // 宽限期（默认7天）
    MaxKeysInJWKS    int           // JWKS 中最多保留密钥数（默认3个）
}
```

### 默认策略

```go
RotationInterval: 30 * 24 * time.Hour,  // 30 天
GracePeriod:      7 * 24 * time.Hour,   // 7 天
MaxKeysInJWKS:    3,                    // 最多 3 个密钥
```

### 密钥生命周期

```
时间线：
─────────────────────────────────────────────────────►

Day 0      创建密钥 K1 (Active)
           └─ 开始签发 JWT
           └─ 发布到 JWKS
           
Day 30     创建密钥 K2 (Active)
           K1 进入 Grace Period
           └─ K1 仅用于验签，不再签发
           └─ K1, K2 都发布到 JWKS
           
Day 37     K1 过期（Grace Period 结束）
           K1 转为 Retired
           └─ K1 从 JWKS 移除
           └─ K1 等待物理删除
           
Day 60     创建密钥 K3 (Active)
           K2 进入 Grace Period
           └─ 如果密钥数超过 MaxKeysInJWKS，
              清理最老的 Grace/Retired 密钥
```

## 使用方式

### 1. 启动服务时

```go
// 初始化模块
authnModule := &assembler.AuthnModule{}
err := authnModule.Initialize(db, redisClient)

// 启动调度器
ctx := context.Background()
err = authnModule.StartSchedulers(ctx)
```

### 2. 关闭服务时

```go
// 清理资源（会自动停止调度器）
err := authnModule.Cleanup()
```

### 3. 手动触发轮换

```go
// 通过应用服务手动触发
resp, err := authnModule.KeyRotationApp.RotateKey(ctx)

// 或通过调度器立即触发
err := authnModule.RotationScheduler.TriggerNow(ctx)
```

### 4. 查询轮换状态

```go
// 检查是否需要轮换
shouldRotate, err := authnModule.KeyRotationApp.ShouldRotate(ctx)

// 获取详细状态
status, err := authnModule.KeyRotationApp.GetRotationStatus(ctx)
fmt.Printf("Active Key: %s\n", status.ActiveKey.Kid)
fmt.Printf("Grace Keys: %d\n", len(status.GraceKeys))
fmt.Printf("Next Rotation: %s\n", status.NextRotation)
```

### 5. 更新轮换策略

```go
newPolicy := jwks.RotationPolicy{
    RotationInterval: 60 * 24 * time.Hour,  // 60天
    GracePeriod:      14 * 24 * time.Hour,  // 14天
    MaxKeysInJWKS:    5,                    // 最多5个密钥
}

err := authnModule.KeyRotationApp.UpdateRotationPolicy(ctx, newPolicy)
```

## 配置

### 环境变量 / 配置文件

```yaml
jwks:
  # 密钥存储目录
  keys_dir: "/var/keys/jwks"
  
  # 轮换策略
  rotation_interval: "720h"  # 30天
  grace_period: "168h"       # 7天
  max_keys_in_jwks: 3
  
  # 调度器配置
  rotation_check_interval: "1h"  # 每小时检查一次
```

## 监控和日志

### 日志示例

```
2025-10-18 00:00:00.000 INFO  jwks/key_rotation.go:45  Starting key rotation
2025-10-18 00:00:00.100 INFO  jwks/key_rotation.go:58  Moved active key to grace period  {"kid": "old-key"}
2025-10-18 00:00:00.200 INFO  jwks/key_rotation.go:75  New key generated and activated  {"kid": "new-key", "algorithm": "RS256"}
2025-10-18 00:00:00.300 INFO  jwks/key_rotation.go:95  Cleaned up expired keys  {"count": 2}
2025-10-18 00:00:00.400 INFO  jwks/key_rotation.go:100 Key rotation completed successfully
```

### 监控指标（建议）

- `jwks_rotation_total` - 轮换次数
- `jwks_rotation_duration_seconds` - 轮换耗时
- `jwks_active_keys_count` - Active 密钥数量
- `jwks_grace_keys_count` - Grace 密钥数量
- `jwks_retired_keys_count` - Retired 密钥数量
- `jwks_last_rotation_timestamp` - 最后一次轮换时间
- `jwks_next_rotation_timestamp` - 下次计划轮换时间

## 安全考虑

1. **密钥存储安全**
   - 私钥文件权限 0600（仅所有者可读写）
   - 使用 PKCS#8 标准格式
   - 支持扩展到 KMS/HSM

2. **轮换策略**
   - RotationInterval 应足够长以避免频繁轮换
   - GracePeriod 应足够长以确保客户端有时间更新 JWKS
   - MaxKeysInJWKS 避免 JWKS 体积过大

3. **优雅降级**
   - 轮换失败不影响现有密钥验证
   - 自动重试机制（通过定时调度）
   - 详细日志记录便于排查问题

## REST API 端点（待实现）

```
# 手动触发轮换
POST /v1/admin/jwks/rotation

# 查询轮换状态
GET /v1/admin/jwks/rotation/status

# 更新轮换策略
PUT /v1/admin/jwks/rotation/policy

# 获取当前策略
GET /v1/admin/jwks/rotation/policy
```

## 测试

### 单元测试

```bash
# 测试密钥轮换服务
go test -v ./internal/apiserver/modules/authn/domain/jwks/service/

# 测试调度器
go test -v ./internal/apiserver/modules/authn/infra/scheduler/
```

### 集成测试

```bash
# E2E测试（包含轮换场景）
go test -v -run TestE2E ./internal/apiserver/modules/authn/
```

## 已知限制

1. ✅ 领域服务和调度器已实现
2. ✅ 容器集成完成
3. ⚠️ 单元测试需要修复（Mock接口问题）
4. ⏳ REST API端点待实现
5. ⏳ 配置文件支持待完善
6. ⏳ 监控指标待添加

## 后续改进

1. **配置化**
   - 从配置文件读取轮换策略
   - 从配置文件读取调度间隔
   - 支持热更新配置

2. **监控**
   - 集成 Prometheus metrics
   - 添加健康检查端点
   - 告警规则配置

3. **REST API**
   - 实现管理端点
   - 添加权限控制
   - API 文档（Swagger）

4. **测试**
   - 修复单元测试
   - 添加更多集成测试场景
   - 性能测试

5. **扩展**
   - 支持多算法轮换（RS256, RS384, RS512, ES256等）
   - 支持密钥版本标记
   - 支持密钥回滚机制

## 相关文档

- [JWKS 架构文档](./authn-architecture.md#密钥轮换机制)
- [私钥存储文档](./private-key-storage.md)
- [错误处理文档](./error-handling.md)
