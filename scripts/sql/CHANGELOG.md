# 数据库变更日志

本文件记录 IAM Contracts 项目的数据库结构变更历史。

## 版本规范

遵循语义化版本规范（Semantic Versioning）：

- **MAJOR**: 不兼容的 API 变更
- **MINOR**: 向后兼容的功能新增
- **PATCH**: 向后兼容的问题修复

---

## [1.0.0] - 2025-10-18

### 新增 (Added)

#### 用户中心 (User Center)

- ✅ `users` - 用户表
  - 支持多租户隔离
  - 包含基本信息字段（姓名、手机、邮箱、身份证）
  - 软删除支持
  - 乐观锁版本控制

- ✅ `children` - 儿童表
  - 儿童基本信息管理
  - 包含身高、体重等健康数据
  - 软删除支持

- ✅ `guardianships` - 监护关系表
  - 用户-儿童多对多关系
  - 支持多种监护关系类型（父亲、母亲、祖父母、监护人）
  - 支持监护关系撤销
  - 外键约束确保数据完整性

#### 认证中心 (Authentication Center)

- ✅ `accounts` - 账户表
  - 支持多种认证提供商（微信、企业微信、本地）
  - 密码哈希存储（bcrypt）
  - 与用户表关联
  - 唯一约束：provider + external_id

- ✅ `sessions` - 会话表
  - Refresh Token 管理
  - 设备信息追踪
  - IP 地址和 User Agent 记录
  - 会话过期管理

- ✅ `signing_keys` - 签名密钥表
  - JWT 密钥对管理
  - 支持密钥轮换
  - 多种状态（active, grace, expired）
  - 算法支持（RS256, RS384, RS512）

- ✅ `token_blacklist` - Token 黑名单表
  - 已撤销 Token 管理
  - 支持提前过期
  - 包含撤销原因

#### 授权中心 (Authorization Center)

- ✅ `resources` - 资源表
  - 多种资源类型（api, menu, button, data）
  - 支持层级结构（parent_id）
  - HTTP 方法和路径定义
  - 软删除支持

- ✅ `roles` - 角色表
  - 系统角色和自定义角色
  - 租户隔离
  - 唯一约束：tenant_id + code

- ✅ `user_roles` - 用户角色关联表
  - 用户-角色多对多关系
  - 记录授予人和授予时间
  - 唯一约束：user_id + role_id

- ✅ `role_resources` - 角色资源关联表
  - 角色-资源多对多关系
  - 操作权限配置（read, write, delete）
  - 唯一约束：role_id + resource_id

- ✅ `casbin_rule` - Casbin 策略规则表
  - RBAC 策略存储
  - 支持 Casbin 适配器
  - 6 个灵活的策略值字段

#### 系统表 (System Tables)

- ✅ `tenants` - 租户表
  - 多租户管理
  - 联系人信息
  - 租户状态管理
  - 唯一租户编码

- ✅ `system_configs` - 系统配置表
  - 全局和租户级配置
  - JSON 格式配置值
  - 支持配置热更新

- ✅ `operation_logs` - 操作日志表
  - 用户操作审计
  - 请求和响应数据记录
  - IP 和 User Agent 追踪
  - 操作结果记录

### 索引 (Indexes)

#### 主键索引

- 所有表使用 VARCHAR(64) 或 BIGINT UNSIGNED 作为主键

#### 唯一索引

- `accounts`: (provider, external_id)
- `sessions`: (refresh_token)
- `signing_keys`: (kid)
- `token_blacklist`: (token_id)
- `tenants`: (code)
- `roles`: (tenant_id, code)
- `user_roles`: (user_id, role_id)
- `role_resources`: (role_id, resource_id)
- `casbin_rule`: (ptype, v0, v1, v2, v3, v4, v5)

#### 普通索引

- `users`: tenant_id, phone, email, id_card, status, deleted_at
- `children`: tenant_id, (name, birthday), id_card, deleted_at
- `guardianships`: tenant_id, user_id, child_id, (user_id, child_id), deleted_at
- `accounts`: tenant_id, user_id, status, deleted_at
- `sessions`: tenant_id, user_id, expires_at
- `signing_keys`: status, expires_at
- `token_blacklist`: user_id, expires_at
- `resources`: tenant_id, resource_type, parent_id, status, deleted_at
- `roles`: tenant_id, status, deleted_at
- `user_roles`: tenant_id, user_id, role_id
- `role_resources`: tenant_id, role_id, resource_id
- `tenants`: status, deleted_at
- `system_configs`: config_key
- `operation_logs`: tenant_id, user_id, operation_type, resource_type, created_at

### 外键约束 (Foreign Keys)

- `guardianships`:
  - FK: user_id → users(id) ON DELETE CASCADE
  - FK: child_id → children(id) ON DELETE CASCADE

- `accounts`:
  - FK: user_id → users(id) ON DELETE CASCADE

- `sessions`:
  - FK: user_id → users(id) ON DELETE CASCADE

- `user_roles`:
  - FK: user_id → users(id) ON DELETE CASCADE
  - FK: role_id → roles(id) ON DELETE CASCADE

- `role_resources`:
  - FK: role_id → roles(id) ON DELETE CASCADE
  - FK: resource_id → resources(id) ON DELETE CASCADE

### 种子数据 (Seed Data)

#### 租户

- ✅ 系统租户 (SYSTEM)
- ✅ 演示租户 (DEMO)

#### 用户

