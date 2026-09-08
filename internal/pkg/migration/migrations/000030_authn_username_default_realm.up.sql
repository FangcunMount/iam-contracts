-- AuthN 不再支持数字租户；用户名统一使用 default 命名空间。
-- 部署时停止旧版本的注册/绑定写入。任何同名身份（包括禁用/删除记录）
-- 都必须先人工处理，不能自动选择用户、合并身份或删除凭据。
SET @iam_username_conflicts = (
    SELECT COUNT(*) FROM (
        SELECT identifier FROM auth_login_identities
        WHERE provider = 'username'
        GROUP BY identifier HAVING COUNT(*) > 1
    ) AS conflicts
);
DROP TEMPORARY TABLE IF EXISTS iam_username_realm_assertion;
CREATE TEMPORARY TABLE iam_username_realm_assertion (
    message VARCHAR(128) NOT NULL PRIMARY KEY
);
INSERT INTO iam_username_realm_assertion VALUES ('AuthN username realm conflict; resolve duplicate usernames first');
INSERT INTO iam_username_realm_assertion
SELECT 'AuthN username realm conflict; resolve duplicate usernames first'
WHERE @iam_username_conflicts <> 0;
DROP TEMPORARY TABLE iam_username_realm_assertion;

UPDATE auth_login_identities SET realm = 'default'
WHERE provider = 'username' AND BINARY realm <> BINARY 'default';

ALTER TABLE auth_login_identities
    ADD CONSTRAINT chk_authn_username_default_realm
    CHECK (provider <> 'username' OR BINARY realm = BINARY 'default');
