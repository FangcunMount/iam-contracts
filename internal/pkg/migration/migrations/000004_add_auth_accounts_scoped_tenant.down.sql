ALTER TABLE `auth_accounts`
    DROP INDEX `idx_scoped_tenant_id`,
    DROP COLUMN `scoped_tenant_id`;
