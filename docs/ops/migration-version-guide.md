# Migration 版本管理指南

## 核心概念

**版本是通过文件名管理的，完全手动创建！**

- ✅ 版本号在**文件名**中定义
- ✅ 完全**手动**创建和递增
- ✅ golang-migrate 通过**文件名**识别版本
- ❌ 不是通过代码字段
- ❌ 不是自动生成

## 文件命名规范

```text
格式：{version}_{description}.{direction}.sql

示例：
- 000001_init_schema.up.sql          版本 1 升级脚本
- 000001_init_schema.down.sql        版本 1 回滚脚本
- 000002_add_oauth_tables.up.sql     版本 2 升级脚本
- 000002_add_oauth_tables.down.sql   版本 2 回滚脚本
```

## 实战演示

### 场景 1：添加新表（OAuth 功能）

#### 步骤 1：创建迁移文件（手动）

```bash
cd /Users/yangshujie/workspace/golang/src/github.com/fangcun-mount/iam-contracts

# 创建版本 2 的迁移文件
touch internal/pkg/migration/migrations/000002_add_oauth_tables.up.sql
touch internal/pkg/migration/migrations/000002_add_oauth_tables.down.sql
```

#### 步骤 2：编写升级脚本

```bash
cat > internal/pkg/migration/migrations/000002_add_oauth_tables.up.sql << 'EOF'
-- ==============================================================================
-- Migration Version: 2
-- Description: Add OAuth support tables
-- Author: Your Name
-- Date: 2025-10-31
-- ==============================================================================

-- OAuth 客户端表
CREATE TABLE IF NOT EXISTS `iam_oauth_clients` (
    `id`            BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT 'OAuth 客户端ID',
    `name`          VARCHAR(100)    NOT NULL COMMENT '客户端名称',
    `client_id`     VARCHAR(100)    NOT NULL COMMENT '客户端标识',
    `client_secret` VARCHAR(255)    NOT NULL COMMENT '客户端密钥',
    `redirect_uris` TEXT                     COMMENT '重定向 URI 列表（JSON）',
    `grant_types`   VARCHAR(255)             COMMENT '授权类型',
    `scope`         VARCHAR(255)             COMMENT '权限范围',
    `created_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`    DATETIME                 DEFAULT NULL COMMENT '删除时间',
    `created_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`       INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    UNIQUE KEY `uk_client_id` (`client_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='OAuth 客户端表';

