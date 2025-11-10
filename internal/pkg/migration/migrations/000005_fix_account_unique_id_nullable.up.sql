-- ============================================================================
-- Migration 000005: 修复账户表 unique_id 字段允许为空
-- ============================================================================
-- 问题: iam_auth_accounts 表的 unique_id 字段定义为 NOT NULL DEFAULT ''
-- 原因: 运营后台账号(opera)没有 unionid，使用指针 *string 时会尝试插入 NULL
-- 解决: 将 unique_id 字段改为允许 NULL，运营账号存储 NULL，微信账号存储 unionid
-- ============================================================================

-- 修改 iam_auth_accounts 表的 unique_id 字段为可空
ALTER TABLE `iam_auth_accounts`
    MODIFY COLUMN `unique_id` VARCHAR(128) DEFAULT NULL COMMENT '全局唯一标识: 微信unionid|企微userid|运营后台为NULL';

-- 将现有的空字符串转换为 NULL
UPDATE `iam_auth_accounts`
SET `unique_id` = NULL
WHERE `unique_id` = '';
