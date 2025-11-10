-- Rollback: Remove audit fields from iam_idp_wechat_apps table

DROP INDEX `idx_deleted_at` ON `iam_idp_wechat_apps`;

ALTER TABLE `iam_idp_wechat_apps`
    DROP COLUMN `version`,
    DROP COLUMN `deleted_by`,
    DROP COLUMN `updated_by`,
    DROP COLUMN `created_by`,
    DROP COLUMN `deleted_at`;
