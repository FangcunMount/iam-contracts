-- 前向迁移；授权数据必须先由离线工具完成预检和准备。
CREATE TABLE IF NOT EXISTS iam_authorization_retirement_preparation (
 id TINYINT UNSIGNED PRIMARY KEY, source_hash VARCHAR(64) NOT NULL,
 prepared_hash VARCHAR(64) NOT NULL, initial_version BIGINT NOT NULL,
 prepared_at DATETIME(3) NOT NULL
);
SET @iam_retirement_ready = (SELECT COUNT(*) FROM iam_authorization_retirement_preparation
 WHERE id=1 AND source_hash REGEXP '^[0-9a-f]{64}$' AND prepared_hash REGEXP '^[0-9a-f]{64}$' AND initial_version>0);
SET @iam_retirement_empty = ((SELECT COUNT(*) FROM authz_roles)+(SELECT COUNT(*) FROM authz_assignments)+(SELECT COUNT(*) FROM authz_permission_grants)+(SELECT COUNT(*) FROM authz_role_inheritances)+(SELECT COUNT(*) FROM authz_policy_versions));
SET @iam_sql = IF(@iam_retirement_ready=1 OR @iam_retirement_empty=0,'SELECT 1','SELECT iam_authorization_retirement_preparation_required()');
PREPARE iam_stmt FROM @iam_sql;
EXECUTE iam_stmt;
DEALLOCATE PREPARE iam_stmt;
SET @iam_global_version = COALESCE((SELECT initial_version FROM iam_authorization_retirement_preparation WHERE id=1),1);
DROP TRIGGER IF EXISTS `iam_retire_authz_roles_INSERT`;
DROP TRIGGER IF EXISTS `iam_retire_authz_roles_UPDATE`;
DROP TRIGGER IF EXISTS `iam_retire_authz_roles_DELETE`;
DROP TRIGGER IF EXISTS `iam_retire_authz_assignments_INSERT`;
DROP TRIGGER IF EXISTS `iam_retire_authz_assignments_UPDATE`;
DROP TRIGGER IF EXISTS `iam_retire_authz_assignments_DELETE`;
DROP TRIGGER IF EXISTS `iam_retire_authz_role_inheritances_INSERT`;
DROP TRIGGER IF EXISTS `iam_retire_authz_role_inheritances_UPDATE`;
DROP TRIGGER IF EXISTS `iam_retire_authz_role_inheritances_DELETE`;
DROP TRIGGER IF EXISTS `iam_retire_authz_permission_grants_INSERT`;
DROP TRIGGER IF EXISTS `iam_retire_authz_permission_grants_UPDATE`;
DROP TRIGGER IF EXISTS `iam_retire_authz_permission_grants_DELETE`;
DROP TRIGGER IF EXISTS `iam_retire_authz_resources_INSERT`;
DROP TRIGGER IF EXISTS `iam_retire_authz_resources_UPDATE`;
DROP TRIGGER IF EXISTS `iam_retire_authz_resources_DELETE`;
DROP TRIGGER IF EXISTS `iam_retire_authz_policy_versions_INSERT`;
DROP TRIGGER IF EXISTS `iam_retire_authz_policy_versions_UPDATE`;
DROP TRIGGER IF EXISTS `iam_retire_authz_policy_versions_DELETE`;

ALTER TABLE authz_roles DROP INDEX uk_tenant_name, DROP INDEX idx_tenant_id, DROP COLUMN tenant_id, ADD UNIQUE INDEX uk_role_name(name);
ALTER TABLE authz_assignments DROP INDEX uk_authz_assignments_active, DROP INDEX idx_tenant_id, DROP COLUMN tenant_id, ADD UNIQUE INDEX uk_authz_assignments_active(subject_type,subject_id,role_id,active_guard);
ALTER TABLE authz_role_inheritances DROP INDEX uk_authz_role_inheritances_active, DROP INDEX idx_authz_role_inheritances_parent, DROP COLUMN tenant_id, ADD UNIQUE INDEX uk_authz_role_inheritances_active(role_id,inherited_role_id,active_guard), ADD INDEX idx_authz_role_inheritances_parent(inherited_role_id);
ALTER TABLE authz_permission_grants DROP INDEX idx_authz_permission_grants_tenant_role, DROP INDEX idx_authz_permission_grants_resource_action, DROP COLUMN tenant_id, ADD INDEX idx_authz_permission_grants_role(role_id), ADD INDEX idx_authz_permission_grants_resource_action(resource_pattern,action);
-- 原版本历史保留在切换前数据库备份；在线只保留一条全局版本。
DELETE FROM authz_policy_versions;
ALTER TABLE authz_policy_versions DROP INDEX uk_tenant_version, DROP COLUMN tenant_id, ADD CONSTRAINT chk_authz_policy_version_singleton CHECK (id=1 AND policy_version>0);
INSERT INTO authz_policy_versions(id,policy_version,changed_by,reason) VALUES(1,@iam_global_version,'migration','统一授权空间');
DROP TABLE iam_authorization_retirement_preparation;
