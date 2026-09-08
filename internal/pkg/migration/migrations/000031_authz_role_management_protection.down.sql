-- 仅允许配套旧版本整体回退时执行；不得在新版本运行期间取消角色保护。
ALTER TABLE authz_roles DROP CHECK chk_authz_role_management_protection,
    DROP COLUMN management_protection;
