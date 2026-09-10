-- Run role-model-migrate apply, verify and archive-inheritance first, with
-- authorization/identity writers stopped. Historical migrations stay intact.
CREATE TABLE IF NOT EXISTS iam_role_inheritance_archives (
 migration_id VARCHAR(64) NOT NULL PRIMARY KEY,
 schema_sql LONGTEXT NOT NULL,
 rows_json LONGTEXT NOT NULL,
 checksum CHAR(64) NOT NULL,
 after_hash CHAR(64) NOT NULL,
 created_at DATETIME(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
SET @iam_role_archive_ready = (
 SELECT COUNT(*) FROM iam_role_inheritance_archives a
 JOIN iam_role_model_migrations m ON BINARY m.migration_id=BINARY a.migration_id
 WHERE a.migration_id='independent-business-roles-v1' AND m.status='applied'
 AND BINARY a.after_hash=BINARY m.after_hash
 AND BINARY a.checksum=BINARY SHA2(CONCAT(a.schema_sql,CHAR(0),a.rows_json),256)
 AND JSON_VALID(a.rows_json)
);
SET @iam_role_edges_active = (SELECT COUNT(*) FROM authz_role_inheritances WHERE revoked_at IS NULL AND deleted_at IS NULL);
SET @iam_role_legacy_active = (SELECT COUNT(*) FROM authz_roles WHERE name IN ('qs:staff','qs:evaluator','super_admin') AND deleted_at IS NULL);
SET @iam_role_sql = IF(@iam_role_archive_ready=1 AND @iam_role_edges_active=0 AND @iam_role_legacy_active=0,
 'SELECT 1','SELECT iam_independent_role_migration_and_archive_required()');
PREPARE iam_role_stmt FROM @iam_role_sql;
EXECUTE iam_role_stmt;
DEALLOCATE PREPARE iam_role_stmt;
DROP TABLE authz_role_inheritances;
