-- ============================================================================
-- IAM Contracts - Database Schema
-- Version: 1
-- Description: Initial schema for all modules
-- Date: 2025-10-31
-- ============================================================================
-- Notes:
--   * All tables use `iam_` prefix
--   * utf8mb4 / InnoDB
--   * Soft delete fields kept where applicable
--   * Database should be created before running migrations
--   * Migration only contains table DDL, no database creation
-- ============================================================================

-- ============================================================================
-- Module 1: User Center (UC) 用户中心
-- ============================================================================

-- 1.1 用户表

CREATE TABLE IF NOT EXISTS `iam_users`
(
    `id`         BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '用户ID',
    `name`       VARCHAR(64)     NOT NULL COMMENT '用户名称',
    `phone`      VARCHAR(20)     NOT NULL COMMENT '手机号',
    `email`      VARCHAR(100)    NOT NULL COMMENT '邮箱',
    `id_card`    VARCHAR(20)     NOT NULL COMMENT '身份证号',
    `status`     INT             NOT NULL DEFAULT 1 COMMENT '用户状态: 1-正常, 2-禁用',
    `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME                 DEFAULT NULL COMMENT '删除时间',
    `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`    INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    UNIQUE KEY `uk_id_card` (`id_card`),
    KEY `idx_phone` (`phone`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='用户表';

-- 1.2 儿童档案表

CREATE TABLE IF NOT EXISTS `iam_children`
(
    `id`         BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '儿童ID',
    `name`       VARCHAR(64)     NOT NULL COMMENT '儿童姓名',
    `id_card`    VARCHAR(20)              DEFAULT NULL COMMENT '身份证号码',
    `gender`     TINYINT         NOT NULL DEFAULT 0 COMMENT '性别: 0-未知, 1-男, 2-女',
    `birthday`   VARCHAR(10)              DEFAULT NULL COMMENT '出生日期 (YYYY-MM-DD)',
    `height`     BIGINT                   DEFAULT NULL COMMENT '身高 (以0.1cm为单位)',
    `weight`     BIGINT                   DEFAULT NULL COMMENT '体重 (以0.1kg为单位)',
    `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME                 DEFAULT NULL COMMENT '删除时间',
    `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`    INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    UNIQUE KEY `uk_id_card` (`id_card`),
    KEY `idx_deleted_at` (`deleted_at`),
    KEY `idx_name_gender_birthday` (`name`, `gender`, `birthday`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='儿童档案表';

-- 1.3 监护关系表

CREATE TABLE IF NOT EXISTS `iam_guardianships`
(
    `id`             BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '监护关系ID',
    `user_id`        BIGINT UNSIGNED NOT NULL COMMENT '监护人ID (用户ID)',
    `child_id`       BIGINT UNSIGNED NOT NULL COMMENT '儿童ID',
    `relation`       VARCHAR(16)     NOT NULL COMMENT '监护关系: parent-父母, guardian-监护人',
    `established_at` DATETIME        NOT NULL COMMENT '建立时间',
    `revoked_at`     DATETIME                 DEFAULT NULL COMMENT '撤销时间',
    `created_at`     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME                 DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`        INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    KEY `idx_user_child_ref` (`user_id`, `child_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='监护关系表';

-- ============================================================================
-- Module 2: Authentication (Authn)
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

-- 2.4 JWKS 密钥表

CREATE TABLE IF NOT EXISTS `iam_jwks_keys`
(
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '密钥ID',
    `kid`        VARCHAR(64)     NOT NULL COMMENT 'Key ID',
    `status`     TINYINT         NOT NULL DEFAULT 1 COMMENT '1-Active, 2-Grace, 3-Retired',
    `kty`        VARCHAR(32)     NOT NULL COMMENT 'Key Type: RSA/EC',
    `use`        VARCHAR(16)     NOT NULL COMMENT '密钥用途: sig/enc',
    `alg`        VARCHAR(32)     NOT NULL COMMENT '算法: RS256/RS384/RS512 等',
    `jwk_json`   JSON            NOT NULL COMMENT '公钥JWK JSON',
    `not_before` DATETIME                 DEFAULT NULL COMMENT '生效时间',
    `not_after`  DATETIME                 DEFAULT NULL COMMENT '过期时间',
    `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE KEY `uk_kid` (`kid`),
    KEY `idx_status` (`status`),
    KEY `idx_alg` (`alg`),
    KEY `idx_not_after` (`not_after`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='JWKS 密钥表';

-- 2.5 会话表

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

-- 2.6 Token 黑名单表

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
-- Module 3: Authorization (Authz)
-- ============================================================================

-- 3.1 资源表

CREATE TABLE IF NOT EXISTS `iam_authz_resources`
(
    `id`           BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '资源ID',
    `key`          VARCHAR(128)    NOT NULL COMMENT '资源唯一标识键',
    `display_name` VARCHAR(128)             DEFAULT NULL COMMENT '资源显示名称',
    `app_name`     VARCHAR(32)              DEFAULT NULL COMMENT '所属应用名称',
    `domain`       VARCHAR(32)              DEFAULT NULL COMMENT '资源域',
    `type`         VARCHAR(32)              DEFAULT NULL COMMENT '资源类型',
    `actions`      TEXT                     DEFAULT NULL COMMENT '资源可用操作 (JSON数组格式)',
    `description`  VARCHAR(512)             DEFAULT NULL COMMENT '资源描述',
    `created_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`   DATETIME                 DEFAULT NULL COMMENT '删除时间',
    `created_by`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`      INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    UNIQUE KEY `uk_key` (`key`),
    KEY `idx_app_name` (`app_name`),
    KEY `idx_domain` (`domain`),
    KEY `idx_type` (`type`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='资源表';

-- 3.2 角色表

CREATE TABLE IF NOT EXISTS `iam_authz_roles`
(
    `id`           BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '角色ID',
    `name`         VARCHAR(64)     NOT NULL COMMENT '角色名称 (标识符)',
    `display_name` VARCHAR(128)             DEFAULT NULL COMMENT '角色显示名称',
    `tenant_id`    VARCHAR(64)     NOT NULL COMMENT '租户ID',
    `is_system`    TINYINT         NOT NULL DEFAULT 0 COMMENT '系统内置角色标识',
    `description`  VARCHAR(512)             DEFAULT NULL COMMENT '角色描述',
    `created_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`   DATETIME                 DEFAULT NULL COMMENT '删除时间',
    `created_by`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`      INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    UNIQUE KEY `uk_tenant_name` (`tenant_id`, `name`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='角色表';

-- 3.3 赋权表 (角色分配)

CREATE TABLE IF NOT EXISTS `iam_authz_assignments`
(
    `id`           BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '赋权记录ID',
    `subject_type` VARCHAR(16)     NOT NULL COMMENT '主体类型: user/group',
    `subject_id`   VARCHAR(64)     NOT NULL COMMENT '主体ID',
    `role_id`      BIGINT UNSIGNED NOT NULL COMMENT '角色ID',
    `tenant_id`    VARCHAR(64)     NOT NULL COMMENT '租户ID',
    `granted_by`   VARCHAR(64)              DEFAULT NULL COMMENT '授权操作人',
    `granted_at`   DATETIME        NOT NULL COMMENT '授权时间',
    `created_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`   DATETIME                 DEFAULT NULL COMMENT '删除时间',
    `created_by`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`      INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    KEY `idx_subject` (`subject_type`, `subject_id`),
    KEY `idx_role_id` (`role_id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='角色赋权表';

-- 3.4 策略版本表

CREATE TABLE IF NOT EXISTS `iam_authz_policy_versions`
(
    `id`             BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '版本记录ID',
    `tenant_id`      VARCHAR(64)     NOT NULL COMMENT '租户ID',
    `policy_version` BIGINT          NOT NULL COMMENT '策略版本号',
    `changed_by`     VARCHAR(64)              DEFAULT NULL COMMENT '变更操作人',
    `reason`         VARCHAR(512)             DEFAULT NULL COMMENT '变更原因',
    `created_at`     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME                 DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`        INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    UNIQUE KEY `uk_tenant_version` (`tenant_id`, `policy_version`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='策略版本表';

-- 3.5 Casbin 策略规则表

CREATE TABLE IF NOT EXISTS `iam_casbin_rule`
(
    `id`    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `ptype` VARCHAR(100)    NOT NULL COMMENT '策略类型 (p/g)',
    `v0`    VARCHAR(100) DEFAULT NULL COMMENT '值0 (subject)',
    `v1`    VARCHAR(100) DEFAULT NULL COMMENT '值1 (object)',
    `v2`    VARCHAR(100) DEFAULT NULL COMMENT '值2 (action)',
    `v3`    VARCHAR(100) DEFAULT NULL COMMENT '值3 (effect)',
    `v4`    VARCHAR(100) DEFAULT NULL COMMENT '值4 (扩展)',
    `v5`    VARCHAR(100) DEFAULT NULL COMMENT '值5 (扩展)',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_iam_casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='Casbin 策略规则表 - 存储 RBAC 策略规则';

-- ============================================================================
-- Module 4: Identity Provider (IDP)
-- ============================================================================

-- 4.1 微信应用表

CREATE TABLE IF NOT EXISTS `iam_idp_wechat_apps`
(
    `id`                     BIGINT UNSIGNED NOT NULL COMMENT '主键 ID (Snowflake)',
    `app_id`                 VARCHAR(64)     NOT NULL COMMENT '微信应用 ID (AppID)',
    `name`                   VARCHAR(255)    NOT NULL COMMENT '应用名称',
    `type`                   VARCHAR(32)     NOT NULL COMMENT '应用类型 (MiniProgram/MP/OpenPlatform)',
    `status`                 VARCHAR(32)     NOT NULL DEFAULT 'Enabled' COMMENT '应用状态 (Enabled/Disabled/Archived)',
    `auth_secret_cipher`     BLOB                     DEFAULT NULL COMMENT 'AppSecret 密文 (AES-GCM 加密)',
    `auth_secret_fp`         VARCHAR(128)             DEFAULT NULL COMMENT 'AppSecret 指纹 (SHA256)',
    `auth_secret_version`    INT             NOT NULL DEFAULT 0 COMMENT 'AppSecret 版本号',
    `auth_secret_rotated_at` DATETIME                 DEFAULT NULL COMMENT 'AppSecret 最后轮换时间',
    `msg_callback_token`     VARCHAR(128)             DEFAULT NULL COMMENT '消息推送回调 Token',
    `msg_aes_key_cipher`     BLOB                     DEFAULT NULL COMMENT '消息加密密钥密文 (EncodingAESKey)',
    `msg_secret_version`     INT             NOT NULL DEFAULT 0 COMMENT '消息密钥版本号',
    `msg_secret_rotated_at`  DATETIME                 DEFAULT NULL COMMENT '消息密钥最后轮换时间',
    `created_at`             DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`             DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_app_id` (`app_id`),
    KEY `idx_type` (`type`),
    KEY `idx_status` (`status`),
    KEY `idx_created_at` (`created_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='微信应用表 - 管理微信小程序/公众号应用配置';


-- ============================================================================
-- Module 5: Platform / System
-- ============================================================================

-- 5.1 租户表

CREATE TABLE IF NOT EXISTS `iam_tenants`
(
    `id`            VARCHAR(64)  NOT NULL COMMENT '租户ID',
    `name`          VARCHAR(100) NOT NULL COMMENT '租户名称',
    `code`          VARCHAR(50)  NOT NULL COMMENT '租户编码',
    `contact_name`  VARCHAR(100)          DEFAULT NULL COMMENT '联系人姓名',
    `contact_phone` VARCHAR(20)           DEFAULT NULL COMMENT '联系人电话',
    `contact_email` VARCHAR(100)          DEFAULT NULL COMMENT '联系人邮箱',
    `status`        VARCHAR(20)  NOT NULL DEFAULT 'active' COMMENT '状态 (active/inactive/suspended)',
    `max_users`     INT                   DEFAULT NULL COMMENT '最大用户数限制',
    `max_roles`     INT                   DEFAULT NULL COMMENT '最大角色数限制',
    `created_at`    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`    TIMESTAMP    NULL     DEFAULT NULL COMMENT '删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code` (`code`),
    KEY `idx_status` (`status`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='租户表 - 管理多租户信息';


-- 5.3 操作日志表

CREATE TABLE IF NOT EXISTS `iam_operation_logs`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `user_id`        BIGINT UNSIGNED          DEFAULT NULL COMMENT '操作用户ID',
    `tenant_id`      VARCHAR(64)              DEFAULT NULL COMMENT '租户ID',
    `operation_type` VARCHAR(50)     NOT NULL COMMENT '操作类型 (CREATE/UPDATE/DELETE/LOGIN/LOGOUT)',
    `resource_type`  VARCHAR(50)     NOT NULL COMMENT '资源类型 (user/role/resource/policy)',
    `resource_id`    VARCHAR(64)              DEFAULT NULL COMMENT '资源ID',
    `operation_desc` VARCHAR(500)             DEFAULT NULL COMMENT '操作描述',
    `ip_address`     VARCHAR(45)              DEFAULT NULL COMMENT 'IP地址',
    `user_agent`     VARCHAR(500)             DEFAULT NULL COMMENT '浏览器UA',
    `request_data`   TEXT                     DEFAULT NULL COMMENT '请求数据 (JSON)',
    `response_data`  TEXT                     DEFAULT NULL COMMENT '响应数据 (JSON)',
    `status`         VARCHAR(20)     NOT NULL COMMENT '状态 (success/failure)',
    `error_message`  VARCHAR(500)             DEFAULT NULL COMMENT '错误信息',
    `duration_ms`    INT                      DEFAULT NULL COMMENT '操作耗时 (毫秒)',
    `created_at`     TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    KEY `idx_iam_operation_logs_user_id` (`user_id`),
    KEY `idx_iam_operation_logs_tenant_id` (`tenant_id`),
    KEY `idx_iam_operation_logs_operation_type` (`operation_type`),
    KEY `idx_iam_operation_logs_resource_type` (`resource_type`),
    KEY `idx_iam_operation_logs_created_at` (`created_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='操作日志表 - 记录所有重要操作';

-- 5.4 审计日志表

CREATE TABLE IF NOT EXISTS `iam_audit_logs`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `event_id`       VARCHAR(64)     NOT NULL COMMENT '事件ID (UUID)',
    `event_type`     VARCHAR(50)     NOT NULL COMMENT '事件类型 (authentication/authorization/data_access)',
    `event_category` VARCHAR(50)     NOT NULL COMMENT '事件分类 (security/compliance/system)',
    `severity`       VARCHAR(20)     NOT NULL COMMENT '严重级别 (info/warning/error/critical)',
    `user_id`        BIGINT UNSIGNED          DEFAULT NULL COMMENT '用户ID',
    `subject`        VARCHAR(128)             DEFAULT NULL COMMENT '主体 (用户名/服务名)',
    `action`         VARCHAR(100)    NOT NULL COMMENT '动作',
    `object`         VARCHAR(256)             DEFAULT NULL COMMENT '对象 (资源标识)',
    `result`         VARCHAR(20)     NOT NULL COMMENT '结果 (success/failure/denied)',
    `ip_address`     VARCHAR(45)              DEFAULT NULL COMMENT 'IP地址',
    `details`        JSON                     DEFAULT NULL COMMENT '详细信息 (JSON)',
    `created_at`     TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_event_id` (`event_id`),
    KEY `idx_event_type` (`event_type`),
    KEY `idx_severity` (`severity`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_created_at` (`created_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='审计日志表 - 用于安全合规审计';

-- 5.5 数据字典表

CREATE TABLE IF NOT EXISTS `iam_data_dictionary`
(
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `dict_type`  VARCHAR(50)     NOT NULL COMMENT '字典类型 (user_status/gender/relation_type)',
    `dict_code`  VARCHAR(50)     NOT NULL COMMENT '字典编码',
    `dict_value` VARCHAR(200)    NOT NULL COMMENT '字典值',
    `dict_label` VARCHAR(200)    NOT NULL COMMENT '字典标签',
    `sort_order` INT             NOT NULL DEFAULT 0 COMMENT '排序',
    `is_default` TINYINT         NOT NULL DEFAULT 0 COMMENT '是否默认 (0=否 1=是)',
    `status`     TINYINT         NOT NULL DEFAULT 1 COMMENT '状态 (1=启用 0=禁用)',
    `remark`     VARCHAR(500)             DEFAULT NULL COMMENT '备注',
    `created_at` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_type_code` (`dict_type`, `dict_code`),
    KEY `idx_type` (`dict_type`),
    KEY `idx_status` (`status`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='数据字典表 - 管理系统枚举值';

-- ============================================================================
-- Schema 版本管理
-- ============================================================================

CREATE TABLE IF NOT EXISTS `iam_schema_version`
(
    `id`          INT         NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `version`     VARCHAR(20) NOT NULL COMMENT 'Schema版本号',
    `description` VARCHAR(500)         DEFAULT NULL COMMENT '版本说明',
    `applied_at`  TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '应用时间',
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='Schema 版本管理表';

-- ============================================================================
-- Seed Data
-- ============================================================================

-- ============================================================================
-- IAM Contracts 种子数据脚本
-- ============================================================================
-- 描述: 初始化系统所需的基础数据和测试数据
-- 版本: 3.0.0
-- 更新时间: 2025-10-31
-- 说明: 与 configs/mysql/schema.sql 完全同步
-- ============================================================================
-- 注意:
-- 1. Snowflake ID 应该由应用程序生成
-- 2. 这里为了演示，使用简单的大整数作为 ID
-- 3. 生产环境中，请使用 idutil.GetIntID() 生成真实的 Snowflake ID
-- ============================================================================

-- ============================================================================
-- 1. 租户数据
-- ============================================================================

-- 默认租户已在 schema.sql 中插入，这里添加测试租户
INSERT INTO `iam_tenants` (`id`, `name`, `code`, `contact_name`, `contact_phone`, `contact_email`, `status`, `max_users`, `max_roles`)
VALUES ('demo', '演示租户', 'DEMO', '张三', '13800138000', 'demo@example.com', 'active', 1000, 100)
ON DUPLICATE KEY UPDATE `name`=VALUES(`name`);

-- ============================================================================
-- 2. 用户数据 (User Center)
-- ============================================================================

-- 系统管理员和测试用户
INSERT INTO `iam_users` (
    `id`, `name`, `phone`, `email`, `id_card`, `status`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
-- 系统管理员
(
    1000000000000001,
    '系统管理员',
    '10086000001',
    'admin@system.com',
    '110101199001011001',
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 测试用户 1 - 张三
(
    1000000000000002,
    '张三',
    '13800138000',
    'zhangsan@example.com',
    '110101199001011002',
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 测试用户 2 - 李四
(
    1000000000000003,
    '李四',
    '13800138001',
    'lisi@example.com',
    '110101199001011003',
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 测试用户 3 - 王五 (监护人)
(
    1000000000000004,
    '王五',
    '13800138002',
    'wangwu@example.com',
    '110101198001011004',
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 测试用户 4 - 赵六 (监护人)
(
    1000000000000005,
    '赵六',
    '13800138003',
    'zhaoliu@example.com',
    '110101198001011005',
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `name`=VALUES(`name`);

-- ============================================================================
-- 3. 儿童数据 (User Center)
-- ============================================================================

INSERT INTO `iam_children` (
    `id`, `name`, `id_card`, `gender`, `birthday`, `height`, `weight`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
-- 儿童 1 - 小明
(
    2000000000000001,
    '小明',
    '110101201501011001',
    1, -- 男
    '2015-01-01',
    1450, -- 145.0cm
    350,  -- 35.0kg
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 儿童 2 - 小红
(
    2000000000000002,
    '小红',
    '110101201502011002',
    2, -- 女
    '2015-02-01',
    1420, -- 142.0cm
    330,  -- 33.0kg
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 儿童 3 - 小刚
(
    2000000000000003,
    '小刚',
    '110101201603011003',
    1, -- 男
    '2016-03-01',
    1380, -- 138.0cm
    310,  -- 31.0kg
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 儿童 4 - 小丽 (无身份证)
(
    2000000000000004,
    '小丽',
    NULL, -- 无身份证
    2, -- 女
    '2018-05-15',
    1100, -- 110.0cm
    200,  -- 20.0kg
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `name`=VALUES(`name`);

-- ============================================================================
-- 4. 监护关系数据 (User Center)
-- ============================================================================

INSERT INTO `iam_guardianships` (
    `id`, `user_id`, `child_id`, `relation`, `established_at`, `revoked_at`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
-- 王五 是 小明 的父亲
(
    3000000000000001,
    1000000000000004, -- 王五
    2000000000000001, -- 小明
    'father',
    NOW(),
    NULL,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 赵六 是 小红 的母亲
(
    3000000000000002,
    1000000000000005, -- 赵六
    2000000000000002, -- 小红
    'mother',
    NOW(),
    NULL,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 王五 是 小刚 的监护人
(
    3000000000000003,
    1000000000000004, -- 王五
    2000000000000003, -- 小刚
    'guardian',
    NOW(),
    NULL,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 赵六 是 小丽 的母亲
(
    3000000000000004,
    1000000000000005, -- 赵六
    2000000000000004, -- 小丽
    'mother',
    NOW(),
    NULL,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `relation`=VALUES(`relation`);

-- ============================================================================
-- 5. 认证账号数据 (Authentication)
-- ============================================================================

-- 运营后台账号
INSERT INTO `iam_auth_accounts` (
    `id`, `user_id`, `provider`, `external_id`, `app_id`, `status`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
-- 系统管理员 - 运营账号
(
    4000000000000001,
    1000000000000001, -- 系统管理员
    'operation',
    'admin',
    NULL,
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 张三 - 运营账号
(
    4000000000000002,
    1000000000000002, -- 张三
    'operation',
    'zhangsan',
    NULL,
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 王五 - 微信账号
(
    4000000000000003,
    1000000000000004, -- 王五
    'wechat',
    'wangwu_openid_123',
    'wx1234567890abcdef',
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `status`=VALUES(`status`);

-- 运营账号凭证 (密码: Admin@123)
-- 注意: 实际生产环境应使用真实的密码哈希算法
INSERT INTO `iam_auth_operation_accounts` (
    `id`, `account_id`, `username`, `password_hash`, `algo`, `params`,
    `failed_attempts`, `locked_until`, `last_changed_at`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
-- admin 账号 (密码: Admin@123)
(
    5000000000000001,
    4000000000000001, -- admin 账号ID
    'admin',
    0x243261243132246B5870656C6B3776366F6E4E704F6F6F67436C382E, -- bcrypt hash of 'Admin@123'
    'bcrypt',
    0x636F73743D3132, -- cost=12
    0,
    NULL,
    NOW(),
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- zhangsan 账号 (密码: Pass@123)
(
    5000000000000002,
    4000000000000002, -- zhangsan 账号ID
    'zhangsan',
    0x243261243132246B5870656C6B3776366F6E4E704F6F6F67436C382E, -- bcrypt hash placeholder
    'bcrypt',
    0x636F73743D3132, -- cost=12
    0,
    NULL,
    NOW(),
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `username`=VALUES(`username`);

-- 微信账号扩展信息
INSERT INTO `iam_auth_wechat_accounts` (
    `id`, `account_id`, `app_id`, `open_id`, `union_id`, `nickname`, `avatar_url`, `meta`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
(
    6000000000000001,
    4000000000000003, -- 王五的微信账号
    'wx1234567890abcdef',
    'wangwu_openid_123',
    'wangwu_unionid_456',
    '王五',
    'https://example.com/avatar/wangwu.jpg',
    '{"province": "北京", "city": "北京市"}',
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `nickname`=VALUES(`nickname`);

-- ============================================================================
-- 6. 授权资源数据 (Authorization)
-- ============================================================================

INSERT INTO `iam_authz_resources` (
    `id`, `key`, `display_name`, `app_name`, `domain`, `type`, `actions`, `description`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
-- 用户管理资源
(
    7000000000000001,
    'uc:users',
    '用户管理',
    'iam',
    'uc',
    'collection',
    '["create", "read", "update", "delete", "list"]',
    '用户中心的用户管理权限',
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 儿童管理资源
(
    7000000000000002,
    'uc:children',
    '儿童管理',
    'iam',
    'uc',
    'collection',
    '["create", "read", "update", "delete", "list"]',
    '用户中心的儿童档案管理权限',
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 监护关系管理资源
(
    7000000000000003,
    'uc:guardianships',
    '监护关系管理',
    'iam',
    'uc',
    'collection',
    '["create", "read", "update", "delete", "list", "revoke"]',
    '用户中心的监护关系管理权限',
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 角色管理资源
(
    7000000000000004,
    'authz:roles',
    '角色管理',
    'iam',
    'authz',
    'collection',
    '["create", "read", "update", "delete", "list", "assign"]',
    '授权模块的角色管理权限',
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 策略管理资源
(
    7000000000000005,
    'authz:policies',
    '策略管理',
    'iam',
    'authz',
    'collection',
    '["create", "read", "update", "delete", "list"]',
    '授权模块的策略管理权限',
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `display_name`=VALUES(`display_name`);

-- ============================================================================
-- 7. 角色赋权数据 (Authorization)
-- ============================================================================

INSERT INTO `iam_authz_assignments` (
    `id`, `subject_type`, `subject_id`, `role_id`, `tenant_id`, `granted_by`, `granted_at`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
-- 系统管理员 拥有 super_admin 角色
(
    8000000000000001,
    'user',
    '1000000000000001', -- 系统管理员
    1, -- super_admin
    'default',
    'system',
    NOW(),
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 张三 拥有 user 角色
(
    8000000000000002,
    'user',
    '1000000000000002', -- 张三
    3, -- user
    'default',
    'system',
    NOW(),
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 王五 拥有 user 角色
(
    8000000000000003,
    'user',
    '1000000000000004', -- 王五
    3, -- user
    'default',
    'system',
    NOW(),
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `role_id`=VALUES(`role_id`);

-- ============================================================================
-- 8. Casbin 策略规则 (Authorization)
-- ============================================================================

-- 超级管理员策略: 拥有所有权限
INSERT INTO `iam_casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
VALUES 
-- 角色策略: super_admin 可以对所有资源执行所有操作
('p', 'role:super_admin', '*', '*', 'allow', NULL, NULL),
-- 角色策略: tenant_admin 可以管理租户内资源
('p', 'role:tenant_admin', 'tenant:*', '*', 'allow', NULL, NULL),
-- 角色策略: user 只能读取和更新自己的信息
('p', 'role:user', 'user:self', 'read', 'allow', NULL, NULL),
('p', 'role:user', 'user:self', 'update', 'allow', NULL, NULL),
-- 角色继承: super_admin 继承 tenant_admin
('g', 'role:super_admin', 'role:tenant_admin', NULL, NULL, NULL, NULL),
-- 角色继承: tenant_admin 继承 user
('g', 'role:tenant_admin', 'role:user', NULL, NULL, NULL, NULL)
ON DUPLICATE KEY UPDATE `ptype`=VALUES(`ptype`);

-- ============================================================================
-- 9. 微信应用配置 (Identity Provider)
-- ============================================================================

-- 测试微信小程序应用
INSERT INTO `iam_idp_wechat_apps` (
    `id`, `app_id`, `name`, `type`, `status`,
    `auth_secret_cipher`, `auth_secret_fp`, `auth_secret_version`, `auth_secret_rotated_at`,
    `msg_callback_token`, `msg_aes_key_cipher`, `msg_secret_version`, `msg_secret_rotated_at`,
    `created_at`, `updated_at`
) VALUES
(
    9000000000000001,
    'wx1234567890abcdef',
    '测试小程序',
    'MiniProgram',
    'Enabled',
    0x616263646566, -- 密文占位符
    'fp_placeholder_123456',
    1,
    NOW(),
    'test_token_123',
    0x616263646566, -- 密文占位符
    1,
    NOW(),
    NOW(), NOW()
)
ON DUPLICATE KEY UPDATE `name`=VALUES(`name`);

-- ============================================================================
-- 10. JWKS 密钥 (Authentication)
-- ============================================================================

-- 示例 RSA 公钥 (测试用)
INSERT INTO `iam_jwks_keys` (
    `kid`, `status`, `kty`, `use`, `alg`, `jwk_json`, `not_before`, `not_after`,
    `created_at`, `updated_at`
) VALUES
(
    'rsa-key-2025-10-31',
    1, -- Active
    'RSA',
    'sig',
    'RS256',
    '{"kty":"RSA","use":"sig","alg":"RS256","n":"test_n_value","e":"AQAB"}',
    NOW(),
    DATE_ADD(NOW(), INTERVAL 1 YEAR),
    NOW(), NOW()
)
ON DUPLICATE KEY UPDATE `status`=VALUES(`status`);



-- 系统默认角色

INSERT INTO `iam_authz_roles` (`id`, `name`, `display_name`, `tenant_id`, `is_system`, `description`, `created_at`,
                               `updated_at`, `created_by`, `updated_by`, `deleted_by`, `version`)
VALUES (1, 'super_admin', '超级管理员', 'default', 1, '拥有所有权限', NOW(), NOW(), 0, 0, 0, 1),
       (2, 'tenant_admin', '租户管理员', 'default', 1, '管理本租户内的所有资源', NOW(), NOW(), 0, 0, 0, 1),
       (3, 'user', '普通用户', 'default', 1, '普通用户权限', NOW(), NOW(), 0, 0, 0, 1)
ON DUPLICATE KEY UPDATE `display_name`=VALUES(`display_name`),
                        `description`=VALUES(`description`);

-- 数据字典 - 性别

INSERT INTO `iam_data_dictionary` (`dict_type`, `dict_code`, `dict_value`, `dict_label`, `sort_order`, `is_default`)
VALUES ('gender', '0', '0', '未知', 1, 1),
       ('gender', '1', '1', '男', 2, 0),
       ('gender', '2', '2', '女', 3, 0)
ON DUPLICATE KEY UPDATE `dict_label`=VALUES(`dict_label`);

-- 数据字典 - 用户状态

INSERT INTO `iam_data_dictionary` (`dict_type`, `dict_code`, `dict_value`, `dict_label`, `sort_order`, `is_default`)
VALUES ('user_status', '1', '1', '正常', 1, 1),
       ('user_status', '2', '2', '禁用', 2, 0),
       ('user_status', '3', '3', '删除', 3, 0)
ON DUPLICATE KEY UPDATE `dict_label`=VALUES(`dict_label`);

-- 数据字典 - 监护关系

INSERT INTO `iam_data_dictionary` (`dict_type`, `dict_code`, `dict_value`, `dict_label`, `sort_order`)
VALUES ('relation_type', 'father', 'father', '父亲', 1),
       ('relation_type', 'mother', 'mother', '母亲', 2),
       ('relation_type', 'grandfather', 'grandfather', '祖父/外祖父', 3),
       ('relation_type', 'grandmother', 'grandmother', '祖母/外祖母', 4),
       ('relation_type', 'guardian', 'guardian', '法定监护人', 5)
ON DUPLICATE KEY UPDATE `dict_label`=VALUES(`dict_label`);

-- ============================================================================
-- 完成
-- ============================================================================

SELECT '种子数据加载完成！' AS message;
SELECT COUNT(*) AS user_count FROM `iam_users`;
SELECT COUNT(*) AS child_count FROM `iam_children`;
SELECT COUNT(*) AS guardianship_count FROM `iam_guardianships`;
SELECT COUNT(*) AS account_count FROM `iam_auth_accounts`;
SELECT COUNT(*) AS role_count FROM `iam_authz_roles`;
SELECT COUNT(*) AS resource_count FROM `iam_authz_resources`;


-- 记录 Schema 版本

INSERT INTO `iam_schema_version` (`version`, `description`)
VALUES ('2.0', '2025-10-31 - 完整的模块化 Schema 定义 (修订版)')
ON DUPLICATE KEY UPDATE `description`=VALUES(`description`);
