# 集成测试

本目录包含 Authz 模块的集成测试和性能测试。

## 测试列表

### 1. 端到端集成测试 (`authz_e2e_test.go`)

测试完整的授权流程：

```
创建资源 → 创建角色 → 配置策略 → 赋权 → 权限检查 → 撤销权限
```

运行测试：
```bash
go test -v ./test/integration -run TestAuthzEndToEnd
```

预期输出：
```
步骤 1: 创建资源
✓ 创建资源成功: 订单 (ID: 1)
步骤 2: 创建角色
✓ 创建角色成功: 订单管理员 (ID: 1)
步骤 3: 配置策略规则
✓ 添加策略规则成功: order-admin -> order:read
✓ 添加策略规则成功: order-admin -> order:write
步骤 4: 给用户赋权
✓ 用户赋权成功: user-alice -> 订单管理员
步骤 5: 权限检查
✓ 权限检查通过: user-alice 有 order:read 权限
✓ 权限检查通过: user-alice 有 order:write 权限
✓ 权限检查通过: user-alice 没有 order:delete 权限（符合预期）
步骤 6: 撤销权限
✓ 撤销权限成功
步骤 7: 验证权限已撤销
✓ 权限检查通过: user-alice 已没有 order:read 权限

🎉 端到端集成测试通过！
```

### 2. 批量权限检查测试 (`batch_check_test.go`)

测试批量权限检查功能。

运行测试：
```bash
go test -v ./test/integration -run TestBatchPermissionCheck
```

### 3. 性能测试 (`performance_test.go`)

#### 基准测试

```bash
# 单次权限检查性能
go test -bench=BenchmarkPermissionCheck -benchmem ./test/integration

# 批量权限检查性能
go test -bench=BenchmarkBatchPermissionCheck -benchmem ./test/integration

# 所有基准测试
go test -bench=. -benchmem ./test/integration
```

#### 并发测试

```bash
go test -v ./test/integration -run TestConcurrentPermissionCheck
```

## 测试覆盖率

```bash
# 生成覆盖率报告
go test -coverprofile=coverage.out ./test/integration

# 查看覆盖率
go tool cover -func=coverage.out

# 生成 HTML 报告
go tool cover -html=coverage.out -o coverage.html
```

## 测试环境

- **数据库**: SQLite 内存数据库（`:memory:`）
- **Casbin**: 内存模型
- **Redis**: 可选（用于策略变更通知）

## 测试场景

### 场景 1: 正常授权流程

1. 创建资源（order）
2. 创建角色（order-admin）
3. 添加策略规则（order-admin 可以 read/write order）
4. 用户赋权（user-alice 获得 order-admin 角色）
5. 权限检查（验证 user-alice 有相应权限）

### 场景 2: 权限撤销

1. 撤销用户的角色
2. 验证权限检查失败

### 场景 3: 批量权限检查

1. 一次性检查多个权限
2. 验证批量检查结果

### 场景 4: 并发权限检查

1. 100 个并发请求
2. 验证线程安全性
3. 测量性能

## 性能指标

### 预期性能

- **单次权限检查**: < 1ms
- **批量权限检查（5个）**: < 3ms
- **并发 100 个请求**: < 100ms

### 优化建议

1. 启用缓存（`CacheTTL`）
2. 使用批量检查替代多次单独检查
3. 合理使用 Casbin 的内存策略

## 故障排查

### 测试失败

1. 检查数据库迁移是否成功
2. 检查 Casbin 模型配置
3. 查看详细错误日志

### 性能问题

1. 检查是否启用了缓存
2. 分析 Casbin 策略数量
3. 使用 pprof 进行性能分析

```bash
go test -cpuprofile=cpu.prof -memprofile=mem.prof ./test/integration
go tool pprof cpu.prof
```

## 持续集成

在 CI/CD 流程中运行测试：

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: '1.21'
      - name: Run tests
        run: go test -v -cover ./test/integration
```

## 注意事项

1. 测试使用内存数据库，数据不持久化
2. 每个测试独立运行，互不影响
3. 测试完成后自动清理资源
4. 适合 CI/CD 环境快速验证
