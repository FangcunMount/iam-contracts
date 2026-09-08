-- 建立独立的角色管理保护属性；授权域的移除由后续迁移完成。
ALTER TABLE authz_roles
    ADD COLUMN management_protection VARCHAR(16) NOT NULL DEFAULT 'standard',
    ADD CONSTRAINT chk_authz_role_management_protection CHECK (management_protection IN ('standard','protected'));
UPDATE authz_roles SET management_protection = 'protected' WHERE tenant_id = 'platform';
