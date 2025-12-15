# IAM Seed Data Tool

IAM 系统数据库初始化和种子数据填充工具。

## 功能概述

该工具用于快速初始化 IAM 系统的基础数据，包括:

0. **系统初始化** (init) - 一键初始化管理员用户、认证账号和 JWKS 密钥（推荐首次部署使用）
1. **租户数据** (tenants) - 创建默认租户和演示租户
2. **用户中心** (user) - 创建系统管理员和测试用户、儿童档案、监护关系
3. **认证账号** (authn) - 创建运营后台登录账号和凭证
4. **基础角色** (roles) - 创建系统预定义角色
5. **授权资源** (resources) - 创建系统资源定义
6. **角色分配** (assignments) - 为用户分配角色
7. **Casbin策略** (casbin) - 初始化权限控制策略规则
8. **JWKS密钥** (jwks) - 生成JWT签名密钥对
9. **微信应用** (wechatapp) - 创建微信小程序/公众号应用配置
10. **批量家庭** (family) - 批量生成家庭数据（用于测试）

## 快速开始

### 前置条件

1. MySQL 数据库已创建并完成迁移（运行过 `make migrate-up`）
2. Redis 服务已启动（可选）
   - **Cache Redis**: 用于缓存、会话、限流等临时数据
   - **Store Redis**: 用于持久化存储、队列、发布订阅等
3. 准备好存放 JWKS 密钥的目录
4. 配置种子数据文件 `configs/seeddata.yaml`

### 基本用法

```bash
# 使用命令行参数(完整示例)
go run ./cmd/tools/seeddata \
  --dsn "root:password@tcp(localhost:3306)/iam_contracts?parseTime=true&loc=Local" \
  --redis-cache "redis-cache:6379" \
  --redis-cache-username "app" \
  --redis-cache-password "your_cache_password" \
  --redis-store "r-xxx.redis.rds.aliyuncs.com:6380" \
  --redis-store-username "app" \
  --redis-store-password "your_store_password" \
  --keys-dir "./tmp/keys" \
  --casbin-model "./configs/casbin_model.conf" \
  --config "./configs/seeddata.yaml"

# 最小配置(只使用必需参数)
go run ./cmd/tools/seeddata \
  --dsn "root:password@tcp(localhost:3306)/iam_contracts?parseTime=true&loc=Local"

# 使用环境变量
export IAM_SEEDER_DSN="root:password@tcp(localhost:3306)/iam_contracts?parseTime=true&loc=Local"
export IAM_SEEDER_REDIS_CACHE="redis-cache:6379"
export IAM_SEEDER_REDIS_CACHE_PASSWORD="your_cache_password"
go run ./cmd/tools/seeddata
```

### 选择性执行

可以通过 `--steps` 参数指定要执行的步骤：

```bash
# 系统初始化（首次部署推荐，默认步骤）
# 包含：创建管理员用户 + 管理员认证账号（用户名密码）+ JWKS 密钥
go run ./cmd/tools/seeddata \
  --dsn "..."

# 或明确指定 init 步骤
go run ./cmd/tools/seeddata \
  --dsn "..." \
  --steps "init"

# 只创建租户和用户
go run ./cmd/tools/seeddata \
  --dsn "..." \
  --steps "tenants,user"

# 只创建认证账号和JWKS密钥
go run ./cmd/tools/seeddata \
  --dsn "..." \
  --steps "authn,jwks"

# 批量创建家庭数据（用于压力测试）
go run ./cmd/tools/seeddata \
  --dsn "..." \
  --steps "family" \
  --family-count 10000 \
  --worker-count 100

# 执行所有步骤
go run ./cmd/tools/seeddata --dsn "..." --steps "init,tenants,user,authn,roles,resources,assignments,casbin,jwks,wechatapp"
```

## 命令行参数

| 参数 | 环境变量 | 默认值 | 说明 |
|------|----------|--------|------|
| `--dsn` | `IAM_SEEDER_DSN` | 必填 | MySQL数据源名称（DSN） |
| `--redis-cache` | `IAM_SEEDER_REDIS_CACHE` | 可选 | Cache Redis地址（host:port，用于缓存、会话、限流） |
| `--redis-cache-username` | `IAM_SEEDER_REDIS_CACHE_USERNAME` | 可选 | Cache Redis用户名（Redis 6.0+ ACL） |
| `--redis-cache-password` | `IAM_SEEDER_REDIS_CACHE_PASSWORD` | 可选 | Cache Redis密码 |
| `--redis-store` | `IAM_SEEDER_REDIS_STORE` | 可选 | Store Redis地址（host:port，用于持久化存储、队列） |
| `--redis-store-username` | `IAM_SEEDER_REDIS_STORE_USERNAME` | 可选 | Store Redis用户名（Redis 6.0+ ACL） |
| `--redis-store-password` | `IAM_SEEDER_REDIS_STORE_PASSWORD` | 可选 | Store Redis密码 |
| `--keys-dir` | - | `./tmp/keys` | JWKS私钥存储目录 |
| `--casbin-model` | - | `configs/casbin_model.conf` | Casbin模型配置文件路径 |
| `--config` | - | `configs/seeddata.yaml` | 种子数据配置文件路径 |
| `--steps` | - | 所有步骤 | 逗号分隔的步骤列表 |

