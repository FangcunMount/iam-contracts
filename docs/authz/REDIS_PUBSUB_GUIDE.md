# Redis 发布订阅机制说明文档

**版本**: v1.0  
**日期**: 2025年10月18日

---

## 📖 概述

在 authz 授权模块中，Redis 发布订阅（Pub/Sub）机制用于 **分布式缓存失效通知**，确保多个服务实例之间的 Casbin 缓存一致性。

---

## 🎯 核心问题：为什么需要发布订阅？

### 问题场景

假设你有 3 个服务实例同时运行：

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  Service A  │    │  Service B  │    │  Service C  │
│             │    │             │    │             │
│  Casbin     │    │  Casbin     │    │  Casbin     │
│  Cache ✓    │    │  Cache ✓    │    │  Cache ✓    │
└─────────────┘    └─────────────┘    └─────────────┘
       ↓                  ↓                  ↓
       └──────────────────┴──────────────────┘
                          ↓
                   ┌─────────────┐
                   │   MySQL     │
                   │  策略数据    │
                   └─────────────┘
```

**问题**：
1. 管理员在 Service A 上修改了策略（添加/删除角色、修改权限）
2. Service A 的 Casbin 缓存被刷新 ✓
3. **Service B 和 C 的缓存仍然是旧的** ❌
4. 用户请求到 Service B/C 时，仍然使用旧策略判定权限 ❌

### 解决方案：策略版本 + Redis Pub/Sub

```
                    策略变更流程
                    
┌──────────────────────────────────────────────────┐
│ 1. 管理员修改策略（通过 PAP）                     │
└────────────┬─────────────────────────────────────┘
             ↓
┌──────────────────────────────────────────────────┐
│ 2. Service A:                                    │
│    - 更新 MySQL 策略表                           │
│    - 递增版本号（v1 → v2）                       │
│    - 刷新本地 Casbin 缓存                        │
└────────────┬─────────────────────────────────────┘
             ↓
┌──────────────────────────────────────────────────┐
│ 3. Service A 发布版本变更通知到 Redis            │
│    PUBLISH authz:policy_changed                  │
│    {"tenant_id": "tenant1", "version": 2}        │
└────────────┬─────────────────────────────────────┘
             ↓
┌──────────────────────────────────────────────────┐
│ 4. Service B, C 收到通知：                       │
│    - 检测到版本号变化（v1 → v2）                 │
│    - 主动刷新本地 Casbin 缓存                    │
│    - 从 MySQL 重新加载最新策略                   │
└──────────────────────────────────────────────────┘
```

---

## 🏗️ 架构设计

### 组件角色

```
┌─────────────────────────────────────────────────────┐
│                    PAP (策略管理)                    │
│  - REST API: POST /authz/roles, /authz/policies   │
│  - 负责修改策略数据                                 │
│  - 负责发布版本变更通知                             │
└────────────────┬────────────────────────────────────┘
                 │ ① 发布通知
                 ↓
         ┌──────────────┐
         │    Redis     │
         │ Pub/Sub 频道 │
         │authz:policy_ │
         │   changed    │
         └──────┬───────┘
                │ ② 广播通知
      ┌─────────┼─────────┐
      ↓         ↓         ↓
┌──────────┐ ┌──────────┐ ┌──────────┐
│Service A │ │Service B │ │Service C │
│  (PDP)   │ │  (PDP)   │ │  (PDP)   │
│          │ │          │ │          │
│ ③ 刷新   │ │ ③ 刷新   │ │ ③ 刷新   │
│  缓存    │ │  缓存    │ │  缓存    │
└──────────┘ └──────────┘ └──────────┘
```

---

## 💻 代码实现

### 1. 发布者（PAP 服务）

**场景**：管理员修改策略后

```go
// application/policy/service.go
type PolicyService struct {
    policyRepo     PolicyVersionRepo
    casbinAdapter  CasbinPort
    versionNotifier VersionNotifier  // Redis 发布者
}

