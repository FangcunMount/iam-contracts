-- ============================================================================
-- IAM Contracts - Database Schema Migration
-- Version: 2
-- Description: Refactor Authn module schema to match DDD design
-- Date: 2025-11-05
-- ============================================================================
-- 重构认证模块表结构，使其与领域驱动设计保持一致
-- 主要变更：
-- 1. 统一 Account 表结构（type, app_id, external_id, unique_id）
-- 2. 统一 Credential 表结构（支持多种凭据类型）
-- 3. 简化 Token 存储（使用 Redis，表仅做审计）
-- ============================================================================

-- ============================================================================
-- Step 1: 备份现有数据（如果需要迁移数据）
-- ============================================================================
-- 在实际生产环境中，需要先备份数据再执行迁移

-- ============================================================================
-- Step 2: 删除旧的 Authn 相关表
-- ============================================================================

DROP TABLE IF EXISTS `iam_auth_wechat_accounts`;
DROP TABLE IF EXISTS `iam_auth_operation_accounts`;
DROP TABLE IF EXISTS `iam_auth_token_blacklist`;
DROP TABLE IF EXISTS `iam_auth_sessions`;
DROP TABLE IF EXISTS `iam_auth_accounts`;

-- ============================================================================
-- Step 3: 创建新的统一账户表（Account）
-- ============================================================================
-- 说明：统一管理所有类型的第三方登录账户
-- 账户类型通过 type 字段区分：wc-minip, wc-offi, wc-com, opera
-- ============================================================================

