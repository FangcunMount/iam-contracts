-- ============================================================================
-- Migration Rollback: Remove child_id index from guardianships table
-- Version: 000002
-- Date: 2025-12-08
-- ============================================================================

-- 删除 guardianships 表的 child_id 索引
DROP INDEX `idx_child_id` ON `guardianships`;
