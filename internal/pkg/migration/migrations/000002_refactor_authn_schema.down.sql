-- ============================================================================
-- IAM Contracts - Database Schema Migration Rollback
-- Version: 2
-- Description: Rollback Authn module schema refactoring
-- Date: 2025-11-05
-- ============================================================================
-- 回滚认证模块表结构重构，恢复到 v1 的表结构
-- ============================================================================

-- ============================================================================
-- Step 1: 删除新表
-- ============================================================================

DROP TABLE IF EXISTS `iam_authn_token_audit`;
DROP TABLE IF EXISTS `iam_authn_credentials`;
DROP TABLE IF EXISTS `iam_authn_accounts`;

-- ============================================================================
-- Step 2: 恢复旧表结构（来自 000001_init_schema.up.sql）
-- ============================================================================

-- 2.1 认证账号表

CREATE TABLE IF NOT EXISTS `iam_auth_accounts`
(
    `id`          BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '账号ID',
    `user_id`     BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `provider`    VARCHAR(32)     NOT NULL COMMENT '认证提供者: wechat/operation/...',
    `external_id` VARCHAR(128)    NOT NULL COMMENT '外部ID (如微信OpenID)',
    `app_id`      VARCHAR(64)              DEFAULT NULL COMMENT '应用ID (如微信AppID)',
    `status`      TINYINT         NOT NULL DEFAULT 1 COMMENT '账号状态: 1-正常, 2-禁用',
    `created_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`  DATETIME                 DEFAULT NULL COMMENT '删除时间',
    `created_by`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`     INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    KEY `idx_user_provider` (`user_id`, `provider`),
    UNIQUE KEY `uk_iam_auth_accounts_provider_app_external` (`provider`, `app_id`, `external_id`),
    KEY `idx_iam_auth_accounts_deleted_at` (`deleted_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='认证账号表';

-- 2.2 微信账号扩展信息表

CREATE TABLE IF NOT EXISTS `iam_auth_wechat_accounts`
(
    `id`         BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '记录ID',
    `account_id` BIGINT UNSIGNED NOT NULL COMMENT '账号ID (关联 iam_auth_accounts.id)',
    `app_id`     VARCHAR(64)     NOT NULL COMMENT '微信应用ID',
    `open_id`    VARCHAR(128)    NOT NULL COMMENT '微信OpenID',
    `union_id`   VARCHAR(128)             DEFAULT NULL COMMENT '微信UnionID',
    `nickname`   VARCHAR(128)             DEFAULT NULL COMMENT '微信昵称',
    `avatar_url` VARCHAR(256)             DEFAULT NULL COMMENT '头像URL',
    `meta`       JSON                     DEFAULT NULL COMMENT '扩展元数据 (JSON格式)',
    `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME                 DEFAULT NULL COMMENT '删除时间',
    `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`    INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    UNIQUE KEY `uk_account_id` (`account_id`),
    KEY `idx_app_open` (`app_id`, `open_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='微信账号扩展信息表';

-- 2.3 运营后台账号凭证表

CREATE TABLE IF NOT EXISTS `iam_auth_operation_accounts`
(
    `id`              BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '记录ID',
    `account_id`      BIGINT UNSIGNED NOT NULL COMMENT '账号ID (关联 iam_auth_accounts.id)',
    `username`        VARCHAR(64)     NOT NULL COMMENT '用户名',
    `password_hash`   VARBINARY(255)  NOT NULL COMMENT '密码哈希值',
    `algo`            VARCHAR(32)     NOT NULL COMMENT '密码哈希算法',
    `params`          VARBINARY(512)           DEFAULT NULL COMMENT '哈希算法参数',
    `failed_attempts` INT             NOT NULL DEFAULT 0 COMMENT '失败登录尝试次数',
    `locked_until`    DATETIME                 DEFAULT NULL COMMENT '锁定截止时间',
    `last_changed_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '最后修改密码时间',
    `created_at`      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`      DATETIME                 DEFAULT NULL COMMENT '删除时间',
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`         INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    UNIQUE KEY `uk_account_id` (`account_id`),
    UNIQUE KEY `uk_username` (`username`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='运营后台账号凭证表';

-- 2.4 会话表

CREATE TABLE IF NOT EXISTS `iam_auth_sessions`
(
    `id`              BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '会话ID (Snowflake)',
    `user_id`         BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `refresh_token`   VARCHAR(255)    NOT NULL COMMENT 'Refresh Token',
    `access_token_id` VARCHAR(64)              DEFAULT NULL COMMENT '当前 Access Token ID (jti)',
    `device_id`       VARCHAR(100)             DEFAULT NULL COMMENT '设备ID',
    `device_type`     VARCHAR(32)              DEFAULT NULL COMMENT '设备类型 (ios/android/web/desktop)',
    `ip_address`      VARCHAR(45)              DEFAULT NULL COMMENT 'IP地址',
    `user_agent`      VARCHAR(500)             DEFAULT NULL COMMENT '浏览器UA',
    `expires_at`      TIMESTAMP       NOT NULL COMMENT '过期时间',
    `last_active_at`  TIMESTAMP       NULL     DEFAULT NULL COMMENT '最后活跃时间',
    `created_at`      TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE KEY `uk_refresh_token` (`refresh_token`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_expires_at` (`expires_at`),
    KEY `idx_device_id` (`device_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='会话表';

-- 2.5 Token 黑名单表

CREATE TABLE IF NOT EXISTS `iam_auth_token_blacklist`
(
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `token_id`   VARCHAR(64)     NOT NULL COMMENT 'Token ID (jti)',
    `user_id`    BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `token_type` VARCHAR(20)     NOT NULL DEFAULT 'access' COMMENT 'Token类型 (access/refresh)',
    `reason`     VARCHAR(100)             DEFAULT NULL COMMENT '原因 (logout/password_change/admin_revoke)',
    `expires_at` TIMESTAMP       NOT NULL COMMENT '原过期时间',
    `created_at` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_token_id` (`token_id`),
    KEY `idx_user_id` (`user_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='Token 黑名单表';

-- ============================================================================
-- Step 3: 更新 Schema 版本（回滚到 v2.0）
-- ============================================================================

DELETE FROM `iam_schema_version` WHERE `version` = '2.1';

-- ============================================================================
-- Rollback Complete
-- ============================================================================
-- 注意事项：
-- 1. 回滚会删除新表中的所有数据
-- 2. 如有数据需要迁移回旧表，请在执行前添加数据迁移脚本
-- 3. 生产环境执行前请务必备份数据库
-- ============================================================================
