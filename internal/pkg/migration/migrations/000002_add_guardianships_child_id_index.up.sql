-- ============================================================================
-- Migration: Add child_id index to guardianships table
-- Version: 000002
-- Description: Add index on child_id column for faster child-based lookups
-- Date: 2025-12-08
-- ============================================================================

-- 为 guardianships 表的 child_id 列添加索引
-- 用于优化通过儿童 ID 查询监护关系的性能
CREATE INDEX `idx_child_id` ON `guardianships` (`child_id`);
