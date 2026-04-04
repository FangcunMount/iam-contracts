ALTER TABLE `auth_accounts`
    ADD COLUMN `scoped_tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '运营账号租户作用域，仅 type=opera 有效' AFTER `external_id`,
    ADD KEY `idx_scoped_tenant_id` (`scoped_tenant_id`);