// AddPolicy 添加策略规则
func (s *PolicyService) AddPolicy(ctx context.Context, 
    tenantID string, rule PolicyRule, changedBy string) error {
    
    // 1. 添加策略到 Casbin
    if err := s.casbinAdapter.AddPolicy(ctx, rule); err != nil {
        return err
    }
    
    // 2. 递增版本号并保存到数据库
    newVersion, err := s.policyRepo.Increment(ctx, tenantID, changedBy, "添加策略规则")
    if err != nil {
        return err
    }
    
    // 3. 发布版本变更通知到 Redis
    if err := s.versionNotifier.Publish(ctx, tenantID, newVersion.Version); err != nil {
        log.Errorf("Failed to publish version change: %v", err)
        // 注意：发布失败不阻塞主流程，只记录日志
    }
    
    return nil
}
```

**关键点**：
- ✅ 先更新策略，再发布通知
- ✅ 发布失败不影响主流程（其他实例会通过轮询或下次请求时发现版本变化）

---

### 2. 订阅者（业务服务 - PDP）

**场景**：业务服务启动时订阅版本变更

```go
// 业务服务启动代码
// cmd/apiserver/main.go 或类似的启动文件

func main() {
    // 初始化组件
    db := initDB()
    redisClient := initRedis()
    
    // 创建 Casbin Enforcer
    casbinAdapter := casbin.NewCasbinAdapter(...)
    
    // 创建版本通知器
    versionNotifier := redis.NewVersionNotifier(redisClient, "authz:policy_changed")
    
    // 订阅版本变更通知
    ctx := context.Background()
    err := versionNotifier.Subscribe(ctx, func(tenantID string, version int64) {
        log.Infof("收到策略版本变更通知: tenant=%s, version=%d", tenantID, version)
        
        // 刷新该租户的 Casbin 缓存
        if err := casbinAdapter.InvalidateCache(ctx); err != nil {
            log.Errorf("刷新缓存失败: %v", err)
        } else {
            log.Infof("成功刷新租户 %s 的策略缓存", tenantID)
        }
    })
    
    if err != nil {
        log.Fatalf("订阅策略版本变更失败: %v", err)
    }
    
    // 启动 HTTP 服务
    startHTTPServer()
    
    // 优雅关闭
    defer versionNotifier.Close()
}
```

---

### 3. 完整流程示例

#### 场景：管理员给用户 Alice 授予 "scale-editor" 角色

```go
// ============ PAP 服务（管理端）============

// REST API Handler
func (h *AssignmentHandler) GrantRole(c *gin.Context) {
    var req struct {
        UserID   string `json:"user_id"`   // "alice"
        RoleID   uint64 `json:"role_id"`   // 角色ID
        TenantID string `json:"tenant_id"` // "tenant1"
    }
    c.BindJSON(&req)
    
    // 1. 创建赋权
    assignment := assignment.NewAssignment(
        assignment.SubjectTypeUser,
        req.UserID,
        req.RoleID,
        req.TenantID,
    )
    
    // 2. 保存到数据库
    if err := assignmentRepo.Create(c.Request.Context(), &assignment); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // 3. 添加到 Casbin（g 规则）
    groupingRule := GroupingRule{
        Subject: "user:alice",
        Role:    "role:10",
        Domain:  "tenant1",
    }
    if err := casbinAdapter.AddGroupingPolicy(c.Request.Context(), groupingRule); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // 4. 递增版本号
    newVersion, err := policyVersionRepo.Increment(
        c.Request.Context(),
        req.TenantID,
        "admin",
        "授予用户 alice scale-editor 角色",
    )
    if err != nil {
        log.Errorf("递增版本失败: %v", err)
    }
    
    // 5. 发布版本变更通知 ← 关键步骤
    if err := versionNotifier.Publish(c.Request.Context(), req.TenantID, newVersion.Version); err != nil {
        log.Errorf("发布版本变更失败: %v", err)
        // 不阻塞主流程
    }
    
    c.JSON(200, gin.H{"message": "授权成功"})
}


// ============ PDP 服务（业务端）============

// 订阅处理函数（在服务启动时注册）
func handleVersionChange(tenantID string, version int64) {
    log.Infof("[版本变更] tenant=%s, version=%d", tenantID, version)
    
    // 方案 1: 直接清空 Casbin 缓存（简单）
    casbinAdapter.InvalidateCache(context.Background())
    
    // 方案 2: 重新加载策略（更彻底，但开销大）
    // casbinAdapter.LoadPolicy(context.Background())
    
    log.Infof("[缓存刷新] 租户 %s 的策略缓存已刷新", tenantID)
}