## 种子数据详情

### 1. 租户数据 (tenants)

创建两个租户：

| ID | 名称 | 代码 | 最大用户数 | 最大角色数 |
|----|------|------|-----------|-----------|
| default | 默认租户 | DEFAULT | 100000 | 1000 |
| demo | 演示租户 | DEMO | 1000 | 100 |

### 2. 用户中心 (user)

#### 用户数据

| 别名 | 姓名 | 手机 | 邮箱 | 身份证号 |
|------|------|------|------|----------|
| admin | 系统管理员 | 10086000001 | <admin@system.com> | 110101199001011001 |
| zhangsan | 张三 | 13800138000 | <zhangsan@example.com> | 110101199001011002 |
| lisi | 李四 | 13800138001 | <lisi@example.com> | 110101199001011003 |
| wangwu | 王五 | 13800138002 | <wangwu@example.com> | 110101198001011004 |
| zhaoliu | 赵六 | 13800138003 | <zhaoliu@example.com> | 110101198001011005 |

#### 儿童档案

| 别名 | 姓名 | 性别 | 生日 | 身高(cm) | 体重(kg) |
|------|------|------|------|----------|----------|
| xiaoming | 小明 | 男 | 2015-01-01 | 145.0 | 35.0 |
| xiaohong | 小红 | 女 | 2015-02-01 | 142.0 | 33.0 |
| xiaogang | 小刚 | 男 | 2016-03-01 | 138.0 | 31.0 |
| xiaoli | 小丽 | 女 | 2018-05-15 | 110.0 | 20.0 |

#### 监护关系

- 王五 → 小明（父母）
- 赵六 → 小红（父母）
- 王五 → 小刚（监护人）
- 赵六 → 小丽（父母）

### 3. 认证账号 (authn)

创建运营后台登录账号：

| 用户名 | 密码 | 关联用户 | 哈希算法 |
|--------|------|----------|----------|
| admin | Admin@123 | 系统管理员 | bcrypt |
| zhangsan | Pass@123 | 张三 | bcrypt |

### 4. 授权资源 (resources)

创建系统资源定义：

| 资源键 | 显示名称 | 应用 | 域 | 操作 |
|--------|----------|------|-----|------|
| uc:users | 用户管理 | iam | uc | create, read, update, delete, list |
| uc:children | 儿童管理 | iam | uc | create, read, update, delete, list |
| uc:guardianships | 监护关系管理 | iam | uc | create, read, update, delete, list, revoke |
| authz:roles | 角色管理 | iam | authz | create, read, update, delete, list, assign |
| authz:policies | 策略管理 | iam | authz | create, read, update, delete, list |

### 5. 角色分配 (assignments)

为用户分配角色：

| 用户 | 角色ID | 租户 |
|------|--------|------|
| admin | 1 (超级管理员) | default |
| zhangsan | 3 | default |
| wangwu | 3 | default |

### 6. Casbin策略 (casbin)

初始化基础权限策略：

**策略规则**：

- `role:super_admin` → `*` @ `default`（超级管理员拥有所有权限）
- `role:tenant_admin` → `tenant:*` @ `default`（租户管理员管理租户资源）

**角色继承**：

- `role:super_admin` 继承 `role:tenant_admin`
- `role:tenant_admin` 继承 `role:user`

### 7. JWKS密钥 (jwks)

生成RSA密钥对用于JWT签名：

- 算法：RS256
- 密钥长度：2048位
- 有效期：1年
- 私钥存储：`--keys-dir` 指定的目录
- 公钥：存储在数据库，通过JWKS端点公开

## 开发指南

### 添加新的种子数据

1. 在 `main.go` 中定义新的步骤常量
2. 创建对应的 `seed<Name>()` 函数
3. 在 `main()` 函数的 switch 中添加新的 case
4. 更新 `defaultSteps` 列表

示例：

```go
const (
    // ... 现有步骤
    stepNewFeature seedStep = "newfeature"
)

func seedNewFeature(ctx context.Context, deps *dependencies, state *seedContext) error {
    // 实现种子数据逻辑
    return nil
}
```

### 数据幂等性

所有种子函数都设计为幂等的，可以安全地多次执行：

- **租户**: 使用 UPSERT 策略（ON CONFLICT UPDATE）
- **用户**: 先查询，存在则更新，不存在则创建
- **账号**: 先查询，存在则跳过创建，直接更新凭证
- **资源**: 先查询，存在则跳过
- **角色分配**: 检查重复错误并忽略
- **JWKS**: 创建新密钥（不会覆盖现有密钥）

