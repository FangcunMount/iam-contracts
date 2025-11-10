-- ============================================================================
-- Migration 000005 Rollback: 恢复 unique_id 字段为非空
-- ============================================================================
-- 警告: 回滚前需确保所有 NULL 值的 unique_id 都已转换为空字符串
-- ============================================================================

-- 将 NULL 值转换为空字符串
UPDATE `iam_auth_accounts`
SET `unique_id` = ''
WHERE `unique_id` IS NULL;

-- 恢复 iam_auth_accounts 表的 unique_id 字段为非空
ALTER TABLE `iam_auth_accounts`
    MODIFY COLUMN `unique_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '全局唯一标识: unionid|运营后台为空';
