-- 原域归属只能从备份恢复，禁止猜测回填。
SELECT iam_authorization_restore_requires_database_backup();
