-- ============================================================================
-- IAM Contracts 种子数据脚本
-- ============================================================================
-- 描述: 初始化系统所需的基础数据和测试数据
-- 版本: 3.0.0
-- 更新时间: 2025-10-31
-- 说明: 与 configs/mysql/schema.sql 完全同步
-- ============================================================================
-- 注意:
-- 1. Snowflake ID 应该由应用程序生成
-- 2. 这里为了演示，使用简单的大整数作为 ID
-- 3. 生产环境中，请使用 idutil.GetIntID() 生成真实的 Snowflake ID
-- ============================================================================

USE `iam_contracts`;

-- ============================================================================
-- 1. 租户数据
-- ============================================================================

-- 默认租户已在 schema.sql 中插入，这里添加测试租户
INSERT INTO `iam_tenants` (`id`, `name`, `code`, `contact_name`, `contact_phone`, `contact_email`, `status`, `max_users`, `max_roles`)
VALUES ('demo', '演示租户', 'DEMO', '张三', '13800138000', 'demo@example.com', 'active', 1000, 100)
ON DUPLICATE KEY UPDATE `name`=VALUES(`name`);

-- ============================================================================
-- 2. 用户数据 (User Center)
-- ============================================================================

-- 系统管理员和测试用户
INSERT INTO `iam_users` (
    `id`, `name`, `phone`, `email`, `id_card`, `status`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
-- 系统管理员
(
    1000000000000001,
    '系统管理员',
    '10086000001',
    'admin@system.com',
    '110101199001011001',
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 测试用户 1 - 张三
(
    1000000000000002,
    '张三',
    '13800138000',
    'zhangsan@example.com',
    '110101199001011002',
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 测试用户 2 - 李四
(
    1000000000000003,
    '李四',
    '13800138001',
    'lisi@example.com',
    '110101199001011003',
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 测试用户 3 - 王五 (监护人)
(
    1000000000000004,
    '王五',
    '13800138002',
    'wangwu@example.com',
    '110101198001011004',
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 测试用户 4 - 赵六 (监护人)
(
    1000000000000005,
    '赵六',
    '13800138003',
    'zhaoliu@example.com',
    '110101198001011005',
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `name`=VALUES(`name`);

-- ============================================================================
-- 3. 儿童数据 (User Center)
-- ============================================================================

INSERT INTO `iam_children` (
    `id`, `name`, `id_card`, `gender`, `birthday`, `height`, `weight`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
-- 儿童 1 - 小明
(
    2000000000000001,
    '小明',
    '110101201501011001',
    1, -- 男
    '2015-01-01',
    1450, -- 145.0cm
    350,  -- 35.0kg
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 儿童 2 - 小红
(
    2000000000000002,
    '小红',
    '110101201502011002',
    2, -- 女
    '2015-02-01',
    1420, -- 142.0cm
    330,  -- 33.0kg
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 儿童 3 - 小刚
(
    2000000000000003,
    '小刚',
    '110101201603011003',
    1, -- 男
    '2016-03-01',
    1380, -- 138.0cm
    310,  -- 31.0kg
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 儿童 4 - 小丽 (无身份证)
(
    2000000000000004,
    '小丽',
    NULL, -- 无身份证
    2, -- 女
    '2018-05-15',
    1100, -- 110.0cm
    200,  -- 20.0kg
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `name`=VALUES(`name`);

-- ============================================================================
-- 4. 监护关系数据 (User Center)
-- ============================================================================

INSERT INTO `iam_guardianships` (
    `id`, `user_id`, `child_id`, `relation`, `established_at`, `revoked_at`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
-- 王五 是 小明 的父亲
(
    3000000000000001,
    1000000000000004, -- 王五
    2000000000000001, -- 小明
    'father',
    NOW(),
    NULL,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 赵六 是 小红 的母亲
(
    3000000000000002,
    1000000000000005, -- 赵六
    2000000000000002, -- 小红
    'mother',
    NOW(),
    NULL,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 王五 是 小刚 的监护人
(
    3000000000000003,
    1000000000000004, -- 王五
    2000000000000003, -- 小刚
    'guardian',
    NOW(),
    NULL,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 赵六 是 小丽 的母亲
(
    3000000000000004,
    1000000000000005, -- 赵六
    2000000000000004, -- 小丽
    'mother',
    NOW(),
    NULL,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `relation`=VALUES(`relation`);

-- ============================================================================
-- 5. 认证账号数据 (Authentication)
-- ============================================================================

-- 运营后台账号
INSERT INTO `iam_auth_accounts` (
    `id`, `user_id`, `provider`, `external_id`, `app_id`, `status`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
-- 系统管理员 - 运营账号
(
    4000000000000001,
    1000000000000001, -- 系统管理员
    'operation',
    'admin',
    NULL,
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 张三 - 运营账号
(
    4000000000000002,
    1000000000000002, -- 张三
    'operation',
    'zhangsan',
    NULL,
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 王五 - 微信账号
(
    4000000000000003,
    1000000000000004, -- 王五
    'wechat',
    'wangwu_openid_123',
    'wx1234567890abcdef',
    1,
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `status`=VALUES(`status`);

-- 运营账号凭证 (密码: Admin@123)
-- 注意: 实际生产环境应使用真实的密码哈希算法
INSERT INTO `iam_auth_operation_accounts` (
    `id`, `account_id`, `username`, `password_hash`, `algo`, `params`,
    `failed_attempts`, `locked_until`, `last_changed_at`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
-- admin 账号 (密码: Admin@123)
(
    5000000000000001,
    4000000000000001, -- admin 账号ID
    'admin',
    0x243261243132246B5870656C6B3776366F6E4E704F6F6F67436C382E, -- bcrypt hash of 'Admin@123'
    'bcrypt',
    0x636F73743D3132, -- cost=12
    0,
    NULL,
    NOW(),
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- zhangsan 账号 (密码: Pass@123)
(
    5000000000000002,
    4000000000000002, -- zhangsan 账号ID
    'zhangsan',
    0x243261243132246B5870656C6B3776366F6E4E704F6F6F67436C382E, -- bcrypt hash placeholder
    'bcrypt',
    0x636F73743D3132, -- cost=12
    0,
    NULL,
    NOW(),
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `username`=VALUES(`username`);

-- 微信账号扩展信息
INSERT INTO `iam_auth_wechat_accounts` (
    `id`, `account_id`, `app_id`, `open_id`, `union_id`, `nickname`, `avatar_url`, `meta`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
(
    6000000000000001,
    4000000000000003, -- 王五的微信账号
    'wx1234567890abcdef',
    'wangwu_openid_123',
    'wangwu_unionid_456',
    '王五',
    'https://example.com/avatar/wangwu.jpg',
    '{"province": "北京", "city": "北京市"}',
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `nickname`=VALUES(`nickname`);

-- ============================================================================
-- 6. 授权资源数据 (Authorization)
-- ============================================================================

INSERT INTO `iam_authz_resources` (
    `id`, `key`, `display_name`, `app_name`, `domain`, `type`, `actions`, `description`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
-- 用户管理资源
(
    7000000000000001,
    'uc:users',
    '用户管理',
    'iam',
    'uc',
    'collection',
    '["create", "read", "update", "delete", "list"]',
    '用户中心的用户管理权限',
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 儿童管理资源
(
    7000000000000002,
    'uc:children',
    '儿童管理',
    'iam',
    'uc',
    'collection',
    '["create", "read", "update", "delete", "list"]',
    '用户中心的儿童档案管理权限',
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 监护关系管理资源
(
    7000000000000003,
    'uc:guardianships',
    '监护关系管理',
    'iam',
    'uc',
    'collection',
    '["create", "read", "update", "delete", "list", "revoke"]',
    '用户中心的监护关系管理权限',
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 角色管理资源
(
    7000000000000004,
    'authz:roles',
    '角色管理',
    'iam',
    'authz',
    'collection',
    '["create", "read", "update", "delete", "list", "assign"]',
    '授权模块的角色管理权限',
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 策略管理资源
(
    7000000000000005,
    'authz:policies',
    '策略管理',
    'iam',
    'authz',
    'collection',
    '["create", "read", "update", "delete", "list"]',
    '授权模块的策略管理权限',
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `display_name`=VALUES(`display_name`);

-- ============================================================================
-- 7. 角色赋权数据 (Authorization)
-- ============================================================================

INSERT INTO `iam_authz_assignments` (
    `id`, `subject_type`, `subject_id`, `role_id`, `tenant_id`, `granted_by`, `granted_at`,
    `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`
) VALUES
-- 系统管理员 拥有 super_admin 角色
(
    8000000000000001,
    'user',
    '1000000000000001', -- 系统管理员
    1, -- super_admin
    'default',
    'system',
    NOW(),
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 张三 拥有 user 角色
(
    8000000000000002,
    'user',
    '1000000000000002', -- 张三
    3, -- user
    'default',
    'system',
    NOW(),
    NOW(), NOW(), NULL,
    0, 0, 0, 1
),
-- 王五 拥有 user 角色
(
    8000000000000003,
    'user',
    '1000000000000004', -- 王五
    3, -- user
    'default',
    'system',
    NOW(),
    NOW(), NOW(), NULL,
    0, 0, 0, 1
)
ON DUPLICATE KEY UPDATE `role_id`=VALUES(`role_id`);

-- ============================================================================
-- 8. Casbin 策略规则 (Authorization)
-- ============================================================================

-- 超级管理员策略: 拥有所有权限
INSERT INTO `iam_casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
VALUES 
-- 角色策略: super_admin 可以对所有资源执行所有操作
('p', 'role:super_admin', '*', '*', 'allow', NULL, NULL),
-- 角色策略: tenant_admin 可以管理租户内资源
('p', 'role:tenant_admin', 'tenant:*', '*', 'allow', NULL, NULL),
-- 角色策略: user 只能读取和更新自己的信息
('p', 'role:user', 'user:self', 'read', 'allow', NULL, NULL),
('p', 'role:user', 'user:self', 'update', 'allow', NULL, NULL),
-- 角色继承: super_admin 继承 tenant_admin
('g', 'role:super_admin', 'role:tenant_admin', NULL, NULL, NULL, NULL),
-- 角色继承: tenant_admin 继承 user
('g', 'role:tenant_admin', 'role:user', NULL, NULL, NULL, NULL)
ON DUPLICATE KEY UPDATE `ptype`=VALUES(`ptype`);

-- ============================================================================
-- 9. 微信应用配置 (Identity Provider)
-- ============================================================================

-- 测试微信小程序应用
INSERT INTO `iam_idp_wechat_apps` (
    `id`, `app_id`, `name`, `type`, `status`,
    `auth_secret_cipher`, `auth_secret_fp`, `auth_secret_version`, `auth_secret_rotated_at`,
    `msg_callback_token`, `msg_aes_key_cipher`, `msg_secret_version`, `msg_secret_rotated_at`,
    `created_at`, `updated_at`
) VALUES
(
    9000000000000001,
    'wx1234567890abcdef',
    '测试小程序',
    'MiniProgram',
    'Enabled',
    0x616263646566, -- 密文占位符
    'fp_placeholder_123456',
    1,
    NOW(),
    'test_token_123',
    0x616263646566, -- 密文占位符
    1,
    NOW(),
    NOW(), NOW()
)
ON DUPLICATE KEY UPDATE `name`=VALUES(`name`);

-- ============================================================================
-- 10. JWKS 密钥 (Authentication)
-- ============================================================================

-- 示例 RSA 公钥 (测试用)
INSERT INTO `iam_jwks_keys` (
    `kid`, `status`, `kty`, `use`, `alg`, `jwk_json`, `not_before`, `not_after`,
    `created_at`, `updated_at`
) VALUES
(
    'rsa-key-2025-10-31',
    1, -- Active
    'RSA',
    'sig',
    'RS256',
    '{"kty":"RSA","use":"sig","alg":"RS256","n":"test_n_value","e":"AQAB"}',
    NOW(),
    DATE_ADD(NOW(), INTERVAL 1 YEAR),
    NOW(), NOW()
)
ON DUPLICATE KEY UPDATE `status`=VALUES(`status`);

-- ============================================================================
-- 完成
-- ============================================================================

SELECT '种子数据加载完成！' AS message;
SELECT COUNT(*) AS user_count FROM `iam_users`;
SELECT COUNT(*) AS child_count FROM `iam_children`;
SELECT COUNT(*) AS guardianship_count FROM `iam_guardianships`;
SELECT COUNT(*) AS account_count FROM `iam_auth_accounts`;
SELECT COUNT(*) AS role_count FROM `iam_authz_roles`;
SELECT COUNT(*) AS resource_count FROM `iam_authz_resources`;