-- OAuth 授权码表
CREATE TABLE IF NOT EXISTS `iam_oauth_authorization_codes` (
    `id`              BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '授权码ID',
    `code`            VARCHAR(255)    NOT NULL COMMENT '授权码',
    `client_id`       BIGINT UNSIGNED NOT NULL COMMENT '客户端ID',
    `user_id`         BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `redirect_uri`    VARCHAR(500)    NOT NULL COMMENT '重定向 URI',
    `scope`           VARCHAR(255)             COMMENT '权限范围',
    `code_challenge`  VARCHAR(255)             COMMENT 'PKCE 挑战码',
    `code_challenge_method` VARCHAR(50)        COMMENT 'PKCE 挑战方法',
    `expires_at`      DATETIME        NOT NULL COMMENT '过期时间',
    `created_at`      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    UNIQUE KEY `uk_code` (`code`),
    KEY `idx_client_user` (`client_id`, `user_id`),
    KEY `idx_expires_at` (`expires_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='OAuth 授权码表';

-- OAuth 访问令牌表
CREATE TABLE IF NOT EXISTS `iam_oauth_access_tokens` (
    `id`            BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '令牌ID',
    `access_token`  VARCHAR(255)    NOT NULL COMMENT '访问令牌',
    `refresh_token` VARCHAR(255)             COMMENT '刷新令牌',
    `client_id`     BIGINT UNSIGNED NOT NULL COMMENT '客户端ID',
    `user_id`       BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `scope`         VARCHAR(255)             COMMENT '权限范围',
    `expires_at`    DATETIME        NOT NULL COMMENT '过期时间',
    `created_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    UNIQUE KEY `uk_access_token` (`access_token`),
    KEY `idx_refresh_token` (`refresh_token`),
    KEY `idx_client_user` (`client_id`, `user_id`),
    KEY `idx_expires_at` (`expires_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='OAuth 访问令牌表';
EOF
```

#### 步骤 3：编写回滚脚本

```bash
cat > internal/pkg/migration/migrations/000002_add_oauth_tables.down.sql << 'EOF'
-- ==============================================================================
-- Migration Rollback Version: 2
-- Description: Remove OAuth support tables
-- ==============================================================================

DROP TABLE IF EXISTS `iam_oauth_access_tokens`;
DROP TABLE IF EXISTS `iam_oauth_authorization_codes`;
DROP TABLE IF EXISTS `iam_oauth_clients`;
EOF
```

#### 步骤 4：验证文件结构

```bash
tree internal/pkg/migration/migrations/

internal/pkg/migration/migrations/
├── 000001_init_schema.down.sql
├── 000001_init_schema.up.sql
├── 000002_add_oauth_tables.down.sql   ← 新增
└── 000002_add_oauth_tables.up.sql     ← 新增
```

#### 步骤 5：重新编译

```bash
go build -o tmp/apiserver ./cmd/apiserver
```

> **重要**：必须重新编译！因为 SQL 文件是通过 `//go:embed` 嵌入到二进制中的。

#### 步骤 6：启动应用（自动迁移）

```bash
./tmp/apiserver --config configs/apiserver.dev.yaml
```

**日志输出**：

```text
[INFO] 🔌 Initializing database connections...
[INFO] ✅ MySQL connected successfully
[INFO] 🔄 Starting database migration...
[INFO] Current version: 1
[INFO] Found new migration: 000002_add_oauth_tables.up.sql
[INFO] Applying migration 000002...
[INFO] ✅ Migration completed successfully (version: 1 -> 2)
```

#### 步骤 7：验证迁移结果

```bash
mysql -u root -p iam_contracts

mysql> SELECT * FROM schema_migrations;
+---------+-------+
| version | dirty |
+---------+-------+
|       2 | false |  ← 版本已更新为 2
+---------+-------+

mysql> SHOW TABLES;
+-------------------------+
| Tables_in_iam_contracts |
+-------------------------+
| iam_users               |
| iam_children            |
| ...                     |
| iam_oauth_clients       | ← 新增
| iam_oauth_authorization_codes | ← 新增
| iam_oauth_access_tokens | ← 新增
| schema_migrations       |
+-------------------------+
```

### 场景 2：修改现有表（添加字段）

#### 步骤 1：创建版本 3（场景 2）

```bash
touch internal/pkg/migration/migrations/000003_add_user_profile.up.sql
touch internal/pkg/migration/migrations/000003_add_user_profile.down.sql
```

#### 步骤 2：编写升级脚本（场景 2）

```bash
cat > internal/pkg/migration/migrations/000003_add_user_profile.up.sql << 'EOF'
-- ==============================================================================
-- Migration Version: 3
-- Description: Add user profile fields (avatar, bio, location)
-- ==============================================================================

ALTER TABLE `iam_users` 
ADD COLUMN `avatar` VARCHAR(255) COMMENT '用户头像 URL' AFTER `email`;

ALTER TABLE `iam_users` 
ADD COLUMN `bio` TEXT COMMENT '用户简介' AFTER `avatar`;

ALTER TABLE `iam_users` 
ADD COLUMN `location` VARCHAR(100) COMMENT '所在地' AFTER `bio`;

-- 添加索引（可选）
CREATE INDEX `idx_location` ON `iam_users`(`location`);
EOF
```

#### 步骤 3：编写回滚脚本（场景 2）

```bash
cat > internal/pkg/migration/migrations/000003_add_user_profile.down.sql << 'EOF'
-- ==============================================================================
-- Migration Rollback Version: 3
-- Description: Remove user profile fields
-- ==============================================================================

DROP INDEX `idx_location` ON `iam_users`;
ALTER TABLE `iam_users` DROP COLUMN `location`;
ALTER TABLE `iam_users` DROP COLUMN `bio`;
ALTER TABLE `iam_users` DROP COLUMN `avatar`;
EOF
```

#### 步骤 4-6：编译、部署、验证

```bash
go build -o tmp/apiserver ./cmd/apiserver
./tmp/apiserver --config configs/apiserver.dev.yaml
```

**结果**：

- 数据库版本：2 -> 3
- `iam_users` 表新增 3 个字段
- 旧数据完全保留 ✅

## 版本号管理最佳实践

### 1. 小项目：顺序编号

```text
000001_init_schema.sql
000002_add_oauth.sql
000003_add_user_profile.sql
000004_add_indexes.sql
```

**优点**：简单直观
**缺点**：团队协作时可能冲突

### 2. 团队协作：时间戳

使用 migrate CLI 自动生成：

```bash
# 安装 CLI
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 生成迁移文件（自动使用时间戳）
cd /Users/yangshujie/workspace/golang/src/github.com/fangcun-mount/iam-contracts
migrate create -ext sql -dir internal/pkg/migration/migrations -seq add_oauth_tables
```

生成文件：

```text
20231031120000_add_oauth_tables.up.sql
20231031120000_add_oauth_tables.down.sql
```

**优点**：避免版本号冲突
**缺点**：文件名较长

### 3. 语义化版本

```text
v1.0.0_init_schema.sql       # 主版本
v1.1.0_add_oauth.sql         # 次版本（新功能）
v1.1.1_fix_user_index.sql    # 修订版（修复）
v1.2.0_add_user_profile.sql  # 次版本（新功能）
```

**优点**：版本含义清晰
**缺点**：需要严格遵守规范

## 重要规则

### ✅ 必须遵守

1. **版本号严格递增**

   ```text
   ✅ 正确：000001, 000002, 000003
   ❌ 错误：000001, 000003, 000002
   ```

2. **已部署的迁移不可修改**

   ```text
   ✅ 正确：创建 000004 修正问题
   ❌ 错误：修改已执行的 000003
   ```

3. **每个 up 必须有对应的 down**

   ```text
   ✅ 正确：
   - 000002_add_oauth.up.sql
   - 000002_add_oauth.down.sql
   
   ❌ 错误：只有 up 没有 down
   ```

4. **回滚脚本必须测试**

```go
// 开发环境测试回滚
if err := migrator.Rollback(); err != nil {
    panic(err) // 版本 2 -> 1
}
if _, _, err := migrator.Run(); err != nil {
    panic(err) // 版本 1 -> 2
}
```

### ⚠️ 特殊情况

1. **数据修正迁移**

   ```sql
   -- up: 修正数据
   UPDATE iam_users SET status = 1 WHERE status IS NULL;
   
   -- down: 无法回滚（数据已修改）
   SELECT 'Warning: Cannot rollback data correction' AS warning;
   ```

2. **不可逆操作**

   ```sql
   -- up: 删除列
   ALTER TABLE iam_users DROP COLUMN old_field;
   
   -- down: 无法恢复数据
   ALTER TABLE iam_users ADD COLUMN old_field VARCHAR(100);
   -- 注意：字段恢复了，但数据丢失了！
   ```

## 快速参考

### 添加新迁移的完整流程

```bash
# 1. 确定版本号（查看现有最大版本）
ls internal/pkg/migration/migrations/
# 输出：000001_init_schema.up.sql
# 新版本：000002

# 2. 创建迁移文件
touch internal/pkg/migration/migrations/000002_add_feature.{up,down}.sql

# 3. 编写 SQL
vim internal/pkg/migration/migrations/000002_add_feature.up.sql
vim internal/pkg/migration/migrations/000002_add_feature.down.sql

# 4. 重新编译（嵌入 SQL 文件）
go build -o tmp/apiserver ./cmd/apiserver

# 5. 启动应用（自动执行迁移）
./tmp/apiserver --config configs/apiserver.dev.yaml

# 6. 验证迁移结果
mysql> SELECT * FROM schema_migrations;

# 7. 提交代码
git add internal/pkg/migration/migrations/000002*
git commit -m "feat: add new feature (migration v2)"
git push
```

## 总结

- ✅ 版本通过**文件名**管理
- ✅ **手动**创建和递增
- ✅ 每次添加新功能 = 创建新版本文件
- ✅ 编译后 SQL 嵌入二进制
- ✅ 启动时自动执行新版本
- ✅ 旧数据完全保留

**记住**：版本号不是代码字段，而是文件名中的数字！
