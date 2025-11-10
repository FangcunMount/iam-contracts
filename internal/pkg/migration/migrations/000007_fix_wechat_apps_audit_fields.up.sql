-- Migration: Add missing audit fields to iam_idp_wechat_apps table
-- Purpose: Add deleted_at, created_by, updated_by, deleted_by, version columns to support AuditFields

ALTER TABLE `iam_idp_wechat_apps`
    ADD COLUMN `deleted_at` DATETIME DEFAULT NULL COMMENT '软删除时间' AFTER `updated_at`,
    ADD COLUMN `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID' AFTER `deleted_at`,
    ADD COLUMN `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID' AFTER `created_by`,
    ADD COLUMN `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID' AFTER `updated_by`,
    ADD COLUMN `version` INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '乐观锁版本号' AFTER `deleted_by`;

-- Add index for deleted_at to support soft delete queries
CREATE INDEX `idx_deleted_at` ON `iam_idp_wechat_apps` (`deleted_at`);
