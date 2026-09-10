-- Authorization rollback is a coordinated maintenance operation. Restore the
-- exact DDL/data from iam_role_inheritance_archives before older binaries.
SELECT iam_role_inheritance_requires_coordinated_restore();
