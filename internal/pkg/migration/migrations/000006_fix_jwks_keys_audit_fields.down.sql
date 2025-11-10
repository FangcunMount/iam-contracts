-- Rollback: Remove audit fields from iam_jwks_keys table

DROP INDEX `idx_deleted_at` ON `iam_jwks_keys`;

ALTER TABLE `iam_jwks_keys`
    DROP COLUMN `version`,
    DROP COLUMN `deleted_by`,
    DROP COLUMN `updated_by`,
    DROP COLUMN `created_by`,
    DROP COLUMN `deleted_at`;
