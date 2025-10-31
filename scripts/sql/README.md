# 数据库脚本说明

本目录包含 IAM Contracts 项目的数据库初始化脚本和工具。

## ⚠️ 重要提示

**这些脚本主要用于开发和测试环境。**

**生产环境部署请使用数据库迁移工具**，参见：

- 📖 [数据库迁移指南](../../docs/DATABASE_MIGRATION_GUIDE.md) - **强烈推荐阅读**
- 🔧 推荐工具：[golang-migrate](https://github.com/golang-migrate/migrate)
- 🐳 容器化部署时，迁移文件会被嵌入到二进制文件中，自动执行

## 📋 文件列表

| 文件 | 说明 | 状态 | 类型 |
|------|------|------|------|
| `init.sql` | 数据库表结构初始化 SQL (v3.0) | ✅ 最新 | SQL脚本 |
| `seed.sql` | 种子数据加载 SQL (v3.0) | ✅ 最新 | SQL脚本 |
| `init-db.sh` | 数据库初始化 Shell 脚本 | ✅ 可用 | Shell脚本 |
| `reset-db.sh` | 数据库重置 Shell 脚本（开发环境） | ✅ 可用 | Shell脚本 |
| `CHANGELOG.md` | 数据库变更日志 | 📝 文档 | Markdown |
| `README.md` | 本说明文件 | � 文档 | Markdown |

## 🔄 版本说明

***当前版本: v3.0 (2025-10-31)***

### 主要特性

- ✅ 所有表使用 `iam_` 前缀
- ✅ ID 类型统一为 `BIGINT UNSIGNED` (Snowflake ID)
- ✅ 完整的审计字段 (`created_by`, `updated_by`, `deleted_by`, `version`)
- ✅ 软删除支持 (`deleted_at`)
- ✅ utf8mb4 字符集和 InnoDB 引擎
- ✅ 与 `configs/mysql/schema.sql` 完全同步

### 与 v2.0 的主要区别

1. 表名统一添加 `iam_` 前缀
2. 修复了字段类型和索引定义
3. 添加了缺失的系统字段 (如 `is_system`)
4. 完善了索引策略 (复合索引、二级索引)

详细对比请查看 `CHANGELOG.md`。

## 🚀 快速开始

### 方式1: 直接使用 MySQL 客户端

```bash
# 1. 创建数据库和表结构
mysql -u root -p < scripts/sql/init.sql

# 2. 加载种子数据
mysql -u root -p < scripts/sql/seed.sql
```

### 方式2: 使用初始化脚本（推荐）

```bash
# 完整初始化（创建表 + 加载种子数据）
cd scripts/sql
./init-db.sh

# 仅创建表结构
./init-db.sh --schema-only

# 仅加载种子数据
./init-db.sh --seed-only

# 指定数据库连接
./init-db.sh -H localhost -u root -p yourpassword
```

### 方式3: 使用 Makefile

```bash
# 完整初始化
make db-init

# 仅迁移表结构
make db-migrate

# 仅加载种子数据
make db-seed

# 重置数据库（危险操作！）
make db-reset
```

## 📦 数据库结构

### 模块划分

| 模块 | 表数 | 主要表 |
|------|------|--------|
| **用户中心 (UC)** | 3 | iam_users, iam_children, iam_guardianships |
| **认证 (Authn)** | 6 | iam_auth_accounts, iam_auth_wechat_accounts, iam_auth_operation_accounts, iam_jwks_keys, iam_auth_sessions, iam_auth_token_blacklist |
| **授权 (Authz)** | 5 | iam_authz_resources, iam_authz_roles, iam_authz_assignments, iam_authz_policy_versions, iam_casbin_rule |
| **身份提供商 (IDP)** | 1 | iam_idp_wechat_apps |
| **平台/系统** | 5 | iam_tenants, iam_operation_logs, iam_audit_logs, iam_data_dictionary, iam_schema_version |

### 默认数据

执行 `seed.sql` 后会创建：

- **租户**: 1 个默认租户 + 1 个演示租户
- **用户**: 5 个测试用户（包括管理员）
- **儿童**: 4 个测试儿童档案
- **监护关系**: 4 条监护关系记录
- **系统角色**: 3 个（super_admin, tenant_admin, user）
- **资源**: 5 个权限资源
- **Casbin 策略**: 基础 RBAC 规则
- **数据字典**: 性别、用户状态、监护关系类型

## 🔧 脚本说明

### init.sql

完整的数据库表结构定义，包括：

**特性**:

- 所有表使用 `iam_` 前缀
- BIGINT UNSIGNED 类型的 Snowflake ID
- 完整的审计字段（created_by, updated_by, deleted_by, version）
- 软删除支持（deleted_at）
- 合理的索引设计（主键、唯一索引、复合索引、二级索引）

**同步来源**: `configs/mysql/schema.sql`

### seed.sql

种子数据脚本，提供：

- 基础租户和角色配置
- 测试用户和儿童数据
- 权限资源定义
- RBAC 策略规则
- 数据字典初始化

**默认管理员账号**:

- 用户名: `admin`
- 密码: 需通过应用程序设置

⚠️ **安全提示**: 生产环境部署后请立即修改默认配置！

### init-db.sh

功能完善的初始化 Shell 脚本:

**功能**:

- ✅ 连接测试
- ✅ 交互式确认
- ✅ 彩色输出
- ✅ 错误处理
- ✅ 灵活配置

**使用示例**:

```bash
# 查看帮助
./init-db.sh --help

# 完整初始化
./init-db.sh -H localhost -u root -p yourpassword

# 仅创建表结构
./init-db.sh --schema-only

# 仅加载种子数据
./init-db.sh --seed-only
```

### reset-db.sh

⚠️ **危险操作**: 完全删除数据库及所有数据！

**安全措施**:

- 需要输入数据库名称确认
- 需要输入 "yes" 进行二次确认
- 红色警告提示
- **仅建议在开发环境使用**

## 📝 数据验证

初始化完成后，可以执行以下查询验证：

```sql
-- 查看所有表
SHOW TABLES;

-- 查看租户
SELECT id, name, code, status FROM iam_tenants;

-- 查看用户
SELECT id, name, phone, email, status FROM iam_users;

-- 查看儿童
SELECT id, name, gender, birthday FROM iam_children;

-- 查看监护关系
SELECT 
    g.id,
    u.name AS guardian_name,
    c.name AS child_name,
    g.relation,
    g.established_at
FROM iam_guardianships g
JOIN iam_users u ON g.user_id = u.id
JOIN iam_children c ON g.child_id = c.id
WHERE g.deleted_at IS NULL;

-- 查看角色
SELECT id, name, display_name, tenant_id, is_system FROM iam_authz_roles;

-- 查看角色赋权
SELECT 
    a.id,
    a.subject_type,
    a.subject_id,
    r.name AS role_name,
    a.tenant_id
FROM iam_authz_assignments a
JOIN iam_authz_roles r ON a.role_id = r.id
WHERE a.deleted_at IS NULL;
```

## 🐛 故障排除

### 连接失败

```bash
# 检查 MySQL 服务状态
sudo systemctl status mysql      # Linux
brew services list               # macOS

# 启动 MySQL 服务
sudo systemctl start mysql       # Linux
brew services start mysql        # macOS
```

### 权限不足

```bash
# 创建用户并授权
mysql -u root -p
```

```sql
CREATE USER 'iam_user'@'localhost' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON iam_contracts.* TO 'iam_user'@'localhost';
FLUSH PRIVILEGES;
```

### 脚本执行权限

```bash
# 添加执行权限
chmod +x scripts/sql/*.sh
```

### 字符集问题

```bash
# 检查数据库字符集
mysql -u root -p -e "SHOW CREATE DATABASE iam_contracts;"

# 应该显示: utf8mb4 和 utf8mb4_unicode_ci
```

### 表已存在

```bash
# 如果需要重建表，使用 reset-db.sh
cd scripts/sql
./reset-db.sh

# 或手动删除数据库
mysql -u root -p -e "DROP DATABASE IF EXISTS iam_contracts;"
mysql -u root -p < init.sql
```

## 📚 相关文档

- **Schema 定义**: `configs/mysql/schema.sql` - 数据库表结构的权威定义
- **表结构参考**: `configs/mysql/TABLE_REFERENCE.md` - 各表字段说明
- **建表语句**: `configs/mysql/01_users.sql` 等 - 各模块的建表脚本
- **变更日志**: `scripts/sql/CHANGELOG.md` - 数据库版本变更记录

## 🔐 安全建议

### 生产环境部署

1. **修改默认配置**
   - 更改默认管理员密码
   - 使用强密码策略
   - 限制数据库用户权限

2. **网络安全**
   - 限制数据库访问 IP
   - 使用 SSL/TLS 连接
   - 配置防火墙规则

3. **数据备份**
   - 定期全量备份
   - 配置增量备份
   - 测试恢复流程

4. **审计日志**
   - 启用 MySQL 审计日志
   - 监控敏感操作
   - 定期审查日志

### 开发环境建议

1. 使用独立的开发数据库
2. 不要使用生产环境密码
3. 定期重置测试数据
4. 使用 Docker 隔离环境
