# 数据库初始化指南

## 概述

本文档描述 IAM Contracts 项目的数据库初始化流程，包括数据库创建、表结构初始化、种子数据加载等操作。

## 目录

- [快速开始](#快速开始)
- [数据库结构](#数据库结构)
- [初始化脚本](#初始化脚本)
- [种子数据](#种子数据)
- [常见操作](#常见操作)
- [故障排除](#故障排除)

---

## 快速开始

### 前置条件

- MySQL 8.0+ 服务已安装并运行
- MySQL 客户端工具已安装
- 具有数据库创建权限的用户账户

### 基础初始化

最简单的初始化方式（使用默认配置）：

```bash
# 方式1: 使用 Shell 脚本
cd scripts/sql
./init-db.sh

# 方式2: 使用 Makefile
make db-init

# 方式3: 直接使用 MySQL 命令
mysql -u root -p < scripts/sql/init.sql
mysql -u root -p < scripts/sql/seed.sql
```

### 自定义配置

指定数据库连接参数：

```bash
# 使用命令行参数
./init-db.sh -H localhost -P 3306 -u myuser -p mypassword -d iam_contracts

# 使用环境变量
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=mypassword
export DB_NAME=iam_contracts
./init-db.sh
```

---

## 数据库结构

### 数据库配置

- **数据库名称**: `iam_contracts`
- **字符集**: `utf8mb4`
- **排序规则**: `utf8mb4_unicode_ci`
- **存储引擎**: InnoDB

### 模块划分

数据库表按照三个核心模块组织：

#### 1. 用户中心 (User Center)

管理用户、儿童和监护关系。

| 表名 | 说明 | 主要字段 |
|------|------|----------|
| `users` | 用户表 | id, tenant_id, name, phone, email, status |
| `children` | 儿童表 | id, tenant_id, name, gender, birthday |
| `guardianships` | 监护关系表 | id, user_id, child_id, relation |

**关系说明**:

- 一个用户可以监护多个儿童（一对多）
- 一个儿童可以有多个监护人（一对多）
- 监护关系支持撤销（soft delete）

#### 2. 认证中心 (Authentication Center)

管理账户、会话和 JWT 密钥。

| 表名 | 说明 | 主要字段 |
|------|------|----------|
| `accounts` | 账户表 | id, user_id, provider, external_id, password_hash |
| `sessions` | 会话表 | id, user_id, refresh_token, expires_at |
| `signing_keys` | 签名密钥表 | id, kid, algorithm, public_key, private_key |
| `token_blacklist` | Token黑名单表 | id, token_id, user_id, reason |

**认证流程**:

1. 用户使用 `accounts` 进行身份验证
2. 创建 `sessions` 记录会话信息
3. 使用 `signing_keys` 签发 JWT Token
4. 注销时将 Token 加入 `token_blacklist`

#### 3. 授权中心 (Authorization Center)

管理角色、资源和权限。

| 表名 | 说明 | 主要字段 |
|------|------|----------|
| `resources` | 资源表 | id, name, resource_type, resource_path, method |
| `roles` | 角色表 | id, name, code, is_system |
| `user_roles` | 用户角色关联表 | id, user_id, role_id |
| `role_resources` | 角色资源关联表 | id, role_id, resource_id, actions |
| `casbin_rule` | Casbin策略规则表 | ptype, v0, v1, v2, v3 |

**RBAC 模型**:

```text
User ──> UserRole ──> Role ──> RoleResource ──> Resource
                       │
                       └──> Casbin Policy
```

#### 4. 系统表

| 表名 | 说明 | 主要字段 |
|------|------|----------|
| `tenants` | 租户表 | id, name, code, status |
| `system_configs` | 系统配置表 | id, tenant_id, config_key, config_value |
| `operation_logs` | 操作日志表 | id, user_id, operation_type, resource_type |

### ER 图概览

```text
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│   Tenants   │────────>│    Users     │<────────│  Accounts   │
└─────────────┘         └──────────────┘         └─────────────┘
                              │   │
                              │   │
                              │   └──────────────┐
                              │                  │
                        ┌─────▼──────┐    ┌──────▼────────┐
                        │ UserRoles  │    │ Guardianships │
                        └─────┬──────┘    └──────┬────────┘
                              │                  │
                        ┌─────▼──────┐    ┌──────▼────────┐
                        │   Roles    │    │   Children    │
                        └─────┬──────┘    └───────────────┘
                              │
                        ┌─────▼──────────┐
                        │ RoleResources  │
                        └─────┬──────────┘
                              │
                        ┌─────▼──────┐
                        │ Resources  │
                        └────────────┘
```

---

## 初始化脚本

### 脚本文件列表

```text
scripts/sql/
├── init.sql          # 数据库和表结构初始化
├── seed.sql          # 种子数据加载
├── init-db.sh        # Shell 初始化脚本
└── reset-db.sh       # 数据库重置脚本（开发用）
```

### init.sql - 表结构初始化

创建所有数据库表和索引。

**主要功能**:

- 创建数据库（如果不存在）
- 创建所有表结构
- 创建索引和外键约束
- 添加表注释和字段注释

**执行方式**:

```bash
# 直接执行
mysql -u root -p < scripts/sql/init.sql

# 使用脚本
./scripts/sql/init-db.sh --schema-only
```

### seed.sql - 种子数据加载

加载初始数据和测试数据。

**包含数据**:

1. **租户数据**: 系统租户 + 演示租户
2. **管理员用户**: 超级管理员 + 租户管理员
3. **系统角色**: SUPER_ADMIN, ADMIN, USER, GUARDIAN
4. **资源定义**: API 端点资源
5. **权限配置**: 角色-资源关联
6. **Casbin 策略**: RBAC 策略规则
7. **测试数据**: 测试用户、儿童、监护关系
8. **系统配置**: JWT、密钥轮换等配置

**默认账户**:

| 用户名 | 密码 | 角色 | 租户 |
|--------|------|------|------|
| admin | admin123 | 超级管理员 | 系统租户 |
| zhangsan | admin123 | 管理员 | 演示租户 |
| lisi | admin123 | 监护人 | 演示租户 |

⚠️ **安全提示**: 生产环境部署后请立即修改默认密码！

**执行方式**:

```bash
# 直接执行
mysql -u root -p iam_contracts < scripts/sql/seed.sql

# 使用脚本
./scripts/sql/init-db.sh --seed-only
```

### init-db.sh - Shell 初始化脚本

功能完善的 Shell 脚本，提供交互式初始化流程。

**主要特性**:

- ✅ 彩色输出
- ✅ 连接测试
- ✅ 确认提示
- ✅ 错误处理
- ✅ 灵活配置

**使用方式**:

```bash
# 查看帮助
./scripts/sql/init-db.sh --help

# 使用默认配置
./scripts/sql/init-db.sh

# 指定连接参数
./scripts/sql/init-db.sh -H localhost -u root -p mypassword

# 使用环境变量
export DB_HOST=localhost
export DB_USER=root
export DB_PASSWORD=mypassword
./scripts/sql/init-db.sh

# 仅创建表结构
./scripts/sql/init-db.sh --schema-only

# 仅加载种子数据
./scripts/sql/init-db.sh --seed-only

# 跳过确认提示
./scripts/sql/init-db.sh --skip-confirm
```

**环境变量**:

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `DB_HOST` | 数据库主机 | 127.0.0.1 |
| `DB_PORT` | 数据库端口 | 3306 |
| `DB_USER` | 数据库用户 | root |
| `DB_PASSWORD` | 数据库密码 | (空) |
| `DB_NAME` | 数据库名称 | iam_contracts |

### reset-db.sh - 数据库重置脚本

⚠️ **危险操作**: 此脚本将完全删除数据库及所有数据，仅供开发环境使用！

**使用场景**:

- 开发环境快速重置
- 测试数据清理
- 数据结构变更后重建

**使用方式**:

```bash
# 交互式重置（需要二次确认）
./scripts/sql/reset-db.sh

# 强制重置（跳过确认）
./scripts/sql/reset-db.sh --force

# 指定数据库
./scripts/sql/reset-db.sh -H localhost -u root -p mypassword
```

**安全措施**:

1. 需要输入数据库名称确认
2. 需要输入 "yes" 进行二次确认
3. 红色警告提示
4. 仅建议在开发环境使用

---

## 种子数据

### 数据结构

#### 租户数据

```sql
tenant-system  (系统租户)
tenant-demo    (演示租户)
```

#### 用户和账户

**系统管理员**:

- ID: `user-admin`
- 租户: `tenant-system`
- 角色: 超级管理员
- 账户: `admin` / `admin123`

**演示租户用户**:

| ID | 姓名 | 手机 | 角色 | 账户 |
|----|------|------|------|------|
| user-demo-001 | 张三 | 13800138000 | 管理员 | zhangsan / admin123 |
| user-demo-002 | 李四 | 13800138001 | 监护人 | lisi / admin123 |

#### 角色定义

| 角色ID | 角色名称 | 角色编码 | 权限范围 | 系统角色 |
|--------|----------|----------|----------|----------|
| role-superadmin | 超级管理员 | SUPER_ADMIN | 所有权限 | ✅ |
| role-admin | 管理员 | ADMIN | 租户内所有权限 | ✅ |
| role-user | 普通用户 | USER | 基础只读权限 | ✅ |
| role-guardian | 监护人 | GUARDIAN | 儿童管理权限 | ❌ |

#### 资源定义

**用户中心资源** (15个):

- 用户管理: LIST, CREATE, GET, UPDATE, DELETE
- 儿童管理: LIST, CREATE, GET, UPDATE, DELETE
- 监护关系: LIST, CREATE, GET, REVOKE

**认证中心资源** (4个):

- LOGIN, LOGOUT, REFRESH, JWKS

**授权中心资源** (6个):

- 角色管理: LIST, CREATE, GET, UPDATE, DELETE, ASSIGN

#### 测试数据

**儿童数据**:

- 小明 (男, 2018-05-15)
- 小红 (女, 2019-08-20)
- 小刚 (男, 2020-03-10)

**监护关系**:

- 张三 → 小明 (father)
- 张三 → 小红 (father)
- 李四 → 小明 (mother)

#### 系统配置

```json
{
  "jwt.access_token_ttl": 3600,           // 1小时
  "jwt.refresh_token_ttl": 2592000,       // 30天
  "jwt.key_rotation_days": 90,            // 90天
  "guardian.max_children_per_user": 5     // 最多5个儿童
}
```

---

## 常见操作

### 使用 Makefile 命令

Makefile 提供了便捷的数据库操作命令：

```bash
# 初始化数据库
make db-init

# 仅创建表结构
make db-migrate

# 仅加载种子数据
make db-seed

# 重置数据库（危险操作）
make db-reset

# 连接到数据库
make db-connect

# 查看数据库状态
make db-status
```

### 手动执行 SQL

```bash
# 连接到数据库
mysql -u root -p -h localhost -P 3306

# 使用数据库
USE iam_contracts;

# 查看所有表
SHOW TABLES;

# 查看表结构
DESCRIBE users;
SHOW CREATE TABLE users;

# 查看表数据
SELECT * FROM users;
SELECT * FROM roles;

# 查看数据统计
SELECT TABLE_NAME, TABLE_ROWS, TABLE_COMMENT
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = 'iam_contracts';
```

### Docker 环境初始化

如果使用 Docker 运行 MySQL：

```bash
# 启动 MySQL 容器
make docker-mysql-up

# 等待 MySQL 启动完成
sleep 10

# 初始化数据库
make db-init

# 或者在容器内执行
docker exec -i iam-mysql mysql -uroot -proot < scripts/sql/init.sql
docker exec -i iam-mysql mysql -uroot -proot < scripts/sql/seed.sql
```

### 备份和恢复

**备份数据库**:

```bash
# 备份完整数据库
mysqldump -u root -p iam_contracts > backup_$(date +%Y%m%d_%H%M%S).sql

# 仅备份结构
mysqldump -u root -p --no-data iam_contracts > schema_backup.sql

# 仅备份数据
mysqldump -u root -p --no-create-info iam_contracts > data_backup.sql
```

**恢复数据库**:

```bash
# 恢复数据库
mysql -u root -p iam_contracts < backup.sql

# 从备份重建
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS iam_contracts;"
mysql -u root -p iam_contracts < backup.sql
```

### 数据验证

初始化完成后，执行以下查询验证数据：

```sql
-- 1. 验证租户
SELECT id, name, code, status FROM tenants;

-- 2. 验证用户
SELECT u.id, u.name, u.phone, t.name as tenant_name
FROM users u
JOIN tenants t ON u.tenant_id = t.id;

-- 3. 验证角色分配
SELECT 
    u.name as user_name,
    r.name as role_name,
    ur.granted_at
FROM user_roles ur
JOIN users u ON ur.user_id = u.id
JOIN roles r ON ur.role_id = r.id;

-- 4. 验证资源数量
SELECT resource_type, COUNT(*) as count
FROM resources
GROUP BY resource_type;

-- 5. 验证监护关系
SELECT 
    c.name as child_name,
    u.name as guardian_name,
    g.relation
FROM guardianships g
JOIN children c ON g.child_id = c.id
JOIN users u ON g.user_id = u.id
WHERE g.revoked_at IS NULL;

-- 6. 验证权限配置
SELECT 
    r.name as role_name,
    COUNT(DISTINCT rr.resource_id) as resource_count
FROM roles r
LEFT JOIN role_resources rr ON r.id = rr.role_id
GROUP BY r.id, r.name;
```

---

## 故障排除

### 常见问题

#### 1. 连接失败

**错误**: `ERROR 2003: Can't connect to MySQL server`

**解决方案**:

```bash
# 检查 MySQL 服务状态
sudo systemctl status mysql      # Linux
brew services list               # macOS

# 启动 MySQL
sudo systemctl start mysql       # Linux
brew services start mysql        # macOS

# 检查端口
netstat -an | grep 3306
lsof -i :3306
```

#### 2. 权限不足

**错误**: `ERROR 1045: Access denied for user`

**解决方案**:

```bash
# 确认用户权限
mysql -u root -p -e "SHOW GRANTS FOR 'root'@'localhost';"

# 授予权限
mysql -u root -p -e "GRANT ALL PRIVILEGES ON iam_contracts.* TO 'myuser'@'localhost';"
mysql -u root -p -e "FLUSH PRIVILEGES;"
```

#### 3. 数据库已存在

**错误**: 数据库已存在，初始化失败

**解决方案**:

```bash
# 方式1: 使用重置脚本
./scripts/sql/reset-db.sh

# 方式2: 手动删除
mysql -u root -p -e "DROP DATABASE IF EXISTS iam_contracts;"
./scripts/sql/init-db.sh

# 方式3: 使用 Makefile
make db-reset
```

#### 4. 外键约束错误

**错误**: `ERROR 1215: Cannot add foreign key constraint`

**解决方案**:

- 确保 MySQL 版本 >= 8.0
- 检查表是否使用 InnoDB 引擎
- 确保外键列数据类型匹配
- 先创建父表，再创建子表

```sql
-- 检查表引擎
SHOW TABLE STATUS WHERE Name='users';

-- 修改表引擎
ALTER TABLE users ENGINE=InnoDB;
```

#### 5. 字符集问题

**错误**: 中文显示乱码

**解决方案**:

```sql
-- 查看数据库字符集
SHOW VARIABLES LIKE 'character_set%';

-- 修改数据库字符集
ALTER DATABASE iam_contracts CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 修改表字符集
ALTER TABLE users CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

#### 6. 脚本执行权限

**错误**: `Permission denied: ./init-db.sh`

**解决方案**:

```bash
# 添加执行权限
chmod +x scripts/sql/*.sh

# 或直接使用 bash 执行
bash scripts/sql/init-db.sh
```

### 日志调试

启用 MySQL 查询日志：

```sql
-- 启用一般查询日志
SET GLOBAL general_log = 'ON';
SET GLOBAL general_log_file = '/tmp/mysql-general.log';

-- 启用慢查询日志
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL slow_query_log_file = '/tmp/mysql-slow.log';
SET GLOBAL long_query_time = 2;

-- 查看日志
SHOW VARIABLES LIKE 'general_log%';
SHOW VARIABLES LIKE 'slow_query_log%';
```

### 性能检查

```sql
-- 检查表大小
SELECT 
    TABLE_NAME,
    ROUND(((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024), 2) AS 'Size (MB)'
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = 'iam_contracts'
ORDER BY (DATA_LENGTH + INDEX_LENGTH) DESC;

-- 检查索引使用
SHOW INDEX FROM users;

-- 分析查询性能
EXPLAIN SELECT * FROM users WHERE phone = '13800138000';
```

---

## 附录

### 表结构快速参考

详细的表结构说明请参考：

- [数据模型文档](./uc/DATA_MODELS.md) - UC 模块数据模型
- [认证设计文档](./authn/SECURITY_DESIGN.md) - 认证模块数据库设计
- [授权架构文档](./authz/README.md) - 授权模块数据库设计

### 相关命令

```bash
# Makefile 命令
make db-init          # 初始化数据库
make db-migrate       # 执行迁移
make db-seed          # 加载种子数据
make db-reset         # 重置数据库
make db-connect       # 连接到数据库
make db-status        # 查看数据库状态

# Shell 脚本
./scripts/sql/init-db.sh          # 初始化
./scripts/sql/reset-db.sh         # 重置

# MySQL 命令
mysql -u root -p                  # 连接
mysqldump -u root -p              # 备份
mysqlcheck -u root -p             # 检查
```

### 配置文件

数据库配置文件位置：

- 应用配置: `configs/apiserver.prod.yaml`
- MySQL 配置: `configs/mysql/my.cnf`
- 环境变量: `configs/env/config.prod.env`

### 下一步

- 阅读 [架构概览](./architecture-overview.md) 了解系统架构
- 阅读 [Makefile 指南](./MAKEFILE_GUIDE.md) 了解更多命令
- 阅读各模块文档了解业务逻辑
  - [UC 模块](./uc/README.md)
  - [AuthN 模块](./authn/README.md)
  - [AuthZ 模块](./authz/README.md)

---

## 版本历史

| 版本 | 日期 | 变更说明 |
|------|------|----------|
| 1.0.0 | 2025-10-18 | 初始版本，包含完整的初始化脚本和文档 |

---

**最后更新**: 2025-10-18  
**维护者**: IAM Contracts Team
