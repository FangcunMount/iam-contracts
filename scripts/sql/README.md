# 数据库脚本说明

本目录包含 IAM Contracts 项目的数据库初始化脚本和工具。

## 文件列表

| 文件 | 说明 | 类型 |
|------|------|------|
| `init.sql` | 数据库表结构初始化 SQL | SQL脚本 |
| `seed.sql` | 种子数据加载 SQL | SQL脚本 |
| `init-db.sh` | 数据库初始化 Shell 脚本 | Shell脚本 |
| `reset-db.sh` | 数据库重置 Shell 脚本（开发环境） | Shell脚本 |

## 快速开始

### 方式1: 使用 Makefile（推荐）

```bash
# 完整初始化（创建表 + 加载种子数据）
make db-init

# 仅创建表结构
make db-migrate

# 仅加载种子数据
make db-seed

# 重置数据库（危险操作！）
make db-reset

# 查看数据库状态
make db-status

# 连接到数据库
make db-connect
```

### 方式2: 使用 Shell 脚本

```bash
# 完整初始化
./init-db.sh

# 指定数据库连接
./init-db.sh -H localhost -P 3306 -u root -p mypassword

# 使用环境变量
export DB_HOST=localhost
export DB_USER=root
export DB_PASSWORD=mypassword
./init-db.sh

# 仅创建表结构
./init-db.sh --schema-only

# 仅加载种子数据
./init-db.sh --seed-only

# 查看帮助
./init-db.sh --help
```

### 方式3: 直接执行 SQL

```bash
# 执行初始化脚本
mysql -u root -p < init.sql

# 执行种子数据脚本
mysql -u root -p iam_contracts < seed.sql
```

## 数据库配置

默认配置：

- **主机**: 127.0.0.1
- **端口**: 3306
- **用户**: root
- **密码**: (空)
- **数据库名**: iam_contracts

可以通过以下方式修改：

**Makefile**:
```bash
make db-init DB_HOST=localhost DB_PORT=3306 DB_USER=root DB_PASSWORD=mypass DB_NAME=iam_contracts
```

**环境变量**:
```bash
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=mypassword
export DB_NAME=iam_contracts
```

## 脚本说明

### init.sql

创建数据库和所有表结构，包括：

**用户中心 (UC)**:
- users - 用户表
- children - 儿童表
- guardianships - 监护关系表

**认证中心 (AuthN)**:
- accounts - 账户表
- sessions - 会话表
- signing_keys - 签名密钥表
- token_blacklist - Token黑名单表

**授权中心 (AuthZ)**:
- resources - 资源表
- roles - 角色表
- user_roles - 用户角色关联表
- role_resources - 角色资源关联表
- casbin_rule - Casbin策略规则表

**系统表**:
- tenants - 租户表
- system_configs - 系统配置表
- operation_logs - 操作日志表

### seed.sql

加载初始数据和测试数据：

1. **租户数据**: 系统租户 + 演示租户
2. **管理员用户**: 超级管理员 + 租户管理员
3. **系统角色**: SUPER_ADMIN, ADMIN, USER, GUARDIAN
4. **资源定义**: 25+ API 端点资源
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

### init-db.sh

功能完善的初始化脚本，提供：

- ✅ 连接测试
- ✅ 交互式确认
- ✅ 彩色输出
- ✅ 错误处理
- ✅ 灵活配置

**选项**:

```
-h, --help              显示帮助信息
-H, --host HOST         数据库主机 (默认: 127.0.0.1)
-P, --port PORT         数据库端口 (默认: 3306)
-u, --user USER         数据库用户 (默认: root)
-p, --password PASS     数据库密码
-d, --database DB       数据库名称 (默认: iam_contracts)
--schema-only           仅创建表结构
--seed-only             仅加载种子数据
--skip-confirm          跳过确认提示
```

### reset-db.sh

⚠️ **危险操作**: 完全删除数据库及所有数据！

此脚本用于开发环境快速重置，包含：

1. 删除现有数据库
2. 重新执行初始化脚本
3. 双重确认机制

**安全措施**:
- 需要输入数据库名称确认
- 需要输入 "yes" 进行二次确认
- 红色警告提示
- 仅建议在开发环境使用

## Docker 环境

如果使用 Docker 运行 MySQL：

```bash
# 启动 MySQL 容器
make docker-mysql-up

# 初始化数据库
make db-init DB_PASSWORD=root

# 停止 MySQL 容器
make docker-mysql-down

# 清理 MySQL 数据
make docker-mysql-clean

# 查看 MySQL 日志
make docker-mysql-logs
```

## 数据验证

初始化完成后，可以执行以下查询验证：

```sql
-- 查看所有表
SHOW TABLES;

-- 查看租户数据
SELECT id, name, code, status FROM tenants;

-- 查看用户数据
SELECT u.id, u.name, u.phone, t.name as tenant_name
FROM users u
JOIN tenants t ON u.tenant_id = t.id;

-- 查看角色分配
SELECT 
    u.name as user_name,
    r.name as role_name,
    ur.granted_at
FROM user_roles ur
JOIN users u ON ur.user_id = u.id
JOIN roles r ON ur.role_id = r.id;

-- 查看资源统计
SELECT resource_type, COUNT(*) as count
FROM resources
GROUP BY resource_type;

-- 查看监护关系
SELECT 
    c.name as child_name,
    u.name as guardian_name,
    g.relation
FROM guardianships g
JOIN children c ON g.child_id = c.id
JOIN users u ON g.user_id = u.id
WHERE g.revoked_at IS NULL;
```

## 故障排除

### 连接失败

```bash
# 检查 MySQL 服务
sudo systemctl status mysql      # Linux
brew services list               # macOS

# 启动 MySQL
sudo systemctl start mysql       # Linux
brew services start mysql        # macOS
```

### 权限不足

```bash
# 授予权限
mysql -u root -p -e "GRANT ALL PRIVILEGES ON iam_contracts.* TO 'myuser'@'localhost';"
mysql -u root -p -e "FLUSH PRIVILEGES;"
```

### 脚本执行权限

```bash
# 添加执行权限
chmod +x *.sh

# 或直接使用 bash 执行
bash init-db.sh
```

## 详细文档

完整的数据库初始化文档请参考：

📖 [数据库初始化指南](../../docs/DATABASE_INITIALIZATION.md)

包含：
- 详细的数据库结构说明
- ER 图和关系模型
- 完整的操作指南
- 故障排除方案
- 最佳实践建议

## 相关资源

- [架构概览](../../docs/architecture-overview.md)
- [Makefile 指南](../../docs/MAKEFILE_GUIDE.md)
- [UC 模块文档](../../docs/uc/README.md)
- [AuthN 模块文档](../../docs/authn/README.md)
- [AuthZ 模块文档](../../docs/authz/README.md)

---

**版本**: 1.0.0  
**最后更新**: 2025-10-18  
**维护者**: IAM Contracts Team
