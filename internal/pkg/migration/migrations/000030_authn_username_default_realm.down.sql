-- 只解除写入约束，不猜测或重建历史数字 Realm。
-- 需要恢复旧命名空间时，必须使用升级前备份；已统一的用户名保持 default。
ALTER TABLE auth_login_identities DROP CHECK chk_authn_username_default_realm;
