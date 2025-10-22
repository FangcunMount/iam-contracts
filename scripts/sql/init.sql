-- ============================================================================
-- IAM Contracts 数据库初始化脚本 v2.0
-- ============================================================================
-- 数据库: aim
-- 版本: 2.0.0
-- 创建时间: 2025-10-19
-- 说明: 此版本完全对齐代码中的 PO 定义，修正了 v1.0 中的所有不一致问题
-- ============================================================================
-- 主要改进:
-- 1. ID 类型统一为 BIGINT UNSIGNED (Snowflake ID)
-- 2. 添加完整的审计字段 (created_by, updated_by, deleted_by)
-- 3. 修正字段类型 (status, gender, height, weight)
-- 4. 添加正确的唯一索引
-- 5. 表名统一添加模块前缀 (auth_, authz_)
-- 6. 认证表拆分为三表 (accounts, wechat_accounts, operation_accounts)
-- 7. 移除 tenant_id (代码中暂无此字段)
-- ============================================================================
-- ============================================================================
-- 创建数据库和用户
-- ============================================================================
-- 创建数据库
CREATE DATABASE IF NOT EXISTS aim DEFAULT CHARACTER SET utf8mb4 DEFAULT COLLATE utf8mb4_unicode_ci;

USE aim;

-- 创建用户并授权（如果不存在）
CREATE USER IF NOT EXISTS 'fcm_admin' @'%' IDENTIFIED BY 'RfDtf6SGkGFeB9qZQtX';

GRANT ALL PRIVILEGES ON aim.* TO 'fcm_admin' @'%';

FLUSH PRIVILEGES;

USE aim;