### 数据依赖关系

种子数据有明确的依赖关系，建议按以下顺序执行：

```text
tenants (基础)
    ↓
user (依赖租户)
    ↓
authn (依赖用户)
    ↓
roles (独立)
    ↓
resources (独立)
    ↓
assignments (依赖用户、角色和资源)
    ↓
casbin (依赖角色和资源)
    ↓
jwks (独立)
    ↓
wechatapp (依赖租户)
```

## 常见问题

### Q: 双 Redis 架构的用途是什么？

**A**: seeddata 工具支持双 Redis 实例架构:

- **Cache Redis** (`--redis-cache`): 用于临时数据
  - 会话管理(Session)
  - API 限流(Rate Limiting)
  - 临时缓存(Cache)
  - 特点: 可以配置 LRU/LFU 淘汰策略,丢失数据不影响业务

- **Store Redis** (`--redis-store`): 用于持久化数据
  - Token 黑名单
  - 消息队列
  - 发布订阅
  - 分布式锁
  - 特点: 需要持久化配置,数据不能丢失

两个 Redis 实例都是**可选的**。如果不需要相关功能,可以不配置。

### Q: 可以在生产环境运行吗？

**A**: 建议仅在开发和测试环境使用。生产环境应该：

- 手动创建租户
- 使用强密码
- 限制管理员权限
- 通过安全的方式管理密钥

### Q: 密码安全吗？

**A**: 种子数据中的密码仅用于开发/测试。所有密码都使用 bcrypt 哈希存储，但明文密码在代码中可见。**生产环境必须使用强密码并安全管理**。

### Q: 重复运行会有问题吗？

**A**: 不会。所有函数都是幂等的，重复运行会：

- 更新已存在的租户信息
- 跳过已存在的用户、账号、资源
- 忽略重复的角色分配
- 创建新的JWKS密钥（不影响旧密钥）

### Q: 如何只重置某个模块的数据？

**A**: 使用 `--steps` 参数指定要执行的步骤：

```bash
# 只重置认证账号
go run ./cmd/tools/seeddata --dsn "..." --steps "authn"

# 重置用户和账号
go run ./cmd/tools/seeddata --dsn "..." --steps "user,authn"
```

### Q: JWKS密钥存储在哪里？

**A**:

- **私钥**: 存储在 `--keys-dir` 指定的目录（默认 `./tmp/keys`），PEM格式
- **公钥**: 存储在数据库的 `jwks_key` 表中
- **公开访问**: 通过 `/api/v1/authn/.well-known/jwks.json` 端点获取

### Q: 如何清理所有种子数据？

**A**: 可以使用数据库迁移回滚，或手动删除：

```sql
-- ⚠️ 警告：这会删除所有数据
DELETE FROM tenant WHERE id IN ('default', 'demo');
DELETE FROM user WHERE phone LIKE '1380013800%';
-- ... 其他清理语句
```

或者重新运行迁移：

```bash
make migrate-down
make migrate-up
```

## 架构设计

### 模块复用

该工具完全复用了项目的现有代码：

- **领域模型**: `internal/apiserver/domain/*`
- **应用服务**: `internal/apiserver/application/*`
- **仓储实现**: `internal/apiserver/infra/*/mysql`
- **密码工具**: `authentication.HashPassword`
- **工作单元**: UnitOfWork 管理事务

### 依赖注入

```go
type dependencies struct {
    DB          *gorm.DB        // 数据库连接
    Redis       *redis.Client   // Redis客户端（可选）
    KeysDir     string          // JWKS密钥目录
    CasbinModel string          // Casbin模型文件路径
    Logger      log.Logger      // 日志记录器
}
```

### 上下文传递

```go
type seedContext struct {
    Users     map[string]string              // 用户别名 → ID
    Children  map[string]string              // 儿童别名 → ID
    Accounts  map[string]accountDomain.AccountID  // 账号别名 → ID
    Resources map[string]uint64              // 资源键 → ID
}
```

## 维护建议

1. **更新种子数据**: 修改各 `seed*()` 函数中的数据切片
2. **添加新步骤**: 遵循现有模式添加新的 seedStep 和函数
3. **测试**: 在干净的数据库上运行以验证幂等性
4. **文档**: 更新本 README 记录新的种子数据

## 相关文档

- [部署检查清单](../../../docs/DEPLOYMENT_CHECKLIST.md)
- [数据库初始化指南](../../../docs/deploy/DATABASE_INITIALIZATION.md)
- [Makefile使用指南](../../../docs/deploy/MAKEFILE_GUIDE.md)

## 技术支持

如有问题或建议，请联系：

- GitHub Issues: <https://github.com/FangcunMount/iam-contracts/issues>
- Email: <support@yangshujie.com>