- ✅ 系统管理员 (admin)
- ✅ 演示租户管理员 (zhangsan)
- ✅ 演示租户监护人 (lisi)

#### 角色

- ✅ 超级管理员 (SUPER_ADMIN)
- ✅ 管理员 (ADMIN)
- ✅ 普通用户 (USER)
- ✅ 监护人 (GUARDIAN)

#### 资源

- ✅ 用户中心资源 (15个)
  - 用户管理 API (5个)
  - 儿童管理 API (5个)
  - 监护关系管理 API (5个)
- ✅ 认证中心资源 (4个)
  - 登录、登出、刷新、JWKS
- ✅ 授权中心资源 (6个)
  - 角色管理 API

#### 测试数据

- ✅ 测试儿童 (3个)
- ✅ 测试监护关系 (3个)

#### 系统配置

- ✅ JWT 配置
- ✅ 密钥轮换配置
- ✅ 业务规则配置

### 脚本和工具 (Scripts & Tools)

- ✅ `init.sql` - 数据库和表结构初始化 SQL
- ✅ `seed.sql` - 种子数据加载 SQL
- ✅ `init-db.sh` - 数据库初始化 Shell 脚本
- ✅ `reset-db.sh` - 数据库重置 Shell 脚本
- ✅ Makefile 数据库管理命令
  - `make db-init` - 完整初始化
  - `make db-migrate` - 仅创建表结构
  - `make db-seed` - 仅加载种子数据
  - `make db-reset` - 重置数据库
  - `make db-status` - 查看数据库状态
  - `make db-connect` - 连接到数据库
  - `make db-backup` - 备份数据库
- ✅ Docker MySQL 管理命令
  - `make docker-mysql-up` - 启动 MySQL 容器
  - `make docker-mysql-down` - 停止 MySQL 容器
  - `make docker-mysql-clean` - 清理 MySQL 数据
  - `make docker-mysql-logs` - 查看 MySQL 日志

### 文档 (Documentation)

- ✅ [数据库初始化指南](../docs/DATABASE_INITIALIZATION.md)
- ✅ [数据库脚本 README](./README.md)
- ✅ [数据库变更日志](./CHANGELOG.md) (本文件)

---

## 变更模板

### [版本号] - YYYY-MM-DD

#### 新增 (Added)

- 新增的表、字段、索引

#### 变更 (Changed)

- 修改的字段类型、长度、默认值等

#### 废弃 (Deprecated)

- 即将删除的功能（保持兼容）

#### 删除 (Removed)

- 已删除的表、字段、索引

#### 修复 (Fixed)

- 数据修复、约束修正等

#### 安全 (Security)

- 安全相关的变更

---

## 迁移指南

### 从 0.x 升级到 1.0.0

**这是首个正式版本，无需迁移。**

对于新安装：

```bash
# 方式1: 使用 Makefile（推荐）
make db-init

# 方式2: 使用 Shell 脚本
cd scripts/sql
./init-db.sh

# 方式3: 直接执行 SQL
mysql -u root -p < scripts/sql/init.sql
mysql -u root -p iam_contracts < scripts/sql/seed.sql
```

### 备份建议

在执行任何数据库变更前，请务必备份：

```bash
# 完整备份
make db-backup

# 或使用 mysqldump
mysqldump -u root -p iam_contracts > backup_$(date +%Y%m%d_%H%M%S).sql
```

---

## 注意事项

### 数据库版本管理

1. **版本号**: 数据库版本号与项目版本号保持一致
2. **变更记录**: 所有变更必须记录在本文件
3. **向后兼容**: MINOR 和 PATCH 版本应保持向后兼容
4. **迁移脚本**: MAJOR 版本变更需提供迁移脚本

### 变更流程

1. 在开发环境创建变更 SQL
2. 在 `CHANGELOG.md` 记录变更
3. 更新 `init.sql` 反映最新结构
4. 如需要，创建迁移脚本 `migrate_X_to_Y.sql`
5. 在测试环境验证
6. 代码审查
7. 在生产环境执行（需要备份）

### 命名规范

- **表名**: 小写复数，下划线分隔（例如：`user_roles`）
- **字段名**: 小写，下划线分隔（例如：`created_at`）
- **索引名**: `idx_表名_字段名`（例如：`idx_users_phone`）
- **外键名**: `fk_表名_字段`（例如：`fk_accounts_user`）
- **唯一索引**: `uk_表名_字段名`（例如：`uk_tenant_code`）

### 数据类型规范

- **ID**: `VARCHAR(64)` (UUID/ULID) 或 `BIGINT UNSIGNED` (自增)
- **时间**: `TIMESTAMP` 或 `DATETIME`
- **状态**: `VARCHAR(20)` (枚举值)
- **金额**: `DECIMAL(10,2)`
- **文本**: `VARCHAR(n)` (小于 255) 或 `TEXT` (大于 255)
- **布尔**: `TINYINT(1)`

---

## 相关资源

- [数据库初始化指南](../docs/DATABASE_INITIALIZATION.md)
- [架构设计文档](../docs/architecture-overview.md)
- [UC 模块数据模型](../docs/uc/DATA_MODELS.md)
- [AuthN 安全设计](../docs/authn/SECURITY_DESIGN.md)
- [AuthZ 架构设计](../docs/authz/README.md)

---

**维护者**: IAM Contracts Team  
**最后更新**: 2025-10-18  
**版本**: 1.0.0