CREATE TABLE IF NOT EXISTS `iam_authn_accounts`
(
    `id`          BIGINT UNSIGNED NOT NULL COMMENT '账户ID（Snowflake）',
    `user_id`     BIGINT UNSIGNED NOT NULL COMMENT '关联用户ID',
    `type`        VARCHAR(16)     NOT NULL COMMENT '账户类型: wc-minip|wc-offi|wc-com|opera',
    `app_id`      VARCHAR(64)     NOT NULL DEFAULT '' COMMENT '应用ID: 微信appid|企业微信corpid|运营后台为空',
    `external_id` VARCHAR(128)    NOT NULL COMMENT '外部平台用户标识: openid|userid|username',
    `unique_id`   VARCHAR(128)    NOT NULL DEFAULT '' COMMENT '全局唯一标识: unionid|运营后台为空',
    `profile`     JSON                     DEFAULT NULL COMMENT '用户资料: 昵称、头像等（JSON格式）',
    `meta`        JSON                     DEFAULT NULL COMMENT '额外元数据（JSON格式）',
    `status`      TINYINT         NOT NULL DEFAULT 1 COMMENT '账户状态: 0-禁用, 1-激活, 2-归档, 3-删除',
    `created_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`  DATETIME                 DEFAULT NULL COMMENT '删除时间（软删除）',
    `created_by`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`     INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_type_app_external` (`type`, `app_id`, `external_id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_unique_id` (`unique_id`),
    KEY `idx_status` (`status`),
    KEY `idx_deleted_at` (`deleted_at`),
    KEY `idx_created_at` (`created_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci
    COMMENT ='认证账户表 - 统一管理所有类型的第三方登录账户';

-- ============================================================================
-- Step 4: 创建新的统一凭据表（Credential）
-- ============================================================================
-- 说明：统一管理所有类型的认证凭据
-- 凭据类型：password, phone_otp, oauth_wx_minip, oauth_wecom
-- ============================================================================

CREATE TABLE IF NOT EXISTS `iam_authn_credentials`
(
    `id`               BIGINT          NOT NULL AUTO_INCREMENT COMMENT '凭据ID',
    `account_id`       BIGINT UNSIGNED NOT NULL COMMENT '关联账户ID',
    `type`             VARCHAR(32)     NOT NULL COMMENT '凭据类型: password|phone_otp|oauth_wx_minip|oauth_wecom',
    `idp`              VARCHAR(32)              DEFAULT NULL COMMENT 'IDP类型: wechat|wecom|phone|NULL(本地)',
    `idp_identifier`   VARCHAR(256)    NOT NULL DEFAULT '' COMMENT 'IDP标识符: unionid|openid@appid|userid|+E164|空',
    `app_id`           VARCHAR(64)              DEFAULT NULL COMMENT '应用ID: wechat=appid|wecom=corpid|NULL(本地)',
    `material`         VARBINARY(512)           DEFAULT NULL COMMENT '凭据材料: PHC哈希（password）|NULL(OAuth/OTP)',
    `algo`             VARCHAR(32)              DEFAULT NULL COMMENT '算法: argon2id|bcrypt|NULL(OAuth/OTP)',
    `params_json`      VARBINARY(1024)          DEFAULT NULL COMMENT '参数JSON: 微信profile|企业微信agentid等',
    `status`           TINYINT         NOT NULL DEFAULT 1 COMMENT '凭据状态: 0-禁用, 1-启用',
    `failed_attempts`  INT             NOT NULL DEFAULT 0 COMMENT '失败尝试次数（仅password）',
    `locked_until`     DATETIME                 DEFAULT NULL COMMENT '锁定截止时间（仅password）',
    `last_success_at`  DATETIME                 DEFAULT NULL COMMENT '最近成功时间',
    `last_failure_at`  DATETIME                 DEFAULT NULL COMMENT '最近失败时间',
    `created_at`       DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`       DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`       DATETIME                 DEFAULT NULL COMMENT '删除时间（软删除）',
    `created_by`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`          INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_account_type` (`account_id`, `type`),
    KEY `idx_idp_app_identifier` (`idp`, `app_id`, `idp_identifier`(191)),
    KEY `idx_account_id` (`account_id`),
    KEY `idx_status` (`status`),
    KEY `idx_deleted_at` (`deleted_at`),
    KEY `idx_created_at` (`created_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci
    COMMENT ='认证凭据表 - 统一管理所有类型的认证凭据（三件套：外部身份+本地凭据）';

-- ============================================================================
-- Step 5: 创建 Token 审计表（简化版，主存储在 Redis）
-- ============================================================================
-- 说明：Token 主要存储在 Redis 中，此表仅用于审计和长期追踪
-- ============================================================================

CREATE TABLE IF NOT EXISTS `iam_authn_token_audit`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `token_id`       VARCHAR(64)     NOT NULL COMMENT 'Token ID (jti)',
    `token_type`     VARCHAR(16)     NOT NULL COMMENT 'Token类型: access|refresh',
    `user_id`        BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    `account_id`     BIGINT UNSIGNED NOT NULL COMMENT '账户ID',
    `tenant_id`      BIGINT UNSIGNED          DEFAULT NULL COMMENT '租户ID（可选）',
    `issued_at`      DATETIME        NOT NULL COMMENT '签发时间',
    `expires_at`     DATETIME        NOT NULL COMMENT '过期时间',
    `revoked_at`     DATETIME                 DEFAULT NULL COMMENT '撤销时间',
    `revoke_reason`  VARCHAR(64)              DEFAULT NULL COMMENT '撤销原因: logout|password_change|admin_revoke',
    `ip_address`     VARCHAR(45)              DEFAULT NULL COMMENT 'IP地址',
    `user_agent`     VARCHAR(500)             DEFAULT NULL COMMENT '浏览器UA',
    `device_id`      VARCHAR(100)             DEFAULT NULL COMMENT '设备ID',
    `created_at`     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_token_id` (`token_id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_account_id` (`account_id`),
    KEY `idx_expires_at` (`expires_at`),
    KEY `idx_revoked_at` (`revoked_at`),
    KEY `idx_created_at` (`created_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci
    COMMENT ='Token审计表 - 记录Token签发和撤销历史（主存储在Redis）';

-- ============================================================================
-- Step 6: JWKS 密钥表保持不变
-- ============================================================================
-- iam_jwks_keys 表结构已经符合设计，无需修改

-- ============================================================================
-- Step 7: 创建索引优化
-- ============================================================================
-- 账户表额外索引（用于快速查找）

CREATE INDEX `idx_authn_accounts_type_status` ON `iam_authn_accounts` (`type`, `status`);

-- 凭据表额外索引（用于认证查询）

CREATE INDEX `idx_authn_credentials_type_status` ON `iam_authn_credentials` (`type`, `status`);

-- ============================================================================
-- Step 8: 更新 Schema 版本
-- ============================================================================

INSERT INTO `iam_schema_version` (`version`, `description`)
VALUES ('2.1', '2025-11-05 - Refactor Authn module to DDD design (Account + Credential unified)')
ON DUPLICATE KEY UPDATE
    `description` = VALUES(`description`),
    `applied_at` = CURRENT_TIMESTAMP;

-- ============================================================================
-- Migration Complete
-- ============================================================================
-- 注意事项：
-- 1. 本迁移会删除旧表中的所有数据，生产环境请先备份
-- 2. 如需数据迁移，请在 Step 1 和 Step 3 之间添加数据迁移脚本
-- 3. Token 数据已迁移到 Redis，表仅做审计用途
-- 4. 新的统一表结构支持所有类型的账户和凭据
-- ============================================================================
