-- 回滚初始化 Schema
-- 按照依赖关系倒序删除所有表

USE `iam_contracts`;

-- 删除所有表
DROP TABLE IF EXISTS `iam_operation_logs`;
DROP TABLE IF EXISTS `iam_audit_logs`;
DROP TABLE IF EXISTS `iam_data_dictionary`;
DROP TABLE IF EXISTS `iam_schema_version`;
DROP TABLE IF EXISTS `iam_tenants`;

DROP TABLE IF EXISTS `iam_idp_wechat_apps`;

DROP TABLE IF EXISTS `iam_casbin_rule`;
DROP TABLE IF EXISTS `iam_authz_policy_versions`;
DROP TABLE IF EXISTS `iam_authz_assignments`;
DROP TABLE IF EXISTS `iam_authz_roles`;
DROP TABLE IF EXISTS `iam_authz_resources`;

DROP TABLE IF EXISTS `iam_auth_token_blacklist`;
DROP TABLE IF EXISTS `iam_auth_sessions`;
DROP TABLE IF EXISTS `iam_jwks_keys`;
DROP TABLE IF EXISTS `iam_auth_operation_accounts`;
DROP TABLE IF EXISTS `iam_auth_wechat_accounts`;
DROP TABLE IF EXISTS `iam_auth_accounts`;

DROP TABLE IF EXISTS `iam_guardianships`;
DROP TABLE IF EXISTS `iam_children`;
DROP TABLE IF EXISTS `iam_users`;
