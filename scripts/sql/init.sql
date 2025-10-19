-- ============================================================================
-- IAM Contracts 数据库初始化脚本
-- ============================================================================
-- 数据库: iam_contracts
-- 版本: 1.0.0
-- 创建时间: 2025-10-18
-- ============================================================================

-- 创建数据库
CREATE DATABASE IF NOT EXISTS iam_contracts 
    DEFAULT CHARACTER SET utf8mb4 
    DEFAULT COLLATE utf8mb4_unicode_ci;

USE iam_contracts;

-- ============================================================================
-- 用户中心 (User Center) 表结构
-- ============================================================================

-- 用户表
CREATE TABLE IF NOT EXISTS `users` (
    `id` VARCHAR(64) NOT NULL COMMENT '用户ID',
    `tenant_id` VARCHAR(64) NOT NULL COMMENT '租户ID',
    `name` VARCHAR(100) NOT NULL COMMENT '用户姓名',
    `phone` VARCHAR(20) COMMENT '手机号',
    `email` VARCHAR(100) COMMENT '邮箱',
    `id_card` VARCHAR(18) COMMENT '身份证号',
    `status` VARCHAR(20) NOT NULL DEFAULT 'active' COMMENT '状态: active, inactive, banned',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间',
    `version` INT NOT NULL DEFAULT 0 COMMENT '版本号（乐观锁）',
    PRIMARY KEY (`id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_phone` (`phone`),
    KEY `idx_email` (`email`),
    KEY `idx_id_card` (`id_card`),
    KEY `idx_status` (`status`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- 儿童表
CREATE TABLE IF NOT EXISTS `children` (
    `id` VARCHAR(64) NOT NULL COMMENT '儿童ID',
    `tenant_id` VARCHAR(64) NOT NULL COMMENT '租户ID',
    `name` VARCHAR(100) NOT NULL COMMENT '儿童姓名',
    `gender` VARCHAR(10) NOT NULL COMMENT '性别: male, female',
    `birthday` DATE NOT NULL COMMENT '出生日期',
    `id_card` VARCHAR(18) COMMENT '身份证号',
    `height` DECIMAL(5,2) COMMENT '身高(cm)',
    `weight` DECIMAL(5,2) COMMENT '体重(kg)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间',
    `version` INT NOT NULL DEFAULT 0 COMMENT '版本号（乐观锁）',
    PRIMARY KEY (`id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_name_birthday` (`name`, `birthday`),
    KEY `idx_id_card` (`id_card`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='儿童表';

-- 监护关系表
CREATE TABLE IF NOT EXISTS `guardianships` (
    `id` VARCHAR(64) NOT NULL COMMENT '监护关系ID',
    `tenant_id` VARCHAR(64) NOT NULL COMMENT '租户ID',
    `user_id` VARCHAR(64) NOT NULL COMMENT '用户ID',
    `child_id` VARCHAR(64) NOT NULL COMMENT '儿童ID',
    `relation` VARCHAR(20) NOT NULL COMMENT '关系: father, mother, grandfather, grandmother, guardian',
    `granted_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '授予时间',
    `revoked_at` TIMESTAMP NULL DEFAULT NULL COMMENT '撤销时间',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间',
    `version` INT NOT NULL DEFAULT 0 COMMENT '版本号（乐观锁）',
    PRIMARY KEY (`id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_child_id` (`child_id`),
    KEY `idx_user_child` (`user_id`, `child_id`),
    KEY `idx_deleted_at` (`deleted_at`),
    CONSTRAINT `fk_guardianships_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_guardianships_child` FOREIGN KEY (`child_id`) REFERENCES `children` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='监护关系表';

-- ============================================================================
-- 认证中心 (Authentication Center) 表结构
-- ============================================================================

-- 账户表
CREATE TABLE IF NOT EXISTS `accounts` (
    `id` VARCHAR(64) NOT NULL COMMENT '账户ID',
    `tenant_id` VARCHAR(64) NOT NULL COMMENT '租户ID',
    `user_id` VARCHAR(64) NOT NULL COMMENT '用户ID',
    `provider` VARCHAR(20) NOT NULL COMMENT '认证提供商: wechat, wework, local',
    `external_id` VARCHAR(100) NOT NULL COMMENT '外部ID (如 openid, unionid)',
    `password_hash` VARCHAR(255) COMMENT '密码哈希（本地认证）',
    `status` VARCHAR(20) NOT NULL DEFAULT 'active' COMMENT '状态: active, inactive',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间',
    `version` INT NOT NULL DEFAULT 0 COMMENT '版本号（乐观锁）',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_provider_external_id` (`provider`, `external_id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_status` (`status`),
    KEY `idx_deleted_at` (`deleted_at`),
    CONSTRAINT `fk_accounts_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='账户表';

-- 会话表
CREATE TABLE IF NOT EXISTS `sessions` (
    `id` VARCHAR(64) NOT NULL COMMENT '会话ID',
    `tenant_id` VARCHAR(64) NOT NULL COMMENT '租户ID',
    `user_id` VARCHAR(64) NOT NULL COMMENT '用户ID',
    `refresh_token` VARCHAR(255) NOT NULL COMMENT 'Refresh Token',
    `access_token_id` VARCHAR(64) COMMENT '当前 Access Token ID',
    `device_id` VARCHAR(100) COMMENT '设备ID',
    `ip_address` VARCHAR(45) COMMENT 'IP地址',
    `user_agent` VARCHAR(500) COMMENT '浏览器UA',
    `expires_at` TIMESTAMP NOT NULL COMMENT '过期时间',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_refresh_token` (`refresh_token`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_expires_at` (`expires_at`),
    CONSTRAINT `fk_sessions_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='会话表';

-- 签名密钥表
CREATE TABLE IF NOT EXISTS `signing_keys` (
    `id` VARCHAR(64) NOT NULL COMMENT '密钥ID',
    `kid` VARCHAR(64) NOT NULL COMMENT '密钥标识符',
    `algorithm` VARCHAR(20) NOT NULL DEFAULT 'RS256' COMMENT '算法: RS256, RS384, RS512',
    `public_key` TEXT NOT NULL COMMENT '公钥（PEM格式）',
    `private_key` TEXT NOT NULL COMMENT '私钥（PEM格式，加密存储）',
    `status` VARCHAR(20) NOT NULL DEFAULT 'active' COMMENT '状态: active, grace, expired',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `expires_at` TIMESTAMP NOT NULL COMMENT '过期时间',
    `rotated_at` TIMESTAMP NULL DEFAULT NULL COMMENT '轮换时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_kid` (`kid`),
    KEY `idx_status` (`status`),
    KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='签名密钥表';

-- Token 黑名单表
CREATE TABLE IF NOT EXISTS `token_blacklist` (
    `id` VARCHAR(64) NOT NULL COMMENT 'ID',
    `token_id` VARCHAR(64) NOT NULL COMMENT 'Token ID (jti)',
    `user_id` VARCHAR(64) NOT NULL COMMENT '用户ID',
    `reason` VARCHAR(100) COMMENT '加入黑名单原因',
    `expires_at` TIMESTAMP NOT NULL COMMENT 'Token原过期时间',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_token_id` (`token_id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Token黑名单表';

-- ============================================================================
-- 授权中心 (Authorization Center) 表结构
-- ============================================================================

-- 资源表
CREATE TABLE IF NOT EXISTS `resources` (
    `id` VARCHAR(64) NOT NULL COMMENT '资源ID',
    `tenant_id` VARCHAR(64) NOT NULL COMMENT '租户ID',
    `name` VARCHAR(100) NOT NULL COMMENT '资源名称',
    `resource_type` VARCHAR(50) NOT NULL COMMENT '资源类型: api, menu, button, data',
    `resource_path` VARCHAR(200) COMMENT '资源路径',
    `method` VARCHAR(10) COMMENT 'HTTP方法: GET, POST, PUT, DELETE',
    `description` VARCHAR(500) COMMENT '描述',
    `parent_id` VARCHAR(64) COMMENT '父资源ID',
    `sort_order` INT DEFAULT 0 COMMENT '排序',
    `status` VARCHAR(20) NOT NULL DEFAULT 'active' COMMENT '状态: active, inactive',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间',
    `version` INT NOT NULL DEFAULT 0 COMMENT '版本号（乐观锁）',
    PRIMARY KEY (`id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_resource_type` (`resource_type`),
    KEY `idx_parent_id` (`parent_id`),
    KEY `idx_status` (`status`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='资源表';

-- 角色表
CREATE TABLE IF NOT EXISTS `roles` (
    `id` VARCHAR(64) NOT NULL COMMENT '角色ID',
    `tenant_id` VARCHAR(64) NOT NULL COMMENT '租户ID',
    `name` VARCHAR(100) NOT NULL COMMENT '角色名称',
    `code` VARCHAR(50) NOT NULL COMMENT '角色编码',
    `description` VARCHAR(500) COMMENT '描述',
    `is_system` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否系统角色',
    `status` VARCHAR(20) NOT NULL DEFAULT 'active' COMMENT '状态: active, inactive',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间',
    `version` INT NOT NULL DEFAULT 0 COMMENT '版本号（乐观锁）',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_tenant_code` (`tenant_id`, `code`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_status` (`status`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色表';

-- 用户角色关联表
CREATE TABLE IF NOT EXISTS `user_roles` (
    `id` VARCHAR(64) NOT NULL COMMENT 'ID',
    `tenant_id` VARCHAR(64) NOT NULL COMMENT '租户ID',
    `user_id` VARCHAR(64) NOT NULL COMMENT '用户ID',
    `role_id` VARCHAR(64) NOT NULL COMMENT '角色ID',
    `granted_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '授予时间',
    `granted_by` VARCHAR(64) COMMENT '授予人ID',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_role` (`user_id`, `role_id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_role_id` (`role_id`),
    CONSTRAINT `fk_user_roles_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_user_roles_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户角色关联表';

-- 角色资源关联表
CREATE TABLE IF NOT EXISTS `role_resources` (
    `id` VARCHAR(64) NOT NULL COMMENT 'ID',
    `tenant_id` VARCHAR(64) NOT NULL COMMENT '租户ID',
    `role_id` VARCHAR(64) NOT NULL COMMENT '角色ID',
    `resource_id` VARCHAR(64) NOT NULL COMMENT '资源ID',
    `actions` VARCHAR(200) COMMENT '允许的操作（逗号分隔）: read, write, delete',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_role_resource` (`role_id`, `resource_id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_role_id` (`role_id`),
    KEY `idx_resource_id` (`resource_id`),
    CONSTRAINT `fk_role_resources_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_role_resources_resource` FOREIGN KEY (`resource_id`) REFERENCES `resources` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色资源关联表';

-- Casbin 策略表
CREATE TABLE IF NOT EXISTS `casbin_rule` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `ptype` VARCHAR(100) NOT NULL COMMENT '策略类型',
    `v0` VARCHAR(100) COMMENT '值0',
    `v1` VARCHAR(100) COMMENT '值1',
    `v2` VARCHAR(100) COMMENT '值2',
    `v3` VARCHAR(100) COMMENT '值3',
    `v4` VARCHAR(100) COMMENT '值4',
    `v5` VARCHAR(100) COMMENT '值5',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Casbin策略规则表';

-- ============================================================================
-- 系统表
-- ============================================================================

-- 租户表
CREATE TABLE IF NOT EXISTS `tenants` (
    `id` VARCHAR(64) NOT NULL COMMENT '租户ID',
    `name` VARCHAR(100) NOT NULL COMMENT '租户名称',
    `code` VARCHAR(50) NOT NULL COMMENT '租户编码',
    `contact_name` VARCHAR(100) COMMENT '联系人姓名',
    `contact_phone` VARCHAR(20) COMMENT '联系人电话',
    `contact_email` VARCHAR(100) COMMENT '联系人邮箱',
    `status` VARCHAR(20) NOT NULL DEFAULT 'active' COMMENT '状态: active, inactive, suspended',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code` (`code`),
    KEY `idx_status` (`status`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='租户表';

-- 系统配置表
CREATE TABLE IF NOT EXISTS `system_configs` (
    `id` VARCHAR(64) NOT NULL COMMENT 'ID',
    `tenant_id` VARCHAR(64) COMMENT '租户ID（NULL表示全局配置）',
    `config_key` VARCHAR(100) NOT NULL COMMENT '配置键',
    `config_value` TEXT NOT NULL COMMENT '配置值（JSON格式）',
    `description` VARCHAR(500) COMMENT '描述',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_tenant_key` (`tenant_id`, `config_key`),
    KEY `idx_config_key` (`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统配置表';

-- 操作日志表
CREATE TABLE IF NOT EXISTS `operation_logs` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `tenant_id` VARCHAR(64) NOT NULL COMMENT '租户ID',
    `user_id` VARCHAR(64) COMMENT '操作用户ID',
    `operation_type` VARCHAR(50) NOT NULL COMMENT '操作类型: CREATE, UPDATE, DELETE',
    `resource_type` VARCHAR(50) NOT NULL COMMENT '资源类型: user, role, resource',
    `resource_id` VARCHAR(64) COMMENT '资源ID',
    `operation_desc` VARCHAR(500) COMMENT '操作描述',
    `ip_address` VARCHAR(45) COMMENT 'IP地址',
    `user_agent` VARCHAR(500) COMMENT '浏览器UA',
    `request_data` TEXT COMMENT '请求数据',
    `response_data` TEXT COMMENT '响应数据',
    `status` VARCHAR(20) NOT NULL COMMENT '状态: success, failure',
    `error_message` VARCHAR(500) COMMENT '错误信息',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_operation_type` (`operation_type`),
    KEY `idx_resource_type` (`resource_type`),
    KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='操作日志表';

-- ============================================================================
-- 初始化完成
-- ============================================================================

-- 显示所有表
SHOW TABLES;

-- 显示表统计信息
SELECT 
    TABLE_NAME as '表名',
    TABLE_COMMENT as '说明',
    TABLE_ROWS as '行数'
FROM 
    information_schema.TABLES 
WHERE 
    TABLE_SCHEMA = 'iam_contracts'
ORDER BY 
    TABLE_NAME;
