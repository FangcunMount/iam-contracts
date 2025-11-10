-- ============================================================================
-- IAM Contracts - Database Schema Migration
-- Version: 3
-- Description: Fix Authn table names to match PO layer definitions
-- Date: 2025-11-10
-- ============================================================================
-- 修复表名不一致问题：
-- 1. PO 层定义使用 iam_auth_* 前缀
-- 2. 但 000002 迁移创建的是 iam_authn_* 前缀
-- 3. 统一改回 iam_auth_* 前缀以匹配代码
-- ============================================================================

-- ============================================================================
-- Step 1: 备份数据（如有）
-- ============================================================================
-- 生产环境需要先备份数据

-- ============================================================================
-- Step 2: 重命名账户表
-- ============================================================================

-- 检查并重命名账户表
DROP TABLE IF EXISTS `iam_auth_accounts`;
RENAME TABLE `iam_authn_accounts` TO `iam_auth_accounts`;

-- ============================================================================
-- Step 3: 重命名凭据表
-- ============================================================================

-- 检查并重命名凭据表
DROP TABLE IF EXISTS `iam_auth_credentials`;
RENAME TABLE `iam_authn_credentials` TO `iam_auth_credentials`;

-- ============================================================================
-- Step 4: 重命名 Token 审计表
-- ============================================================================

-- 检查并重命名 Token 审计表
DROP TABLE IF EXISTS `iam_auth_token_audit`;
RENAME TABLE `iam_authn_token_audit` TO `iam_auth_token_audit`;

-- ============================================================================
-- Step 5: 删除旧的索引并重建（表名变更后需要重建）
-- ============================================================================

-- 账户表索引已随表重命名自动更新，无需手动处理
-- 凭据表索引已随表重命名自动更新，无需手动处理

-- ============================================================================
-- Step 6: 创建 IDP 模块表（微信应用管理）
-- ============================================================================

CREATE TABLE IF NOT EXISTS `iam_idp_wechat_apps`
(
    `id`                     BIGINT UNSIGNED NOT NULL COMMENT '应用ID（Snowflake）',
    `app_id`                 VARCHAR(64)     NOT NULL COMMENT '微信应用ID (AppID)',
    `name`                   VARCHAR(255)    NOT NULL COMMENT '应用名称',
    `type`                   VARCHAR(32)     NOT NULL COMMENT '应用类型: MiniProgram|MP|WebApp|OpenPlatform',
    `status`                 VARCHAR(32)     NOT NULL DEFAULT 'Enabled' COMMENT '应用状态: Enabled|Disabled',
    
    -- 认证凭据（加密存储）
    `auth_secret_cipher`     BLOB                     DEFAULT NULL COMMENT 'AppSecret 加密值',
    `auth_secret_fp`         VARCHAR(128)             DEFAULT NULL COMMENT 'AppSecret 指纹（用于验证）',
    `auth_secret_version`    INT             NOT NULL DEFAULT 0 COMMENT 'AppSecret 版本号',
    `auth_secret_rotated_at` DATETIME                 DEFAULT NULL COMMENT 'AppSecret 最后轮换时间',
    
    -- 消息加解密凭据（加密存储）
    `msg_callback_token`     VARCHAR(128)             DEFAULT NULL COMMENT '消息回调 Token',
    `msg_aes_key_cipher`     BLOB                     DEFAULT NULL COMMENT 'EncodingAESKey 加密值',
    `msg_secret_version`     INT             NOT NULL DEFAULT 0 COMMENT '消息密钥版本号',
    `msg_secret_rotated_at`  DATETIME                 DEFAULT NULL COMMENT '消息密钥最后轮换时间',
    
    -- 审计字段
    `created_at`             DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`             DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`             DATETIME                 DEFAULT NULL COMMENT '删除时间（软删除）',
    `created_by`             BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`             BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`             BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`                INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_app_id` (`app_id`),
    KEY `idx_type` (`type`),
    KEY `idx_status` (`status`),
    KEY `idx_created_at` (`created_at`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci
    COMMENT ='IDP 微信应用配置表';

-- ============================================================================
-- Step 7: 更新 Schema 版本
-- ============================================================================

INSERT INTO `iam_schema_version` (`version`, `description`)
VALUES ('3.0', '2025-11-10 - Fix Authn table names and add IDP wechat apps table')
ON DUPLICATE KEY UPDATE
    `description` = VALUES(`description`),
    `applied_at` = CURRENT_TIMESTAMP;

-- ============================================================================
-- Migration Complete
-- ============================================================================
-- 变更说明：
-- 1. 将 iam_authn_* 表重命名为 iam_auth_* 以匹配 PO 层代码
-- 2. 新增 iam_idp_wechat_apps 表用于微信应用配置管理
-- 3. 凭据采用加密存储，支持密钥轮换
-- ============================================================================