// 业务接口：权限检查
func (h *BusinessHandler) GetScaleForm(c *gin.Context) {
    userID := c.GetString("user_id")   // "alice"
    tenantID := c.GetString("tenant_id") // "tenant1"
    formID := c.Param("id")
    
    // 权限检查：Alice 是否有权限读取量表？
    allowed, err := casbinAdapter.Enforce(
        c.Request.Context(),
        "user:alice",          // 主体
        "tenant1",             // 域
        "scale:form:"+formID,  // 资源
        "read_all",            // 动作
    )
    
    if err != nil || !allowed {
        c.JSON(403, gin.H{"error": "无权访问"})
        return
    }
    
    // ✅ Alice 刚被授权，且缓存已刷新，判定通过
    c.JSON(200, gin.H{"data": "量表数据..."})
}
```

---

## ❓ 常见问题

### Q1: 业务服务必须订阅吗？

**答案：强烈建议订阅，但不是强制的。**

**不订阅的后果**：
- ❌ 策略变更后，需要等待很长时间（或重启服务）才能生效
- ❌ 用户体验差（刚授权的用户仍然显示无权限）
- ❌ 安全风险（刚撤销的权限仍然有效）

**替代方案**（不推荐）：
1. **定时轮询版本号**：每隔 30 秒检查一次版本号，发现变化则刷新缓存
2. **不使用缓存**：每次都从数据库加载策略（性能极差）
3. **手动重启服务**：运维成本高

---

### Q2: 订阅处理函数应该做什么？

**推荐做法：只刷新缓存**

```go
func handleVersionChange(tenantID string, version int64) {
    // ✅ 推荐：清空 Casbin 缓存
    casbinAdapter.InvalidateCache(context.Background())
    
    // ❌ 不推荐：重新加载所有策略（开销大）
    // casbinAdapter.LoadPolicy(context.Background())
    
    // ✅ 可选：记录日志
    log.Infof("策略版本变更: tenant=%s, version=%d", tenantID, version)
    
    // ✅ 可选：发送监控指标
    metrics.PolicyVersionChanged.Inc()
}
```

**为什么只清空缓存？**
- Casbin 的 `CachedEnforcer` 会在下次 `Enforce()` 调用时自动从数据库加载策略
- 避免在收到通知时同步加载策略（可能造成阻塞）

---

### Q3: 如果 Redis 挂了怎么办？

**影响**：
- ❌ 版本变更通知无法发送
- ✅ 策略修改仍然成功（保存到 MySQL）
- ⚠️ 其他服务实例的缓存不会立即刷新

**兜底方案**：

#### 方案 1: 版本号检查（推荐）

```go
// 在每次 Enforce 之前检查版本号
func (a *CasbinAdapter) Enforce(ctx context.Context, sub, dom, obj, act string) (bool, error) {
    // 1. 获取当前版本号
    currentVersion := a.getCurrentVersion(dom)
    
    // 2. 从数据库查询最新版本号
    latestVersion, _ := policyVersionRepo.GetVersionNumber(ctx, dom)
    
    // 3. 如果版本不一致，刷新缓存
    if currentVersion != latestVersion {
        log.Warnf("检测到版本不一致，刷新缓存: %d -> %d", currentVersion, latestVersion)
        a.InvalidateCache(ctx)
        a.setCurrentVersion(dom, latestVersion)
    }
    
    // 4. 执行权限判定
    return a.enforcer.Enforce(sub, dom, obj, act)
}
```

#### 方案 2: 定时轮询版本号

```go
// 启动定时任务，每 30 秒检查一次版本号
func startVersionChecker(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            checkAndRefreshCache(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func checkAndRefreshCache(ctx context.Context) {
    tenants := getAllTenants() // 获取所有租户列表
    
    for _, tenantID := range tenants {
        currentVersion := getCurrentVersion(tenantID)
        latestVersion, _ := policyVersionRepo.GetVersionNumber(ctx, tenantID)
        
        if currentVersion != latestVersion {
            log.Infof("检测到版本变化: tenant=%s, %d -> %d", 
                tenantID, currentVersion, latestVersion)
            casbinAdapter.InvalidateCache(ctx)
            setCurrentVersion(tenantID, latestVersion)
        }
    }
}
```

---

### Q4: 多租户场景如何处理？

**问题**：每个租户的策略是独立的，如何避免一个租户的变更导致所有租户缓存失效？

**方案 1: 租户级别的缓存刷新（推荐）**

```go
func handleVersionChange(tenantID string, version int64) {
    // 只刷新指定租户的缓存
    casbinAdapter.InvalidateCacheForTenant(context.Background(), tenantID)
}

// CasbinAdapter 实现
func (a *CasbinAdapter) InvalidateCacheForTenant(ctx context.Context, tenantID string) error {
    a.mu.Lock()
    defer a.mu.Unlock()
    
    // 删除该租户的缓存键
    cacheKey := fmt.Sprintf("tenant:%s:*", tenantID)
    // 清理逻辑...
    
    return nil
}
```

**方案 2: 全局缓存刷新（简单但性能差）**

```go
func handleVersionChange(tenantID string, version int64) {
    // 刷新所有租户的缓存
    casbinAdapter.InvalidateCache(context.Background())
}
```

**权衡**：
- 方案 1: 性能好，但实现复杂
- 方案 2: 实现简单，但大租户变更会影响所有小租户

---

## 🚀 最佳实践

### 1. 服务启动时订阅

```go
// cmd/apiserver/main.go

func main() {
    // ... 初始化组件 ...
    
    // 创建版本通知器
    versionNotifier := redis.NewVersionNotifier(redisClient, "authz:policy_changed")
    
    // 启动协程订阅
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go func() {
        if err := versionNotifier.Subscribe(ctx, handlePolicyVersionChange); err != nil {
            log.Fatalf("订阅失败: %v", err)
        }
    }()
    
    // ... 启动 HTTP 服务 ...
    
    // 优雅关闭
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    
    log.Info("正在关闭服务...")
    versionNotifier.Close()
}
```

---

### 2. 处理函数：快速返回

```go
func handlePolicyVersionChange(tenantID string, version int64) {
    // ✅ 异步处理，避免阻塞 Redis 订阅线程
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        
        log.Infof("[策略变更] tenant=%s, version=%d", tenantID, version)
        
        if err := casbinAdapter.InvalidateCache(ctx); err != nil {
            log.Errorf("刷新缓存失败: %v", err)
            return
        }
        
        log.Infof("[缓存刷新] 租户 %s 缓存已更新", tenantID)
    }()
}
```

---

### 3. 监控和告警

```go
func handlePolicyVersionChange(tenantID string, version int64) {
    // 记录指标
    metrics.PolicyVersionChangedCounter.WithLabelValues(tenantID).Inc()
    
    startTime := time.Now()
    defer func() {
        duration := time.Since(startTime)
        metrics.CacheRefreshDuration.WithLabelValues(tenantID).Observe(duration.Seconds())
    }()
    
    // 刷新缓存
    if err := casbinAdapter.InvalidateCache(context.Background()); err != nil {
        log.Errorf("刷新缓存失败: %v", err)
        metrics.CacheRefreshErrorCounter.WithLabelValues(tenantID).Inc()
        return
    }
    
    log.Infof("缓存刷新成功: tenant=%s, version=%d, duration=%v", 
        tenantID, version, time.Since(startTime))
}
```

---

## 📊 性能考量

### Redis Pub/Sub 性能

- ✅ **低延迟**：通常 < 10ms
- ✅ **高吞吐**：单实例可支持数万 QPS
- ⚠️ **不保证送达**：订阅者离线时收不到消息

### Casbin 缓存刷新性能

- ✅ **InvalidateCache()**: 仅清空内存缓存，< 1ms
- ⚠️ **LoadPolicy()**: 从数据库加载所有策略，10-100ms

**建议**：
- 使用 `InvalidateCache()` 而非 `LoadPolicy()`
- 让 Casbin 在下次 `Enforce()` 时懒加载策略

---

## 🎯 总结

### 核心要点

1. **Redis Pub/Sub 用于分布式缓存失效通知**
   - PAP 修改策略后发布通知
   - 所有 PDP 实例订阅并刷新缓存

2. **业务服务应该订阅**
   - 确保策略变更实时生效
   - 提升用户体验和安全性

3. **处理函数应该简单快速**
   - 只刷新缓存，不重新加载策略
   - 异步处理，避免阻塞

4. **需要兜底方案**
   - 版本号检查（推荐）
   - 定时轮询
   - 监控和告警

---

## 📚 相关文档

- [架构文档](./README.md)
- [Casbin 适配器实现](../infra/casbin/adapter.go)
- [Redis 版本通知器实现](../infra/redis/version_notifier.go)
- [策略版本仓储实现](../infra/mysql/policy/repo.go)

---

**作者**: GitHub Copilot  
**日期**: 2025年10月18日