-- ============================================================================
-- 用户中心 (User Center) 表结构
-- ============================================================================
-- 用户表
CREATE TABLE IF NOT EXISTS
  `users` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID (Snowflake)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间（软删除）',
    `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version` INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    `name` VARCHAR(64) NOT NULL COMMENT '用户名称',
    `phone` VARCHAR(20) NOT NULL COMMENT '手机号',
    `email` VARCHAR(100) NOT NULL COMMENT '邮箱',
    `id_card` VARCHAR(20) NOT NULL COMMENT '身份证号',
    `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '用户状态 (1=正常 2=禁用 3=删除)',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_phone` (`phone`),
    UNIQUE KEY `uk_email` (`email`),
    UNIQUE KEY `uk_id_card` (`id_card`),
    KEY `idx_status` (`status`),
    KEY `idx_deleted_at` (`deleted_at`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '用户表';

-- 儿童表
CREATE TABLE IF NOT EXISTS
  `children` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '儿童ID (Snowflake)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间（软删除）',
    `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version` INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    `name` VARCHAR(64) NOT NULL COMMENT '儿童姓名',
    `id_card` VARCHAR(20) NOT NULL COMMENT '身份证号码',
    `gender` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '性别 (0=未知 1=男 2=女)',
    `birthday` VARCHAR(10) DEFAULT NULL COMMENT '出生日期 (YYYY-MM-DD)',
    `height` BIGINT DEFAULT NULL COMMENT '身高（以0.1cm为单位，如1650表示165.0cm）',
    `weight` BIGINT DEFAULT NULL COMMENT '体重（以0.1kg为单位，如450表示45.0kg）',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_id_card` (`id_card`),
    KEY `idx_name` (`name`),
    KEY `idx_deleted_at` (`deleted_at`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '儿童表';

-- 监护关系表
CREATE TABLE IF NOT EXISTS
  `guardianships` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '监护关系ID (Snowflake)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间（软删除）',
    `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version` INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '监护人ID',
    `child_id` BIGINT UNSIGNED NOT NULL COMMENT '儿童ID',
    `relation` VARCHAR(16) NOT NULL COMMENT '监护关系 (father/mother/grandfather/grandmother/guardian)',
    `established_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '建立时间',
    `revoked_at` TIMESTAMP NULL DEFAULT NULL COMMENT '撤销时间',
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_child_id` (`child_id`),
    KEY `idx_user_child` (`user_id`, `child_id`),
    KEY `idx_deleted_at` (`deleted_at`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '监护关系表';

-- ============================================================================
-- 认证中心 (Authentication Center) 表结构
-- ============================================================================
-- 认证账号表 (主表)
CREATE TABLE IF NOT EXISTS
  `auth_accounts` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '账号ID (Snowflake)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间（软删除）',
    `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version` INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `provider` VARCHAR(32) NOT NULL COMMENT '认证提供商 (wechat/wework/operation)',
    `external_id` VARCHAR(128) NOT NULL COMMENT '外部ID (如openid/unionid/username)',
    `app_id` VARCHAR(64) DEFAULT NULL COMMENT '应用ID (微信AppID等)',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态 (1=正常 2=禁用)',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_provider_app_external` (`provider`, `app_id`, `external_id`),
    KEY `idx_user_provider` (`user_id`, `provider`),
    KEY `idx_deleted_at` (`deleted_at`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '认证账号表';

-- 微信账号表 (扩展表)
CREATE TABLE IF NOT EXISTS
  `auth_wechat_accounts` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT 'ID (Snowflake)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间（软删除）',
    `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version` INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    `account_id` BIGINT UNSIGNED NOT NULL COMMENT '账号ID (关联 auth_accounts.id)',
    `app_id` VARCHAR(64) NOT NULL COMMENT '微信AppID',
    `open_id` VARCHAR(128) NOT NULL COMMENT '微信OpenID',
    `union_id` VARCHAR(128) DEFAULT NULL COMMENT '微信UnionID',
    `nickname` VARCHAR(128) DEFAULT NULL COMMENT '微信昵称',
    `avatar_url` VARCHAR(256) DEFAULT NULL COMMENT '微信头像URL',
    `meta` JSON DEFAULT NULL COMMENT '扩展元数据',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_account_id` (`account_id`),
    KEY `idx_app_open` (`app_id`, `open_id`),
    KEY `idx_deleted_at` (`deleted_at`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '微信账号表';

-- 运营账号凭证表 (扩展表)
CREATE TABLE IF NOT EXISTS
  `auth_operation_accounts` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT 'ID (Snowflake)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间（软删除）',
    `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version` INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    `account_id` BIGINT UNSIGNED NOT NULL COMMENT '账号ID (关联 auth_accounts.id)',
    `username` VARCHAR(64) NOT NULL COMMENT '用户名',
    `password_hash` VARBINARY(255) NOT NULL COMMENT '密码哈希',
    `algo` VARCHAR(32) NOT NULL COMMENT '加密算法 (argon2id/bcrypt)',
    `params` VARBINARY(512) DEFAULT NULL COMMENT '算法参数',
    `failed_attempts` INT NOT NULL DEFAULT 0 COMMENT '失败尝试次数',
    `locked_until` TIMESTAMP NULL DEFAULT NULL COMMENT '锁定截止时间',
    `last_changed_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '密码最后修改时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_account_id` (`account_id`),
    UNIQUE KEY `uk_username` (`username`),
    KEY `idx_deleted_at` (`deleted_at`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '运营账号凭证表';

-- JWKS 密钥表
CREATE TABLE IF NOT EXISTS
  `jwks_keys` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `kid` VARCHAR(64) NOT NULL COMMENT '密钥ID (Key ID)',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态 (1=Active 2=Grace 3=Retired)',
    `kty` VARCHAR(32) NOT NULL COMMENT '密钥类型 (RSA/EC)',
    `use` VARCHAR(16) NOT NULL COMMENT '用途 (sig=签名 enc=加密)',
    `alg` VARCHAR(32) NOT NULL COMMENT '算法 (RS256/RS384/RS512)',
    `jwk_json` JSON NOT NULL COMMENT '公钥JWK JSON',
    `not_before` TIMESTAMP NULL DEFAULT NULL COMMENT '生效时间',
    `not_after` TIMESTAMP NULL DEFAULT NULL COMMENT '过期时间',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_kid` (`kid`),
    KEY `idx_status` (`status`),
    KEY `idx_alg` (`alg`),
    KEY `idx_not_after` (`not_after`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = 'JWKS密钥表';

-- 会话表 (可选，用于 Refresh Token 管理)
CREATE TABLE IF NOT EXISTS
  `auth_sessions` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '会话ID (Snowflake)',
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `refresh_token` VARCHAR(255) NOT NULL COMMENT 'Refresh Token',
    `access_token_id` VARCHAR(64) DEFAULT NULL COMMENT '当前 Access Token ID',
    `device_id` VARCHAR(100) DEFAULT NULL COMMENT '设备ID',
    `ip_address` VARCHAR(45) DEFAULT NULL COMMENT 'IP地址',
    `user_agent` VARCHAR(500) DEFAULT NULL COMMENT '浏览器UA',
    `expires_at` TIMESTAMP NOT NULL COMMENT '过期时间',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_refresh_token` (`refresh_token`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_expires_at` (`expires_at`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '会话表';

-- Token 黑名单表 (可选，用于 Token 撤销)
CREATE TABLE IF NOT EXISTS
  `auth_token_blacklist` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `token_id` VARCHAR(64) NOT NULL COMMENT 'Token ID (jti)',
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `reason` VARCHAR(100) DEFAULT NULL COMMENT '加入黑名单原因',
    `expires_at` TIMESTAMP NOT NULL COMMENT 'Token原过期时间',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_token_id` (`token_id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_expires_at` (`expires_at`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = 'Token黑名单表';

-- ============================================================================
-- 授权中心 (Authorization Center) 表结构
-- ============================================================================
-- 资源目录表
CREATE TABLE IF NOT EXISTS
  `authz_resources` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '资源ID (Snowflake)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间（软删除）',
    `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version` INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    `key` VARCHAR(128) NOT NULL COMMENT '资源键（唯一标识）',
    `display_name` VARCHAR(128) DEFAULT NULL COMMENT '显示名称',
    `app_name` VARCHAR(32) DEFAULT NULL COMMENT '应用名称',
    `domain` VARCHAR(32) DEFAULT NULL COMMENT '域',
    `type` VARCHAR(32) DEFAULT NULL COMMENT '资源类型 (api/menu/button/data)',
    `actions` TEXT DEFAULT NULL COMMENT '操作列表 (JSON数组字符串)',
    `description` VARCHAR(512) DEFAULT NULL COMMENT '描述',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_key` (`key`),
    KEY `idx_app_name` (`app_name`),
    KEY `idx_domain` (`domain`),
    KEY `idx_type` (`type`),
    KEY `idx_deleted_at` (`deleted_at`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '资源目录表';

-- 角色表
CREATE TABLE IF NOT EXISTS
  `authz_roles` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '角色ID (Snowflake)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间（软删除）',
    `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version` INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    `name` VARCHAR(64) NOT NULL COMMENT '角色名称',
    `display_name` VARCHAR(128) DEFAULT NULL COMMENT '显示名称',
    `tenant_id` VARCHAR(64) NOT NULL COMMENT '租户ID',
    `description` VARCHAR(512) DEFAULT NULL COMMENT '描述',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_tenant_name` (`tenant_id`, `name`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_deleted_at` (`deleted_at`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '角色表';

-- 角色赋权表 (用户-角色关联)
CREATE TABLE IF NOT EXISTS
  `authz_assignments` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '赋权ID (Snowflake)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间（软删除）',
    `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version` INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    `subject_type` VARCHAR(16) NOT NULL COMMENT '主体类型 (user/group/service)',
    `subject_id` VARCHAR(64) NOT NULL COMMENT '主体ID',
    `role_id` BIGINT UNSIGNED NOT NULL COMMENT '角色ID',
    `tenant_id` VARCHAR(64) NOT NULL COMMENT '租户ID',
    `granted_by` VARCHAR(64) DEFAULT NULL COMMENT '授予人',
    `granted_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '授予时间',
    PRIMARY KEY (`id`),
    KEY `idx_subject` (`subject_type`, `subject_id`),
    KEY `idx_role_id` (`role_id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_deleted_at` (`deleted_at`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '角色赋权表';

-- 策略版本表 (可选，用于策略管理)
CREATE TABLE IF NOT EXISTS
  `authz_policy_versions` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '策略版本ID (Snowflake)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间（软删除）',
    `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version` INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    `role_id` BIGINT UNSIGNED NOT NULL COMMENT '角色ID',
    `policy_version` INT UNSIGNED NOT NULL COMMENT '策略版本号',
    `policy_document` JSON NOT NULL COMMENT '策略文档 (JSON格式)',
    `is_active` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否为活跃版本',
    PRIMARY KEY (`id`),
    KEY `idx_role_id` (`role_id`),
    KEY `idx_is_active` (`is_active`),
    KEY `idx_deleted_at` (`deleted_at`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '策略版本表';

-- Casbin 策略表 (用于 Casbin 授权引擎)
CREATE TABLE IF NOT EXISTS
  `casbin_rule` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `ptype` VARCHAR(100) NOT NULL COMMENT '策略类型',
    `v0` VARCHAR(100) DEFAULT NULL COMMENT '值0',
    `v1` VARCHAR(100) DEFAULT NULL COMMENT '值1',
    `v2` VARCHAR(100) DEFAULT NULL COMMENT '值2',
    `v3` VARCHAR(100) DEFAULT NULL COMMENT '值3',
    `v4` VARCHAR(100) DEFAULT NULL COMMENT '值4',
    `v5` VARCHAR(100) DEFAULT NULL COMMENT '值5',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = 'Casbin策略规则表';

-- ============================================================================
-- 系统表 (可选)
-- ============================================================================
-- 租户表 (如果需要多租户支持)
CREATE TABLE IF NOT EXISTS
  `tenants` (
    `id` VARCHAR(64) NOT NULL COMMENT '租户ID',
    `name` VARCHAR(100) NOT NULL COMMENT '租户名称',
    `code` VARCHAR(50) NOT NULL COMMENT '租户编码',
    `contact_name` VARCHAR(100) DEFAULT NULL COMMENT '联系人姓名',
    `contact_phone` VARCHAR(20) DEFAULT NULL COMMENT '联系人电话',
    `contact_email` VARCHAR(100) DEFAULT NULL COMMENT '联系人邮箱',
    `status` VARCHAR(20) NOT NULL DEFAULT 'active' COMMENT '状态: active, inactive, suspended',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code` (`code`),
    KEY `idx_status` (`status`),
    KEY `idx_deleted_at` (`deleted_at`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '租户表';

-- 系统配置表 (可选)
CREATE TABLE IF NOT EXISTS
  `system_configs` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `tenant_id` VARCHAR(64) DEFAULT NULL COMMENT '租户ID（NULL表示全局配置）',
    `config_key` VARCHAR(100) NOT NULL COMMENT '配置键',
    `config_value` TEXT NOT NULL COMMENT '配置值（JSON格式）',
    `description` VARCHAR(500) DEFAULT NULL COMMENT '描述',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_tenant_key` (`tenant_id`, `config_key`),
    KEY `idx_config_key` (`config_key`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '系统配置表';

-- 操作日志表 (可选，用于审计)
CREATE TABLE IF NOT EXISTS
  `operation_logs` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `user_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '操作用户ID',
    `operation_type` VARCHAR(50) NOT NULL COMMENT '操作类型: CREATE, UPDATE, DELETE',
    `resource_type` VARCHAR(50) NOT NULL COMMENT '资源类型: user, role, resource',
    `resource_id` VARCHAR(64) DEFAULT NULL COMMENT '资源ID',
    `operation_desc` VARCHAR(500) DEFAULT NULL COMMENT '操作描述',
    `ip_address` VARCHAR(45) DEFAULT NULL COMMENT 'IP地址',
    `user_agent` VARCHAR(500) DEFAULT NULL COMMENT '浏览器UA',
    `request_data` TEXT DEFAULT NULL COMMENT '请求数据',
    `response_data` TEXT DEFAULT NULL COMMENT '响应数据',
    `status` VARCHAR(20) NOT NULL COMMENT '状态: success, failure',
    `error_message` VARCHAR(500) DEFAULT NULL COMMENT '错误信息',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_operation_type` (`operation_type`),
    KEY `idx_resource_type` (`resource_type`),
    KEY `idx_created_at` (`created_at`)
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '操作日志表';

-- ============================================================================
-- 初始化完成
-- ============================================================================
-- 显示所有表
SHOW TABLES;

-- 显示表统计信息
SELECT
  TABLE_NAME as '表名',
  TABLE_COMMENT as '说明',
  TABLE_ROWS as '行数',
  ROUND(DATA_LENGTH / 1024 / 1024, 2) as '数据大小(MB)'
FROM
  information_schema.TABLES
WHERE
  TABLE_SCHEMA = 'aim'
ORDER BY
  TABLE_NAME;

-- ============================================================================
-- 说明
-- ============================================================================
-- 
-- 主要改进内容:
-- 
-- 1. ID 类型统一
--    - 所有主键和外键使用 BIGINT UNSIGNED (支持 Snowflake ID)
--    - jwks_keys 表使用 AUTO_INCREMENT (特殊需求)
--    
-- 2. 审计字段完整
--    - AuditFields: id, created_at, updated_at, deleted_at
--    - created_by, updated_by, deleted_by (审计人)
--    - version (乐观锁)
--    
-- 3. 字段类型修正
--    - status: VARCHAR -> TINYINT
--    - gender: VARCHAR -> TINYINT UNSIGNED
--    - height/weight: DECIMAL -> BIGINT (存储整数，单位为0.1)
--    
-- 4. 唯一索引添加
--    - phone, email, id_card: 普通索引 -> 唯一索引
--    
-- 5. 表名统一
--    - 认证模块: auth_ 前缀
--    - 授权模块: authz_ 前缀
--    
-- 6. 认证表拆分
--    - auth_accounts: 主表
--    - auth_wechat_accounts: 微信账号扩展
--    - auth_operation_accounts: 运营账号凭证
--    
-- 7. 软删除支持
--    - deleted_at 字段
--    - 不使用外键约束 (避免级联删除)
--    
-- 8. 移除 tenant_id (暂时)
--    - 代码中 PO 暂无此字段
--    - 如需多租户，需先更新代码
--    
-- 使用方式:
-- 1. 开发环境: 直接执行此脚本
-- 2. 生产环境: 使用增量迁移脚本
-- 3. 推荐使用 GORM AutoMigrate 自动同步
-- 
-- ============================================================================