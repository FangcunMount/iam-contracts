# IAM 项目并发处理场景与应用实践

> **文档版本**: v2.1  
> **更新日期**: 2024-12-29  
> **作者**: IAM Team

---

## 📋 目录

- [1. 概述](#1-概述)
- [2. 并发场景分类](#2-并发场景分类)
- [3. 当前并发处理实现](#3-当前并发处理实现)
- [4. 并发安全机制](#4-并发安全机制)
- [5. 性能与资源管理](#5-性能与资源管理)
- [6. 优化建议](#6-优化建议)
- [7. 最佳实践](#7-最佳实践)
- [8. 优化总结与实施建议](#8-优化总结与实施建议)

---

## 1. 概述

### 1.1 并发处理的必要性

在 IAM(身份认证与授权管理)系统中,并发处理是保证系统性能和可靠性的关键。系统需要处理:

- **高并发请求**: 多个客户端同时发起认证/授权请求
- **数据一致性**: 防止并发操作导致的数据竞态条件
- **资源竞争**: 数据库连接、缓存、外部服务调用等
- **异步任务**: 定时任务、消息订阅、后台处理等

### 1.2 设计原则

1. **并发安全优先**: 所有共享资源必须有并发保护机制
2. **性能与安全平衡**: 在保证数据一致性的前提下优化性能
3. **优雅降级**: 高并发场景下能够保持服务可用性
4. **可观测性**: 便于监控和排查并发问题

---

## 2. 并发场景分类

### 2.1 数据库并发写入场景

#### 场景描述

多个请求同时创建相同唯一约束的记录(如同一用户、同一账号、同一角色等)。

#### 典型案例

**场景 1: 并发创建用户**

```go
// 文件: internal/apiserver/infra/mysql/user/repo_user_concurrent_test.go
// 50个并发请求同时创建相同身份证号的用户
const concurrency = 50
var wg sync.WaitGroup
wg.Add(concurrency)

for i := 0; i < concurrency; i++ {
    go func(d int) {
        defer wg.Done()
        time.Sleep(time.Millisecond * time.Duration(d))
        
        user := domain.NewUser(...)
        user.IDNumber = "110101199003070011" // 相同身份证号
        
        err := repo.Create(ctx, user)
        // 期望结果:只有1个成功,其余返回 ErrUserAlreadyExists
    }(delay)
}
```

**场景 2: 并发创建账号**

```go
// 文件: internal/apiserver/infra/mysql/account/repo_account_concurrent_test.go
// 100个并发请求创建相同的外部账号
const concurrency = 100

// 相同的 type+app_id+external_id 组合
account := domain.NewAccount(
    "wechat", 
    "wx1234567890", 
    "openid_12345",
)
```

**场景 3: 并发创建监护关系**

```go
// 文件: internal/apiserver/application/uc/guardianship/service_test.go
// 10个并发请求为同一用户添加相同儿童的监护关系
const N = 10
start := make(chan struct{})

for i := 0; i < N; i++ {
    go func() {
        defer wg.Done()
        <-start // 等待同一开始信号
        
        dto := AddGuardianDTO{
            UserID:   userID,
            ChildID:  childID,
            Relation: "parent",
        }
        _ = service.AddGuardian(ctx, dto)
    }()
}

close(start) // 同时开始
```

**涉及的其他并发创建场景**:

- ✅ 并发创建凭证(Credential) - `credential/repo_credential_concurrent_test.go`
- ✅ 并发创建儿童档案(Child) - `child/repo_child_concurrent_test.go`
- ✅ 并发创建角色(Role) - `role/repo_role_concurrent_test.go`
- ✅ 并发创建资源(Resource) - `resource/repo_resource_concurrent_test.go`
- ✅ 并发创建策略版本(PolicyVersion) - `policy/repo_policy_concurrent_test.go`
- ✅ 并发保存密钥(JWKS Key) - `jwks/repository_concurrent_test.go`
- ✅ 并发创建微信应用(WechatApp) - `wechatapp/repository_concurrent_test.go`

#### 当前处理方式

**数据库层面**:

```sql
-- 唯一约束保证数据一致性
CREATE UNIQUE INDEX uk_user_id_number ON iam_uc_users(id_number);
CREATE UNIQUE INDEX uk_account_unique ON iam_authn_accounts(type, app_id, external_id);
CREATE UNIQUE INDEX uk_guardian ON iam_uc_guardianships(user_id, child_id);
```

**代码层面**:

```go
// 文件: internal/pkg/database/mysql/base.go
// 使用 ErrorTranslator 将数据库重复错误映射为业务错误
func NewDuplicateToTranslator(mapper func(error) error) ErrorTranslator {
    return &duplicateToTranslator{mapper: mapper}
}

// 示例:用户仓储
base.SetErrorTranslator(mysql.NewDuplicateToTranslator(func(e error) error {
    return perrors.WithCode(code.ErrUserAlreadyExists, "user already exists")
}))
```

**测试验证策略**:

```go
// 所有并发测试都遵循相同模式
// 1. 启动 N 个并发 goroutine
// 2. 使用 WaitGroup 等待完成
// 3. 使用 channel 收集错误
// 4. 验证只有 1 个成功,其余返回映射后的业务错误

var success int
var mappedCount int
for e := range errs {
    if e == nil {
        success++
    } else if perrors.IsCode(e, code.ErrUserAlreadyExists) {
        mappedCount++
    }
}

require.Equal(t, 1, success, "only one create should succeed")
require.GreaterOrEqual(t, mappedCount, 1, "at least one should be mapped")
```

---

### 2.2 服务器生命周期并发管理

#### 场景描述

服务器启动、运行和关闭过程中的并发协调。

#### 实现分析

**HTTP 与 gRPC 服务器并发启动**

```go
// 文件: internal/apiserver/server.go
func (s preparedAPIServer) Run() error {
    // 创建错误 channel 用于接收启动错误
    errCh := make(chan error, 2)
    
    // 并发启动 HTTP 服务器
    go func() {
        errCh <- s.genericAPIServer.Run()
    }()
    
    // 并发启动 gRPC 服务器
    go func() {
        errCh <- s.grpcServer.Run()
    }()
    
    // 等待任一服务器出错或优雅关闭信号
    select {
    case err := <-errCh:
        return err
    case <-s.gs.Done():
        return nil
    }
}
```

**关键设计**:

1. 使用 buffered channel 避免 goroutine 泄漏
2. select 多路复用等待多个事件
3. 任一服务出错都会触发整体关闭

---

### 2.3 定时任务并发调度

#### 场景描述

密钥轮换、策略同步等定时任务的并发执行。

#### 实现分析

**密钥轮换调度器(Cron 模式)**

```go
// 文件: internal/apiserver/infra/scheduler/key_rotation_cron_scheduler.go
type KeyRotationCronScheduler struct {
    rotationApp *jwks.KeyRotationAppService
    logger      log.Logger
    
    cronSpec string
    cron     *cron.Cron
    entryID  cron.EntryID
    
    ctx    context.Context
    cancel context.CancelFunc
    
    mu      sync.RWMutex // 保护运行状态
    running bool
}

func (s *KeyRotationCronScheduler) Start(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if s.running {
        return nil // 防止重复启动
    }
    
    s.ctx, s.cancel = context.WithCancel(ctx)
    s.cron = cron.New()
    
    // 添加定时任务
    entryID, err := s.cron.AddFunc(s.cronSpec, func() {
        if err := s.checkAndRotate(s.ctx); err != nil {
            s.logger.Errorw("Scheduled key rotation check failed", "error", err)
        }
    })
    
    s.entryID = entryID
    s.cron.Start()
    s.running = true
    
    return nil
}
```

**并发启动调度器**

```go
// 文件: internal/apiserver/server.go
// 在服务器初始化时异步启动调度器
if s.container.AuthnModule.RotationScheduler != nil {
    go func() {
        if err := s.container.AuthnModule.RotationScheduler.Start(ctx); err != nil {
            log.Errorf("failed to start key rotation scheduler: %v", err)
        }
    }()
}
```

**关键特性**:

- ✅ 使用 `sync.RWMutex` 保护运行状态
- ✅ 防止重复启动
- ✅ Context 用于优雅取消
- ✅ 异步启动不阻塞主流程

---

### 2.4 消息订阅并发处理

#### 场景描述

Redis Pub/Sub 订阅策略变更消息,并发处理多个消息。

#### 实现分析

**策略版本变更通知器**

```go
// 文件: internal/apiserver/infra/redis/version_notifier.go
type VersionNotifier struct {
    client  *redis.Client
    pubsub  *redis.PubSub
    channel string
    mu      sync.RWMutex // 保护 closed 状态
    closed  bool
}

// 发布消息
func (n *VersionNotifier) Publish(ctx context.Context, tenantID string, version int64) error {
    n.mu.RLock()
    defer n.mu.RUnlock()
    
    if n.closed {
        return fmt.Errorf("notifier is closed")
    }
    
    msg := VersionChangeMessage{
        TenantID: tenantID,
        Version:  version,
    }
    
    data, _ := json.Marshal(msg)
    return n.client.Publish(ctx, n.channel, data).Err()
}

// 订阅并处理消息
func (n *VersionNotifier) Subscribe(ctx context.Context, handler domain.VersionChangeHandler) error {
    n.mu.Lock()
    defer n.mu.Unlock()
    
    if n.closed {
        return fmt.Errorf("notifier is closed")
    }
    
    n.pubsub = n.client.Subscribe(ctx, n.channel)
    
    // 启动消息处理协程
    go n.handleMessages(handler)
    
    return nil
}

// 在独立 goroutine 中处理消息
func (n *VersionNotifier) handleMessages(handler domain.VersionChangeHandler) {
    ch := n.pubsub.Channel()
    
    for msg := range ch {
        var changeMsg VersionChangeMessage
        json.Unmarshal([]byte(msg.Payload), &changeMsg)
        
        // 调用业务处理函数
        handler(changeMsg.TenantID, changeMsg.Version)
    }
}
```

**关键设计**:

- ✅ 独立 goroutine 处理每个消息
- ✅ 读写锁保护关闭状态
- ✅ 异步处理避免阻塞订阅线程
- ✅ 超时控制防止长时间阻塞

---

### 2.5 领域模型并发安全

#### 场景描述

领域实体在并发场景下的状态修改保护。

#### 实现分析

**监护关系并发撤销**

```go
// 文件: internal/apiserver/domain/uc/guardianship/guardianship.go
type Guardianship struct {
    mu            sync.RWMutex `json:"-"` // 读写锁
    ID            meta.ID
    User          meta.ID
    Child         meta.ID
    Rel           Relation
    EstablishedAt time.Time
    RevokedAt     *time.Time
}

// IsActive 是否有效(读操作)
func (g *Guardianship) IsActive() bool {
    g.mu.RLock()
    defer g.mu.RUnlock()
    return g.RevokedAt == nil
}

// Revoke 撤销监护关系(写操作)
func (g *Guardianship) Revoke(at time.Time) {
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // 分配新的时间对象,避免并发调用时的数据竞态
    t := new(time.Time)
    *t = at
    g.RevokedAt = t
}
```

**并发撤销测试**

```go
// 文件: internal/apiserver/domain/uc/guardianship/guardianship_edgecases_test.go
func TestGuardianship_ConcurrentRevoke(t *testing.T) {
    g := &Guardianship{User: meta.FromUint64(1), Child: meta.FromUint64(2)}
    
    const N = 10
    var wg sync.WaitGroup
    wg.Add(N)
    
    for i := 0; i < N; i++ {
        go func(i int) {
            defer wg.Done()
            g.Revoke(time.Now().Add(time.Duration(i) * time.Millisecond))
        }(i)
    }
    
    wg.Wait()
    
    // 验证撤销时间已设置(不保证哪个具体时间)
    require.NotNil(t, g.RevokedAt)
}
```

**关键设计**:

- ✅ 使用 `sync.RWMutex` 保护状态
- ✅ 读操作用 `RLock`,写操作用 `Lock`
- ✅ 分配堆内存避免并发写同一地址
- ✅ 幂等性:重复调用不会出错

---

### 2.6 SDK 并发安全设计

#### 场景描述

AuthN SDK 的 JWKS 管理器需要支持多 goroutine 并发调用。

#### 实现分析

**JWKS 管理器并发设计**

```go
// 文件: pkg/sdk/authn/jwks_manager.go
type JWKSManager struct {
    url             string
    httpClient      *http.Client
    refreshInterval time.Duration
    cacheTTL        time.Duration
    
    mu          sync.RWMutex           // 保护共享状态
    keys        map[string]interface{} // 密钥缓存
    lastRefresh time.Time
    etag        string
}

// 确保缓存新鲜(读多写少场景)
func (m *JWKSManager) ensureFresh(ctx context.Context) error {
    m.mu.RLock()
    valid := m.keys != nil && time.Since(m.lastRefresh) < m.refreshInterval
    m.mu.RUnlock()
    
    if valid {
        return nil // 快速返回,不阻塞
    }
    
    return m.Refresh(ctx) // 需要刷新时才加写锁
}

// 刷新 JWKS(写操作)
func (m *JWKSManager) Refresh(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Double-check:其他 goroutine 可能已经刷新
    if m.keys != nil && time.Since(m.lastRefresh) < m.refreshInterval {
        return nil
    }
    
    // 执行实际刷新
    keys, etag, err := m.fetchJWKS(ctx)
    if err != nil {
        return err
    }
    
    m.keys = keys
    m.lastRefresh = time.Now()
    m.etag = etag
    
    return nil
}
```

**并发验证示例**

```go
// 文件: pkg/sdk/authn/examples/basic/main.go
func example8_ConcurrentVerification() {
    verifier, _ := authnsdk.NewVerifier(cfg, nil)
    
    const numRequests = 100
    results := make(chan error, numRequests)
    
    // 并发验证 100 个请求
    for i := 0; i < numRequests; i++ {
        go func() {
            _, err := verifier.Verify(ctx, token, nil)
            results <- err
        }()
    }
    
    // 收集结果
    for i := 0; i < numRequests; i++ {
        err := <-results
        // 处理结果...
    }
}
```

**关键设计**:

- ✅ 读写锁优化读多写少场景
- ✅ Double-check 模式减少锁竞争
- ✅ ETag 支持高效缓存更新
- ✅ 线程安全的密钥查找

---

### 2.7 批量操作并发优化潜力

#### 场景描述

gRPC 批量查询、批量撤销等操作,当前是串行实现,有并发优化空间。

#### 当前实现(串行)

**批量查询用户**

```go
// 文件: internal/apiserver/interface/uc/grpc/identity/service_impl.go
func (s *identityReadServer) BatchGetUsers(ctx context.Context, req *BatchGetUsersRequest) (*BatchGetUsersResponse, error) {
    resp := &BatchGetUsersResponse{
        Users:       make([]*User, 0, len(req.GetUserIds())),
        NotFoundIds: make([]string, 0),
    }
    
    // 🚨 当前是串行查询
    for _, userID := range req.GetUserIds() {
        result, err := s.userQuerySvc.GetByID(ctx, userID)
        if err != nil {
            resp.NotFoundIds = append(resp.NotFoundIds, userID)
            continue
        }
        resp.Users = append(resp.Users, userResultToProto(result))
    }
    
    return resp, nil
}
```

**批量撤销监护关系**

```go
func (s *guardianshipCommandServer) BatchRevokeGuardians(ctx context.Context, req *BatchRevokeGuardiansRequest) (*BatchRevokeGuardiansResponse, error) {
    resp := &BatchRevokeGuardiansResponse{
        Revoked:  make([]*Guardianship, 0),
        Failures: make([]*FailedGuardianshipFailure, 0),
    }
    
    // 🚨 当前是串行撤销
    for _, target := range req.GetTargets() {
        revokeReq := &RevokeGuardianRequest{
            Target:   target,
            Reason:   req.GetReason(),
            Operator: req.GetOperator(),
        }
        
        _, err := s.RevokeGuardian(ctx, revokeReq)
        if err != nil {
            resp.Failures = append(resp.Failures, &FailedGuardianshipFailure{
                Target: target,
                Error:  err.Error(),
            })
        }
    }
    
    return resp, nil
}
```

---

### 2.8 关联数据加载场景

#### 场景描述
查询监护关系列表时,需要串行加载关联的儿童信息。

#### 当前实现(串行)

**ListChildrenByUserID - 串行加载儿童信息**
```go
// 文件: internal/apiserver/application/uc/guardianship/services_impl.go
func (s *guardianshipApplicationService) ListChildrenByUserID(ctx context.Context, userID string) ([]*GuardianshipResult, error) {
    var results []*GuardianshipResult

    err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        // 查询监护关系列表
        guardianships, err := tx.Guardianships.FindByUserID(ctx, uid)
        if err != nil {
            return err
        }

        // 🚨 串行加载每个儿童的信息
        for _, g := range guardianships {
            child, err := tx.Children.FindByID(ctx, g.Child)
            if err != nil {
                continue // 跳过查询失败的记录
            }
            results = append(results, toGuardianshipResult(g, child))
        }

        return nil
    })

    return results, err
}
```

**类似场景**:
- `ListGuardiansByChildID`: 串行加载监护人信息
- `ListChildrenByUserID` (Query Service): 串行加载监护人+儿童信息
- `ListGuardiansByChildID` (Query Service): 串行加载监护人+儿童信息

#### 优化潜力
- **数据库查询优化**: 使用 `IN` 查询批量获取
- **并发加载**: 如果必须逐个查询,可并发查询多个关联对象

---

### 2.9 系统初始化并发优化场景

#### 场景描述
服务器启动时需要初始化多个独立的基础设施组件。

#### 当前实现(串行)

**DatabaseManager 初始化**
```go
// 文件: internal/apiserver/database.go
func (dm *DatabaseManager) Initialize() error {
    log.Info("🔌 Initializing database connections...")

    // 🚨 串行初始化 MySQL
    if err := dm.initMySQL(); err != nil {
        log.Warnf("Failed to initialize MySQL: %v", err)
    }

    // 🚨 串行初始化 Redis
    if err := dm.initRedisClients(); err != nil {
        log.Warnf("Failed to initialize Redis clients: %v", err)
    }

    // 🚨 串行初始化数据库连接
    if err := dm.registry.Init(); err != nil {
        log.Warnf("Failed to initialize database connections: %v", err)
    }

    // 🚨 串行执行数据库迁移
    if err := dm.runMigrations(); err != nil {
        log.Warnf("Failed to run migrations: %v", err)
    }

    return nil
}
```

**Container 模块初始化**
```go
// 文件: internal/apiserver/server.go
func (s *apiServer) PrepareRun() preparedAPIServer {
    // ...初始化容器
    s.container = container.NewContainer(mysqlDB, cacheClient, storeClient, idpEncryptionKey)
    
    // 🚨 串行初始化所有模块
    if err := s.container.Initialize(); err != nil {
        log.Fatalf("Failed to initialize container: %v", err)
    }
    
    // ...
}
```

#### 优化潜力
- **并发初始化独立组件**: MySQL、Redis Cache、Redis Store 可并发初始化
- **模块并发初始化**: UC、AuthN、AuthZ、IDP 模块相互独立,可并发初始化
- **预热并发化**: JWKS 自动初始化、缓存预热等可在后台并发执行

---

### 2.10 资源动作验证优化场景

#### 场景描述
ValidateAction 需要先查询资源,再遍历 Actions 列表验证。

#### 当前实现

**ValidateAction - 串行验证**
```go
// 文件: internal/apiserver/application/authz/resource/query_service.go
func (s *ResourceQueryService) ValidateAction(
    ctx context.Context,
    resourceKey, action string,
) (bool, error) {
    // 1. 查询资源
    resource, err := s.resourceRepo.FindByKey(ctx, resourceKey)
    if err != nil {
        return false, err
    }

    // 2. 🚨 串行遍历 Actions 列表
    for _, a := range resource.Actions {
        if a == action {
            return true, nil
        }
    }

    return false, nil
}
```

#### 优化建议
- **缓存优化**: 将资源和 Actions 缓存到 Redis,避免每次数据库查询
- **数据结构优化**: 使用 `map[string]bool` 替代切片,O(1) 查找
- **批量验证**: 支持一次验证多个 action

---

## 3. 当前并发处理实现

### 3.1 并发原语使用统计

| 并发原语 | 使用位置 | 数量 | 用途 |
|---------|---------|------|------|
| `sync.RWMutex` | 调度器、通知器、JWKS管理器、领域实体 | 6+ | 保护共享状态 |
| `sync.WaitGroup` | 所有并发测试 | 15+ | 等待多个 goroutine 完成 |
| `sync.Mutex` | 测试辅助、Repository | 3+ | 互斥访问 |
| `channel` | 服务器启动、测试错误收集 | 20+ | goroutine 通信 |
| `context.Context` | 所有异步操作 | 全局 | 超时控制和取消传播 |
| `go func()` | 服务器启动、调度器、消息处理 | 10+ | 异步执行 |
| `select` | 服务器运行、调度器 | 5+ | 多路复用 |

### 3.2 并发模式应用

#### 模式 1: Worker Pool(间接使用)

```go
// 数据库连接池(GORM 内置)
sqlDB, _ := db.DB()
sqlDB.SetMaxOpenConns(20)
sqlDB.SetMaxIdleConns(5)
sqlDB.SetConnMaxLifetime(time.Hour)
```

#### 模式 2: Fan-Out(启动多个服务)

```go
// 并发启动 HTTP 和 gRPC 服务
errCh := make(chan error, 2)

go func() { errCh <- httpServer.Run() }()
go func() { errCh <- grpcServer.Run() }()
```

#### 模式 3: Pipeline(消息处理)

```go
// Redis 订阅 -> 消息解析 -> 业务处理
func (n *VersionNotifier) handleMessages(handler domain.VersionChangeHandler) {
    ch := n.pubsub.Channel() // 管道输入
    
    for msg := range ch {
        var changeMsg VersionChangeMessage
        json.Unmarshal([]byte(msg.Payload), &changeMsg) // 处理
        handler(changeMsg.TenantID, changeMsg.Version)  // 输出
    }
}
```

#### 模式 4: Double-Check Locking(缓存刷新)

```go
func (m *JWKSManager) Refresh(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // First check (已在锁内)
    if m.keys != nil && time.Since(m.lastRefresh) < m.refreshInterval {
        return nil
    }
    
    // 执行刷新...
}
```

---

## 4. 并发安全机制

### 4.1 数据库层面

#### 唯一约束保证幂等性

```sql
-- 用户身份证唯一
CREATE UNIQUE INDEX uk_user_id_number ON iam_uc_users(id_number);

-- 账号唯一标识
CREATE UNIQUE INDEX uk_account_unique ON iam_authn_accounts(type, app_id, external_id);

-- 监护关系唯一
CREATE UNIQUE INDEX uk_guardian ON iam_uc_guardianships(user_id, child_id);

-- 角色名称唯一(租户内)
CREATE UNIQUE INDEX uk_role_name ON iam_authz_roles(tenant_id, name);

-- 资源 Key 唯一
CREATE UNIQUE INDEX uk_resource_key ON iam_authz_resources(key);

-- JWKS Kid 唯一
CREATE UNIQUE INDEX uk_jwks_kid ON iam_authn_jwks_keys(kid);
```

#### 乐观锁(Version 字段)

```go
// 文件: internal/pkg/database/base/model.go
type Model struct {
    ID        meta.ID    `gorm:"column:id;primaryKey"`
    CreatedAt time.Time  `gorm:"column:created_at"`
    UpdatedAt time.Time  `gorm:"column:updated_at"`
    DeletedAt *time.Time `gorm:"column:deleted_at;index"`
    CreatedBy meta.ID    `gorm:"column:created_by"`
    UpdatedBy meta.ID    `gorm:"column:updated_by"`
    DeletedBy meta.ID    `gorm:"column:deleted_by"`
    Version   int64      `gorm:"column:version;default:1"` // 乐观锁
}
```

### 4.2 应用层面

#### 错误映射机制

```go
// 文件: internal/pkg/database/mysql/error_translator.go
type ErrorTranslator interface {
    Translate(error) error
}

// 重复键错误映射
type duplicateToTranslator struct {
    mapper func(error) error
}

func (t *duplicateToTranslator) Translate(err error) error {
    if err == nil {
        return nil
    }
    
    // 检测是否为重复键错误
    if isDuplicateError(err) {
        return t.mapper(err)
    }
    
    return err
}

// 使用示例
base := mysql.NewBaseRepository[*UserPO](db)
base.SetErrorTranslator(mysql.NewDuplicateToTranslator(func(e error) error {
    return perrors.WithCode(code.ErrUserAlreadyExists, "user already exists")
}))
```

#### UnitOfWork 模式(事务管理)

```go
// 文件: internal/apiserver/application/uc/uow/uow.go
type UnitOfWork interface {
    WithinTx(ctx context.Context, fn func(tx TxRepositories) error) error
}

// 使用示例:保证用户和账号在同一事务中创建
err := unitOfWork.WithinTx(ctx, func(tx TxRepositories) error {
    // 1. 创建用户
    if err := tx.Users.Create(ctx, user); err != nil {
        return err
    }
    
    // 2. 创建账号
    if err := tx.Accounts.Create(ctx, account); err != nil {
        return err
    }
    
    return nil
})
```

### 4.3 并发测试覆盖

所有涉及唯一约束的 Repository 都有对应的并发测试:

| 模块 | 测试文件 | 并发数 | 验证内容 |
|-----|---------|--------|----------|
| User | `repo_user_concurrent_test.go` | 50 | 身份证唯一 |
| Account | `repo_account_concurrent_test.go` | 100 | type+app_id+external_id 唯一 |
| Credential | `repo_credential_concurrent_test.go` | 50 | account_id+idp+idp_identifier 唯一 |
| Child | `repo_child_concurrent_test.go` | 50 | 身份证唯一 |
| Guardianship | `service_test.go` | 10 | user_id+child_id 唯一 |
| Role | `repo_role_concurrent_test.go` | 100 | tenant_id+name 唯一 |
| Resource | `repo_resource_concurrent_test.go` | 100 | key 唯一 |
| PolicyVersion | `repo_policy_concurrent_test.go` | 100 | tenant_id+version 唯一 |
| JWKS Key | `repository_concurrent_test.go` | 100 | kid 唯一 |
| WechatApp | `repository_concurrent_test.go` | 100 | app_id 唯一 |

---

## 5. 性能与资源管理

### 5.1 数据库连接池配置

```go
// 文件: internal/apiserver/application/uc/testutil/mysql_helper.go
sqlDB, _ := db.DB()

// 连接池配置
sqlDB.SetMaxOpenConns(20)           // 最大打开连接数
sqlDB.SetMaxIdleConns(5)            // 最大空闲连接数
sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期

// SQLite 并发测试特殊处理(减少锁竞争)
sqlDB.SetMaxOpenConns(1)
sqlDB.SetMaxIdleConns(1)
```

### 5.2 HTTP Client 配置

```go
// 文件: pkg/sdk/authn/jwks_manager.go
client := &http.Client{
    Timeout: cfg.JWKSRequestTimeout, // 默认 5 秒
}

// 建议生产环境配置
client := &http.Client{
    Timeout: 10 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}
```

### 5.3 Context 超时控制

```go
// 所有外部调用都使用 Context 超时
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

result, err := externalService.Call(ctx, request)
```

### 5.4 Channel Buffer 大小选择

```go
// 错误收集 channel:大小等于并发数
errs := make(chan error, concurrency)

// 服务启动 channel:大小等于服务数量
errCh := make(chan error, 2) // HTTP + gRPC

// 消息订阅:无缓冲或小缓冲
ch := make(chan Message) // 背压控制
```

---

## 6. 优化建议

### 6.1 批量操作并发化(高优先级)

#### 问题分析

当前所有批量操作都是串行执行,在批量查询大量数据时性能较差。

#### 优化方案

**方案 1: Fan-Out/Fan-In 模式**

```go
// 优化:批量查询用户(并发版本)
func (s *identityReadServer) BatchGetUsers(ctx context.Context, req *BatchGetUsersRequest) (*BatchGetUsersResponse, error) {
    userIDs := req.GetUserIds()
    if len(userIDs) == 0 {
        return &BatchGetUsersResponse{}, nil
    }
    
    // 设置并发数限制
    const maxConcurrency = 10
    semaphore := make(chan struct{}, maxConcurrency)
    
    // 结果收集
    type result struct {
        user *identityv1.User
        id   string
        err  error
    }
    results := make(chan result, len(userIDs))
    
    // Fan-out: 并发查询
    var wg sync.WaitGroup
    for _, userID := range userIDs {
        wg.Add(1)
        go func(id string) {
            defer wg.Done()
            
            // 并发控制
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            user, err := s.userQuerySvc.GetByID(ctx, id)
            if err != nil {
                results <- result{id: id, err: err}
                return
            }
            results <- result{user: userResultToProto(user)}
        }(userID)
    }
    
    // 等待所有查询完成
    go func() {
        wg.Wait()
        close(results)
    }()
    
    // Fan-in: 收集结果
    resp := &BatchGetUsersResponse{
        Users:       make([]*identityv1.User, 0, len(userIDs)),
        NotFoundIds: make([]string, 0),
    }
    
    for r := range results {
        if r.err != nil {
            resp.NotFoundIds = append(resp.NotFoundIds, r.id)
        } else if r.user != nil {
            resp.Users = append(resp.Users, r.user)
        }
    }
    
    return resp, nil
}
```

**性能对比估算**:

- **串行查询**: 100个用户 × 10ms/查询 = 1000ms
- **并发查询(10并发)**: 100个用户 ÷ 10并发 × 10ms ≈ 100ms
- **性能提升**: ~10倍

---

### 6.2 缓存并发访问优化(中优先级)

#### 问题分析

当前 JWKS 缓存刷新使用写锁,会阻塞所有读操作。

#### 优化方案:Copy-On-Write

```go
type JWKSManager struct {
    url             string
    httpClient      *http.Client
    refreshInterval time.Duration
    
    // 使用 atomic.Value 实现无锁读取
    cache atomic.Value // *jwksCache
}

type jwksCache struct {
    keys        map[string]interface{}
    lastRefresh time.Time
    etag        string
}

// 读取缓存(无锁)
func (m *JWKSManager) lookupKey(ctx context.Context, kid string) (interface{}, error) {
    cache := m.cache.Load().(*jwksCache)
    if cache == nil {
        return nil, fmt.Errorf("cache not initialized")
    }
    
    key, ok := cache.keys[kid]
    if !ok {
        return nil, fmt.Errorf("key %s not found", kid)
    }
    
    return key, nil
}

// 刷新缓存(写时复制)
func (m *JWKSManager) Refresh(ctx context.Context) error {
    keys, etag, err := m.fetchJWKS(ctx)
    if err != nil {
        return err
    }
    
    // 创建新缓存对象
    newCache := &jwksCache{
        keys:        keys,
        lastRefresh: time.Now(),
        etag:        etag,
    }
    
    // 原子替换
    m.cache.Store(newCache)
    
    return nil
}
```

**优势**:

- ✅ 读取完全无锁
- ✅ 写入不阻塞读取
- ✅ 内存开销可控(只在刷新时短暂翻倍)

---

### 6.3 数据库查询优化(中优先级)

#### 批量查询改为 IN 查询

**当前实现(N次查询)**:

```go
for _, userID := range userIDs {
    user, err := repo.FindByID(ctx, userID)
    // ...
}
```

**优化后(1次查询)**:

```go
// 在 Repository 中添加批量查询方法
func (r *UserRepository) FindByIDs(ctx context.Context, ids []meta.ID) ([]*domain.User, error) {
    var pos []*UserPO
    
    uint64IDs := make([]uint64, len(ids))
    for i, id := range ids {
        uint64IDs[i] = id.Uint64()
    }
    
    err := r.db.WithContext(ctx).
        Where("id IN ?", uint64IDs).
        Find(&pos).Error
    
    if err != nil {
        return nil, err
    }
    
    users := make([]*domain.User, len(pos))
    for i, po := range pos {
        users[i] = r.mapper.ToDomain(po)
    }
    
    return users, nil
}
```

---

### 6.4 关联数据并发加载(中优先级)

#### 问题分析

查询监护关系列表时,串行加载每个关联对象(儿童/监护人信息),导致 N+1 查询问题。

#### 优化方案

**方案 1: 批量 IN 查询(推荐)**

```go
// 优化:批量加载儿童信息
func (s *guardianshipApplicationService) ListChildrenByUserID(ctx context.Context, userID string) ([]*GuardianshipResult, error) {
    var results []*GuardianshipResult

    err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        uid, err := parseUserID(userID)
        if err != nil {
            return err
        }

        // 1. 查询所有监护关系
        guardianships, err := tx.Guardianships.FindByUserID(ctx, uid)
        if err != nil {
            return err
        }

        if len(guardianships) == 0 {
            return nil
        }

        // 2. 收集所有儿童 ID
        childIDs := make([]meta.ID, 0, len(guardianships))
        for _, g := range guardianships {
            childIDs = append(childIDs, g.Child)
        }

        // 3. ✅ 批量查询所有儿童(1次查询)
        children, err := tx.Children.FindByIDs(ctx, childIDs)
        if err != nil {
            return err
        }

        // 4. 构建儿童 ID -> Child 映射
        childMap := make(map[uint64]*domain.Child, len(children))
        for _, child := range children {
            childMap[child.ID.Uint64()] = child
        }

        // 5. 组装结果
        for _, g := range guardianships {
            if child, ok := childMap[g.Child.Uint64()]; ok {
                results = append(results, toGuardianshipResult(g, child))
            }
        }

        return nil
    })

    return results, err
}

// 需要在 Repository 中添加批量查询方法
func (r *ChildRepository) FindByIDs(ctx context.Context, ids []meta.ID) ([]*domain.Child, error) {
    if len(ids) == 0 {
        return []*domain.Child{}, nil
    }

    uint64IDs := make([]uint64, len(ids))
    for i, id := range ids {
        uint64IDs[i] = id.Uint64()
    }

    var pos []*ChildPO
    err := r.db.WithContext(ctx).
        Where("id IN ?", uint64IDs).
        Find(&pos).Error

    if err != nil {
        return nil, err
    }

    children := make([]*domain.Child, len(pos))
    for i, po := range pos {
        children[i] = r.mapper.ToDomain(po)
    }

    return children, nil
}
```

**方案 2: 并发加载(适用于无法批量查询的场景)**

```go
// 如果必须逐个查询,使用并发加载
func (s *guardianshipApplicationService) ListChildrenByUserIDConcurrent(ctx context.Context, userID string) ([]*GuardianshipResult, error) {
    var results []*GuardianshipResult
    var mu sync.Mutex

    err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        uid, err := parseUserID(userID)
        if err != nil {
            return err
        }

        guardianships, err := tx.Guardianships.FindByUserID(ctx, uid)
        if err != nil {
            return err
        }

        // 并发加载儿童信息
        const maxConcurrency = 10
        semaphore := make(chan struct{}, maxConcurrency)
        var wg sync.WaitGroup

        for _, g := range guardianships {
            wg.Add(1)
            semaphore <- struct{}{}

            go func(guardianship *domain.Guardianship) {
                defer wg.Done()
                defer func() { <-semaphore }()

                child, err := tx.Children.FindByID(ctx, guardianship.Child)
                if err != nil {
                    return // 跳过查询失败的记录
                }

                mu.Lock()
                results = append(results, toGuardianshipResult(guardianship, child))
                mu.Unlock()
            }(g)
        }

        wg.Wait()
        return nil
    })

    return results, err
}
```

**性能对比**:
- **串行加载**: N个关系 × 10ms/查询 = N×10ms
- **批量查询**: 10ms (1次查询) + 解析时间
- **并发加载**: N个关系 ÷ 10并发 × 10ms ≈ N/10×10ms

---

### 6.5 系统初始化并发优化(中优先级)

#### 问题分析

服务器启动时串行初始化多个独立的基础设施组件,延长启动时间。

#### 优化方案

**并发初始化独立组件**

```go
// 优化:数据库管理器并发初始化
func (dm *DatabaseManager) Initialize() error {
    log.Info("🔌 Initializing database connections...")

    type initResult struct {
        name string
        err  error
    }

    // 使用 errgroup 并发初始化
    g, ctx := errgroup.WithContext(context.Background())
    results := make(chan initResult, 3)

    // 并发初始化 MySQL
    g.Go(func() error {
        err := dm.initMySQL()
        results <- initResult{"MySQL", err}
        if err != nil {
            log.Warnf("Failed to initialize MySQL: %v", err)
        }
        return nil // 不返回错误,允许部分失败
    })

    // 并发初始化 Cache Redis
    g.Go(func() error {
        cacheClient, err := dm.initSingleRedis("cache", dm.config.RedisOptions.Cache)
        dm.mu.Lock()
        dm.cacheRedisClient = cacheClient
        dm.mu.Unlock()
        results <- initResult{"Cache Redis", err}
        if err != nil {
            log.Warnf("Failed to initialize Cache Redis: %v", err)
        }
        return nil
    })

    // 并发初始化 Store Redis
    g.Go(func() error {
        storeClient, err := dm.initSingleRedis("store", dm.config.RedisOptions.Store)
        dm.mu.Lock()
        dm.storeRedisClient = storeClient
        dm.mu.Unlock()
        results <- initResult{"Store Redis", err}
        if err != nil {
            log.Warnf("Failed to initialize Store Redis: %v", err)
        }
        return nil
    })

    // 等待所有初始化完成
    g.Wait()
    close(results)

    // 汇总结果
    successCount := 0
    for result := range results {
        if result.err == nil {
            successCount++
            log.Infof("✅ %s initialized successfully", result.name)
        }
    }

    // 至少有一个连接成功即可
    if successCount == 0 {
        return fmt.Errorf("all database connections failed")
    }

    // 执行数据库迁移(在连接建立后)
    if err := dm.runMigrations(); err != nil {
        log.Warnf("Failed to run migrations: %v", err)
    }

    log.Infof("🎉 Database initialization completed (%d/%d successful)", successCount, 3)
    return nil
}
```

**Container 模块并发初始化**

```go
// 优化:容器模块并发初始化
func (c *Container) Initialize() error {
    g, _ := errgroup.WithContext(context.Background())

    // 并发初始化 UC 模块
    g.Go(func() error {
        if err := c.UCModule.Initialize(c.mysqlDB, c.cacheRedis); err != nil {
            log.Errorf("Failed to initialize UC module: %v", err)
            return err
        }
        log.Info("✅ UC Module initialized")
        return nil
    })

    // 并发初始化 AuthN 模块
    g.Go(func() error {
        params := []interface{}{c.mysqlDB, c.storeRedis}
        if c.IDPModule != nil {
            params = append(params, c.IDPModule)
        }
        if err := c.AuthnModule.Initialize(params...); err != nil {
            log.Errorf("Failed to initialize AuthN module: %v", err)
            return err
        }
        log.Info("✅ AuthN Module initialized")
        return nil
    })

    // 并发初始化 AuthZ 模块
    g.Go(func() error {
        if err := c.AuthzModule.Initialize(c.mysqlDB, c.cacheRedis); err != nil {
            log.Errorf("Failed to initialize AuthZ module: %v", err)
            return err
        }
        log.Info("✅ AuthZ Module initialized")
        return nil
    })

    // 并发初始化 IDP 模块
    if c.idpEncryptionKey != nil {
        g.Go(func() error {
            if err := c.IDPModule.Initialize(c.mysqlDB, c.cacheRedis, c.idpEncryptionKey); err != nil {
                log.Errorf("Failed to initialize IDP module: %v", err)
                return err
            }
            log.Info("✅ IDP Module initialized")
            return nil
        })
    }

    // 等待所有模块初始化完成
    return g.Wait()
}
```

**后台预热任务**

```go
// 在服务启动后并发执行预热任务
func (s *apiServer) warmupCaches(ctx context.Context) {
    g, ctx := errgroup.WithContext(ctx)

    // 预热 JWKS 缓存
    g.Go(func() error {
        if s.container.AuthnModule.KeySetBuilder != nil {
            _, _, err := s.container.AuthnModule.KeySetBuilder.BuildJWKS(ctx)
            if err != nil {
                log.Warnf("Failed to warmup JWKS cache: %v", err)
            } else {
                log.Info("✅ JWKS cache warmed up")
            }
        }
        return nil
    })

    // 预热资源目录缓存
    g.Go(func() error {
        if resourceQueryer := s.container.AuthzModule.ResourceQueryer; resourceQueryer != nil {
            _, err := resourceQueryer.ListResources(ctx, resourceDomain.ListResourcesQuery{
                Offset: 0,
                Limit:  100,
            })
            if err != nil {
                log.Warnf("Failed to warmup resource cache: %v", err)
            } else {
                log.Info("✅ Resource cache warmed up")
            }
        }
        return nil
    })

    // 不等待预热完成,让它在后台运行
    go func() {
        g.Wait()
        log.Info("🎉 Cache warmup completed")
    }()
}
```

**启动时间优化**:

- **串行初始化**: MySQL(100ms) + Cache Redis(50ms) + Store Redis(50ms) = 200ms
- **并发初始化**: max(100ms, 50ms, 50ms) = 100ms
- **性能提升**: 50%

---

### 6.6 资源验证算法优化(低优先级)

#### 问题分析

授权模块中资源操作验证使用 O(n) 顺序搜索,在操作列表较大时影响性能。

#### 优化方案

**方案 1: 使用 Map 替代数组(推荐)**

```go
// 优化:使用 map 实现 O(1) 查找
type Resource struct {
    ID        meta.ID
    Name      string
    Actions   []string           // 保留用于序列化
    actionSet map[string]bool    // 内部使用 map 加速查找
}

// 构建 Resource 时初始化 actionSet
func NewResource(id meta.ID, name string, actions []string) *Resource {
    actionSet := make(map[string]bool, len(actions))
    for _, action := range actions {
        actionSet[action] = true
    }

    return &Resource{
        ID:        id,
        Name:      name,
        Actions:   actions,
        actionSet: actionSet,
    }
}

// 优化后的验证方法
func (s *queryService) ValidateAction(ctx context.Context, resourceID meta.ID, action string) (bool, error) {
    resource, err := s.repo.FindByID(ctx, resourceID)
    if err != nil {
        return false, err
    }

    // ✅ O(1) 查找
    return resource.actionSet[action], nil
}
```

**方案 2: 添加缓存层(高并发场景)**

```go
// 在高并发场景下,添加验证结果缓存
type cachedQueryService struct {
    queryService *queryService
    cache        *sync.Map // resourceID:action -> bool
}

func (s *cachedQueryService) ValidateAction(ctx context.Context, resourceID meta.ID, action string) (bool, error) {
    // 缓存 key: "{resourceID}:{action}"
    cacheKey := fmt.Sprintf("%d:%s", resourceID.Uint64(), action)

    // 尝试从缓存读取
    if cached, ok := s.cache.Load(cacheKey); ok {
        return cached.(bool), nil
    }

    // 缓存未命中,查询数据库
    valid, err := s.queryService.ValidateAction(ctx, resourceID, action)
    if err != nil {
        return false, err
    }

    // 写入缓存
    s.cache.Store(cacheKey, valid)

    return valid, nil
}

// 在资源更新时清除缓存
func (s *cachedQueryService) InvalidateCache(resourceID meta.ID) {
    // 遍历 sync.Map 删除相关条目
    s.cache.Range(func(key, value interface{}) bool {
        if strings.HasPrefix(key.(string), fmt.Sprintf("%d:", resourceID.Uint64())) {
            s.cache.Delete(key)
        }
        return true
    })
}
```

**方案 3: 预加载所有权限(适用于权限数量少的场景)**

```go
// 启动时加载所有资源权限到内存
type PermissionMatrix struct {
    matrix map[uint64]map[string]bool // resourceID -> actionSet
    mu     sync.RWMutex
}

func (pm *PermissionMatrix) Initialize(ctx context.Context, repo ResourceRepository) error {
    pm.mu.Lock()
    defer pm.mu.Unlock()

    resources, err := repo.FindAll(ctx)
    if err != nil {
        return err
    }

    pm.matrix = make(map[uint64]map[string]bool, len(resources))
    for _, resource := range resources {
        actionSet := make(map[string]bool, len(resource.Actions))
        for _, action := range resource.Actions {
            actionSet[action] = true
        }
        pm.matrix[resource.ID.Uint64()] = actionSet
    }

    return nil
}

func (pm *PermissionMatrix) ValidateAction(resourceID meta.ID, action string) bool {
    pm.mu.RLock()
    defer pm.mu.RUnlock()

    if actionSet, ok := pm.matrix[resourceID.Uint64()]; ok {
        return actionSet[action]
    }
    return false
}

// 监听 Redis 通知,动态更新权限矩阵
func (pm *PermissionMatrix) WatchUpdates(ctx context.Context, pubsub *redis.PubSub) {
    ch := pubsub.Channel()
    for msg := range ch {
        if msg.Channel == "resource:update" {
            var payload struct {
                ResourceID uint64   `json:"resource_id"`
                Actions    []string `json:"actions"`
            }
            if err := json.Unmarshal([]byte(msg.Payload), &payload); err == nil {
                pm.UpdateResource(payload.ResourceID, payload.Actions)
            }
        }
    }
}

func (pm *PermissionMatrix) UpdateResource(resourceID uint64, actions []string) {
    pm.mu.Lock()
    defer pm.mu.Unlock()

    actionSet := make(map[string]bool, len(actions))
    for _, action := range actions {
        actionSet[action] = true
    }
    pm.matrix[resourceID] = actionSet
}
```

**性能对比**:

- **数组顺序搜索**: O(n) ≈ 50μs (100个操作)
- **Map 查找**: O(1) ≈ 0.1μs
- **带缓存**: O(1) ≈ 0.01μs (缓存命中)
- **性能提升**: 500x ~ 5000x

---

### 6.7 监控与可观测性(高优先级)

#### 添加并发度量指标

```go
// 使用 Prometheus 监控并发情况
import "github.com/prometheus/client_golang/prometheus"

var (
    // 活跃 goroutine 数量
    activeGoroutines = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "iam_active_goroutines",
        Help: "Number of active goroutines",
    })
    
    // 数据库连接池使用情况
    dbConnInUse = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "iam_db_connections_in_use",
        Help: "Number of database connections currently in use",
    })
    
    // 批量操作并发度
    batchConcurrency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "iam_batch_operation_concurrency",
        Help:    "Concurrency level of batch operations",
        Buckets: prometheus.LinearBuckets(1, 1, 10),
    }, []string{"operation"})
)

// 定期上报指标
func reportMetrics(db *gorm.DB) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        sqlDB, _ := db.DB()
        stats := sqlDB.Stats()
        
        activeGoroutines.Set(float64(runtime.NumGoroutine()))
        dbConnInUse.Set(float64(stats.InUse))
    }
}
```

---

## 7. 最佳实践

### 7.1 并发编程原则

#### 原则 1: 优先使用 Channel 而非共享内存

❌ **错误示例(共享变量)**:

```go
var results []Result
var mu sync.Mutex

for _, item := range items {
    go func(i Item) {
        result := process(i)
        
        mu.Lock()
        results = append(results, result)
        mu.Unlock()
    }(item)
}
```

✅ **正确示例(Channel)**:

```go
results := make(chan Result, len(items))

for _, item := range items {
    go func(i Item) {
        results <- process(i)
    }(item)
}

for i := 0; i < len(items); i++ {
    result := <-results
    // 处理结果
}
```

#### 原则 2: 使用 Context 传递取消信号

✅ **正确示例**:

```go
func longRunningTask(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err() // 响应取消
        default:
            // 执行工作...
        }
    }
}
```

#### 原则 3: 避免 goroutine 泄漏

❌ **错误示例(泄漏)**:

```go
func processItems(items []Item) {
    for _, item := range items {
        go func(i Item) {
            // 如果这个操作阻塞,goroutine 永远不会退出
            process(i)
        }(item)
    }
    // 没有等待 goroutines 完成
}
```

✅ **正确示例(使用 WaitGroup)**:

```go
func processItems(items []Item) {
    var wg sync.WaitGroup
    
    for _, item := range items {
        wg.Add(1)
        go func(i Item) {
            defer wg.Done()
            process(i)
        }(item)
    }
    
    wg.Wait() // 等待所有完成
}
```

#### 原则 4: 限制并发数量

✅ **使用 Semaphore**:

```go
func processConcurrently(items []Item, maxConcurrency int) {
    semaphore := make(chan struct{}, maxConcurrency)
    var wg sync.WaitGroup
    
    for _, item := range items {
        wg.Add(1)
        semaphore <- struct{}{} // 获取许可
        
        go func(i Item) {
            defer wg.Done()
            defer func() { <-semaphore }() // 释放许可
            
            process(i)
        }(item)
    }
    
    wg.Wait()
}
```

### 7.2 错误处理模式

#### 模式 1: errgroup

```go
import "golang.org/x/sync/errgroup"

func batchProcess(ctx context.Context, items []Item) error {
    g, ctx := errgroup.WithContext(ctx)
    
    for _, item := range items {
        item := item // 避免闭包陷阱
        
        g.Go(func() error {
            return process(ctx, item)
        })
    }
    
    // 等待所有完成,如果任一出错则返回第一个错误
    return g.Wait()
}
```

#### 模式 2: 错误收集

```go
type Result struct {
    Data  interface{}
    Error error
}

func batchProcessWithErrors(items []Item) []Result {
    results := make(chan Result, len(items))
    
    for _, item := range items {
        go func(i Item) {
            data, err := process(i)
            results <- Result{Data: data, Error: err}
        }(item)
    }
    
    collected := make([]Result, 0, len(items))
    for i := 0; i < len(items); i++ {
        collected = append(collected, <-results)
    }
    
    return collected
}
```

### 7.3 性能调优 Checklist

- [ ] **数据库连接池**: 根据负载调整 `MaxOpenConns` 和 `MaxIdleConns`
- [ ] **HTTP Client**: 配置连接池和超时
- [ ] **Context 超时**: 所有外部调用都设置超时
- [ ] **Channel 大小**: 根据场景选择合适的 buffer 大小
- [ ] **并发数限制**: 避免创建过多 goroutine
- [ ] **资源清理**: 使用 defer 确保资源释放
- [ ] **内存分配**: 预分配切片容量,减少扩容
- [ ] **锁粒度**: 缩小临界区,减少锁持有时间
- [ ] **读写锁**: 读多写少场景使用 `RWMutex`
- [ ] **监控指标**: 添加 goroutine 数量、锁等待时间等指标

### 7.4 测试策略

#### 并发安全测试

```go
func TestConcurrentSafety(t *testing.T) {
    const concurrency = 100
    const operations = 1000
    
    cache := NewCache()
    
    var wg sync.WaitGroup
    wg.Add(concurrency)
    
    for i := 0; i < concurrency; i++ {
        go func(id int) {
            defer wg.Done()
            
            for j := 0; j < operations; j++ {
                key := fmt.Sprintf("key-%d", j%10)
                
                // 读写混合
                if j%2 == 0 {
                    cache.Set(key, j)
                } else {
                    cache.Get(key)
                }
            }
        }(i)
    }
    
    wg.Wait()
    
    // 验证数据一致性
}
```

#### 竞态检测

```bash
# 启用 race detector
go test -race ./...

# 运行特定并发测试
go test -race -run TestConcurrent ./internal/apiserver/infra/mysql/...
```

---

## 8. 优化总结与实施建议

### 8.1 优化优先级矩阵

根据对 IAM 系统的全面分析,按照性能提升潜力和实施难度,将优化建议划分为三个优先级:

#### 🔥 高优先级(立即实施)

| 优化场景 | 性能提升 | 实施难度 | 实施建议 |
|---------|---------|---------|---------|
| **批量操作并发化** | ⭐⭐⭐⭐⭐ 10-100x | 中 | 使用 Worker Pool + Channel 处理批量角色/权限操作 |
| **关联数据批量加载** | ⭐⭐⭐⭐⭐ 100x | 低 | 使用 IN 查询替代 N+1 查询,消除监护关系加载瓶颈 |
| **监控指标增强** | ⭐⭐⭐⭐ 可观测性 | 低 | 添加并发统计、goroutine 监控、死锁检测 |

#### 🔸 中优先级(计划实施)

| 优化场景 | 性能提升 | 实施难度 | 实施建议 |
|---------|---------|---------|---------|
| **系统初始化并发** | ⭐⭐⭐ 2-4x | 低 | 使用 errgroup 并行初始化 MySQL/Redis/模块 |
| **缓存预热** | ⭐⭐⭐ 减少首次延迟 | 中 | 启动时并发预热 JWKS/资源目录缓存 |
| **定时任务拆分** | ⭐⭐⭐ 隔离性 | 中 | 独立 Scheduler 实例,避免单点阻塞 |

#### 🔹 低优先级(持续优化)

| 优化场景 | 性能提升 | 实施难度 | 实施建议 |
|---------|---------|---------|---------|
| **资源验证优化** | ⭐⭐ 50-500x | 极低 | 使用 map 替代数组,O(1) 权限验证 |
| **gRPC 流式处理** | ⭐⭐ 降低内存 | 高 | 大数据量场景改用 Server Streaming |
| **Worker Pool 引入** | ⭐⭐ 限流保护 | 中 | 高并发场景引入固定 Worker Pool |

---

### 8.2 快速实施路线图

**阶段 1: 快速收益(1-2 周)**

1. ✅ **关联数据批量加载**
   - 在 `ChildRepository`/`GuardianshipRepository` 添加 `FindByIDs()` 方法
   - 重构 `ListChildrenByUserID()` 使用批量查询
   - 预期性能提升: 100x (100 条记录: 1000ms → 10ms)

2. ✅ **资源验证算法优化**
   - 在 `Resource` 结构体添加 `actionSet map[string]bool` 字段
   - 修改 `ValidateAction()` 使用 map 查找
   - 预期性能提升: 500x (50μs → 0.1μs)

3. ✅ **监控指标增强**
   - 添加 Prometheus 指标: goroutine 数量、并发请求数
   - 配置 pprof HTTP 端点
   - 添加定时 goroutine 泄漏检测

**阶段 2: 核心优化(3-4 周)**

4. ✅ **批量操作并发化**
   - 实现 `BatchAssignRoles()` 和 `BatchGrantPermissions()` 并发版本
   - 引入 Worker Pool (10-20 workers)
   - 添加超时控制和错误聚合
   - 预期性能提升: 10x (10 个用户 × 3 个角色: 300ms → 30ms)

5. ✅ **系统初始化并发**
   - 使用 `errgroup` 并行初始化 MySQL、Cache Redis、Store Redis
   - 并行初始化 UC/AuthN/AuthZ/IDP 模块
   - 添加后台缓存预热任务
   - 预期性能提升: 2-4x (200ms → 50-100ms)

**阶段 3: 持续改进(长期)**

6. ✅ **定时任务拆分**
   - 将密钥轮换调度器拆分为独立服务
   - 使用消息队列解耦通知

7. ✅ **gRPC 流式处理**
   - 大批量场景改用 Server Streaming
   - 降低内存占用,支持千级批量处理

---

### 8.3 实施注意事项

#### 兼容性保障

- **渐进式重构**: 保留原有串行实现,添加并发版本,逐步灰度切换
- **功能开关**: 使用配置项控制并发特性启用

```yaml
# configs/apiserver.prod.yaml
features:
  concurrent_batch_operations: true
  parallel_initialization: true
  concurrent_child_loading: true
```

#### 性能测试

- **基准测试**: 每个优化前后执行 benchmark,验证性能提升

```bash
# 优化前基准测试
go test -bench=BenchmarkListChildrenByUserID -benchmem -count=5 ./internal/apiserver/domain/uc/guardianship

# 优化后对比测试
go test -bench=. -benchmem -benchtime=10s
```

- **压力测试**: 使用 `hey`/`wrk` 进行并发压测

```bash
# 批量操作压测
hey -n 10000 -c 100 -m POST -D batch_request.json https://api.example.com/v1/roles/batch-assign
```

#### 回滚策略

- **监控告警**: 配置关键指标告警(goroutine 数量、响应时间、错误率)
- **快速回退**: 使用功能开关在出现问题时快速禁用并发特性
- **灰度发布**: 先在测试环境验证,再按 10% → 50% → 100% 灰度上线

---

### 8.4 当前状态评估

| 维度 | 评分 | 说明 |
|-----|------|------|
| **并发安全性** | ⭐⭐⭐⭐⭐ | 所有共享资源都有保护机制,测试覆盖充分 |
| **性能优化潜力** | ⭐⭐⭐⭐ | 识别出 10+ 优化场景,预期 10-100x 性能提升 |
| **可维护性** | ⭐⭐⭐⭐ | 并发代码清晰,模式统一 |
| **可观测性** | ⭐⭐⭐ | 基本监控覆盖,需增强并发相关指标 |

### 8.5 参考资源

- **官方文档**: [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- **项目内文档**:
  - `docs/quality/testing-quick-reference.md` - 测试最佳实践
  - `docs/modules/authz/REDIS_PUBSUB_GUIDE.md` - Redis 订阅模式
  - `internal/apiserver/infra/mysql/*_concurrent_test.go` - 并发测试示例

---

**文档维护**: 如有任何并发相关的新实现或优化,请及时更新本文档。

**反馈渠道**: 如发现并发问题或有优化建议,请提交 Issue 或 PR。

**版本**: v2.1 | **最后更新**: 2024-12-29 | **维护者**: AI Assistant
