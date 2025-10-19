-- ============================================================================
-- IAM Contracts 种子数据脚本 v2.0
-- ============================================================================
-- 描述: 初始化系统所需的基础数据
-- 包含: 租户、管理员用户、系统角色、测试数据
-- 版本: 2.0.0
-- 创建时间: 2025-10-19
-- 说明: 此版本配合 init_v2.sql 使用，所有 ID 使用 BIGINT UNSIGNED
-- ============================================================================

USE iam_contracts;

-- ============================================================================
-- 工具函数说明
-- ============================================================================
-- 注意: Snowflake ID 应该由应用程序生成
-- 这里为了演示，使用简单的大整数作为 ID
-- 生产环境中，请使用 idutil.GetIntID() 生成真实的 Snowflake ID
-- ============================================================================

-- ============================================================================
-- 1. 租户数据 (可选，如需多租户支持)
-- ============================================================================

INSERT INTO `tenants` (`id`, `name`, `code`, `contact_name`, `contact_phone`, `contact_email`, `status`) VALUES
('tenant-system', '系统租户', 'SYSTEM', '系统管理员', '10086', 'admin@system.com', 'active'),
('tenant-demo', '演示租户', 'DEMO', '张三', '13800138000', 'demo@example.com', 'active');

-- ============================================================================
-- 2. 用户数据 (User Center)
-- ============================================================================

-- 管理员用户
-- 注意: ID 应该由 Snowflake 生成，这里使用简单的大整数
INSERT INTO `users` (
    `id`, `created_at`, `updated_at`, `deleted_at`, 
    `created_by`, `updated_by`, `deleted_by`, `version`,
    `name`, `phone`, `email`, `id_card`, `status`
) VALUES
-- 系统管理员
(
    1000000000000001, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    '系统管理员', '10086000001', 'admin@system.com', '110101199001011001', 1
),
-- 演示用户1
(
    1000000000000002, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    '张三', '13800138000', 'zhangsan@example.com', '110101199001011002', 1
),
-- 演示用户2
(
    1000000000000003, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    '李四', '13800138001', 'lisi@example.com', '110101199001011003', 1
),
-- 演示用户3 (监护人)
(
    1000000000000004, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    '王五', '13800138002', 'wangwu@example.com', '110101199001011004', 1
);

-- ============================================================================
-- 3. 儿童数据 (User Center)
-- ============================================================================

INSERT INTO `children` (
    `id`, `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`,
    `name`, `id_card`, `gender`, `birthday`, `height`, `weight`
) VALUES
-- 儿童1
(
    2000000000000001, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    '小明', '110101201501011001', 1, '2015-01-01', 1450, 350
),
-- 儿童2
(
    2000000000000002, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    '小红', '110101201502011002', 2, '2015-02-01', 1420, 330
),
-- 儿童3
(
    2000000000000003, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    '小刚', '110101201603011003', 1, '2016-03-01', 1380, 310
);

-- ============================================================================
-- 4. 监护关系数据 (User Center)
-- ============================================================================

INSERT INTO `guardianships` (
    `id`, `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`,
    `user_id`, `child_id`, `relation`, `established_at`, `revoked_at`
) VALUES
-- 张三 是 小明 的父亲
(
    3000000000000001, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    1000000000000002, 2000000000000001, 'father', NOW(), NULL
),
-- 李四 是 小红 的母亲
(
    3000000000000002, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    1000000000000003, 2000000000000002, 'mother', NOW(), NULL
),
-- 王五 是 小刚 的监护人
(
    3000000000000003, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    1000000000000004, 2000000000000003, 'guardian', NOW(), NULL
);

-- ============================================================================
-- 5. 认证账号数据 (Authentication Center)
-- ============================================================================

-- 主账号表
-- 密码: admin123 (BCrypt 哈希，需要由应用程序生成)
-- 这里只是示例，实际密码哈希需要由应用程序生成
INSERT INTO `auth_accounts` (
    `id`, `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`,
    `user_id`, `provider`, `external_id`, `app_id`, `status`
) VALUES
-- 系统管理员账号 (运营后台登录)
(
    4000000000000001, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    1000000000000001, 'operation', 'admin', NULL, 1
),
-- 张三的运营后台账号
(
    4000000000000002, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    1000000000000002, 'operation', 'zhangsan', NULL, 1
),
-- 李四的运营后台账号
(
    4000000000000003, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    1000000000000003, 'operation', 'lisi', NULL, 1
),
-- 张三的微信账号
(
    4000000000000004, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    1000000000000002, 'wechat', 'oXXXX-zhangsan-openid', 'wxapp123456', 1
);

-- 运营账号凭证表
-- 密码: admin123 (BCrypt: $2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iAt6Z5EH)
INSERT INTO `auth_operation_accounts` (
    `id`, `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`,
    `account_id`, `username`, `password_hash`, `algo`, `params`,
    `failed_attempts`, `locked_until`, `last_changed_at`
) VALUES
(
    5000000000000001, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    4000000000000001, 'admin', 
    UNHEX(SHA2('$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iAt6Z5EH', 256)),
    'bcrypt', NULL, 0, NULL, NOW()
),
(
    5000000000000002, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    4000000000000002, 'zhangsan',
    UNHEX(SHA2('$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iAt6Z5EH', 256)),
    'bcrypt', NULL, 0, NULL, NOW()
),
(
    5000000000000003, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    4000000000000003, 'lisi',
    UNHEX(SHA2('$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iAt6Z5EH', 256)),
    'bcrypt', NULL, 0, NULL, NOW()
);

-- 微信账号扩展表
INSERT INTO `auth_wechat_accounts` (
    `id`, `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`,
    `account_id`, `app_id`, `open_id`, `union_id`, `nickname`, `avatar_url`, `meta`
) VALUES
(
    6000000000000001, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    4000000000000004, 'wxapp123456', 'oXXXX-zhangsan-openid', 'uXXXX-zhangsan-unionid',
    '张三', 'https://wx.qlogo.cn/avatar.jpg',
    JSON_OBJECT('country', 'CN', 'province', '北京', 'city', '北京')
);

-- ============================================================================
-- 6. 授权数据 (Authorization Center)
-- ============================================================================

-- 角色数据
INSERT INTO `authz_roles` (
    `id`, `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`,
    `name`, `display_name`, `tenant_id`, `description`
) VALUES
-- 系统租户角色
(
    7000000000000001, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'SUPER_ADMIN', '超级管理员', 'tenant-system', '拥有系统所有权限'
),
-- 演示租户角色
(
    7000000000000002, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'ADMIN', '管理员', 'tenant-demo', '租户管理员，拥有租户内所有权限'
),
(
    7000000000000003, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'USER', '普通用户', 'tenant-demo', '普通用户，拥有基本权限'
),
(
    7000000000000004, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'GUARDIAN', '监护人', 'tenant-demo', '监护人角色，可管理儿童信息'
);

-- 用户角色赋权
INSERT INTO `authz_assignments` (
    `id`, `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`,
    `subject_type`, `subject_id`, `role_id`, `tenant_id`, `granted_by`, `granted_at`
) VALUES
-- 系统管理员分配超级管理员角色
(
    8000000000000001, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'user', '1000000000000001', 7000000000000001, 'tenant-system', NULL, NOW()
),
-- 张三分配管理员角色
(
    8000000000000002, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'user', '1000000000000002', 7000000000000002, 'tenant-demo', NULL, NOW()
),
-- 李四分配普通用户角色
(
    8000000000000003, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'user', '1000000000000003', 7000000000000003, 'tenant-demo', NULL, NOW()
),
-- 王五分配监护人角色
(
    8000000000000004, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'user', '1000000000000004', 7000000000000004, 'tenant-demo', NULL, NOW()
);

-- 资源数据
INSERT INTO `authz_resources` (
    `id`, `created_at`, `updated_at`, `deleted_at`,
    `created_by`, `updated_by`, `deleted_by`, `version`,
    `key`, `display_name`, `app_name`, `domain`, `type`, `actions`, `description`
) VALUES
-- 用户管理资源
(
    9000000000000001, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'uc.user.list', '用户列表', 'iam-contracts', 'uc', 'api',
    '["read"]', '查询用户列表'
),
(
    9000000000000002, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'uc.user.create', '创建用户', 'iam-contracts', 'uc', 'api',
    '["write"]', '创建新用户'
),
(
    9000000000000003, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'uc.user.get', '获取用户详情', 'iam-contracts', 'uc', 'api',
    '["read"]', '获取用户详细信息'
),
(
    9000000000000004, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'uc.user.update', '更新用户', 'iam-contracts', 'uc', 'api',
    '["write"]', '更新用户信息'
),
(
    9000000000000005, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'uc.user.delete', '删除用户', 'iam-contracts', 'uc', 'api',
    '["delete"]', '删除用户'
),
-- 儿童管理资源
(
    9000000000000006, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'uc.child.list', '儿童列表', 'iam-contracts', 'uc', 'api',
    '["read"]', '查询儿童列表'
),
(
    9000000000000007, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'uc.child.create', '创建儿童', 'iam-contracts', 'uc', 'api',
    '["write"]', '创建儿童档案'
),
(
    9000000000000008, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'uc.child.get', '获取儿童详情', 'iam-contracts', 'uc', 'api',
    '["read"]', '获取儿童详细信息'
),
(
    9000000000000009, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'uc.child.update', '更新儿童', 'iam-contracts', 'uc', 'api',
    '["write"]', '更新儿童信息'
),
(
    9000000000000010, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'uc.child.delete', '删除儿童', 'iam-contracts', 'uc', 'api',
    '["delete"]', '删除儿童档案'
),
-- 监护关系资源
(
    9000000000000011, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'uc.guardianship.list', '监护关系列表', 'iam-contracts', 'uc', 'api',
    '["read"]', '查询监护关系列表'
),
(
    9000000000000012, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'uc.guardianship.create', '创建监护关系', 'iam-contracts', 'uc', 'api',
    '["write"]', '建立监护关系'
),
(
    9000000000000013, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'uc.guardianship.get', '获取监护关系详情', 'iam-contracts', 'uc', 'api',
    '["read"]', '获取监护关系详情'
),
(
    9000000000000014, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'uc.guardianship.revoke', '撤销监护关系', 'iam-contracts', 'uc', 'api',
    '["delete"]', '撤销监护关系'
),
-- 认证资源
(
    9000000000000015, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'authn.login', '用户登录', 'iam-contracts', 'authn', 'api',
    '["execute"]', '用户登录接口'
),
(
    9000000000000016, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'authn.logout', '用户登出', 'iam-contracts', 'authn', 'api',
    '["execute"]', '用户登出接口'
),
(
    9000000000000017, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'authn.refresh', '刷新Token', 'iam-contracts', 'authn', 'api',
    '["execute"]', '刷新访问令牌'
),
(
    9000000000000018, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'authn.jwks', 'JWKS公钥', 'iam-contracts', 'authn', 'api',
    '["read"]', '获取JWT公钥集'
),
-- 角色管理资源
(
    9000000000000019, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'authz.role.list', '角色列表', 'iam-contracts', 'authz', 'api',
    '["read"]', '查询角色列表'
),
(
    9000000000000020, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'authz.role.create', '创建角色', 'iam-contracts', 'authz', 'api',
    '["write"]', '创建新角色'
),
(
    9000000000000021, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'authz.role.get', '获取角色详情', 'iam-contracts', 'authz', 'api',
    '["read"]', '获取角色详细信息'
),
(
    9000000000000022, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'authz.role.update', '更新角色', 'iam-contracts', 'authz', 'api',
    '["write"]', '更新角色信息'
),
(
    9000000000000023, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'authz.role.delete', '删除角色', 'iam-contracts', 'authz', 'api',
    '["delete"]', '删除角色'
),
(
    9000000000000024, NOW(), NOW(), NULL,
    0, 0, 0, 1,
    'authz.role.assign', '分配角色', 'iam-contracts', 'authz', 'api',
    '["write"]', '为用户分配角色'
);

-- ============================================================================
-- 7. Casbin 策略规则
-- ============================================================================

-- 管理员策略（所有权限）
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`) VALUES
('p', '7000000000000002', '/api/v1/*', '*', 'tenant-demo'),
('p', '7000000000000002', '/.well-known/*', 'GET', 'tenant-demo');

-- 监护人策略
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`) VALUES
('p', '7000000000000004', '/api/v1/children', 'GET', 'tenant-demo'),
('p', '7000000000000004', '/api/v1/children/:id', 'GET', 'tenant-demo'),
('p', '7000000000000004', '/api/v1/children/:id', 'PUT', 'tenant-demo'),
('p', '7000000000000004', '/api/v1/guardianships', 'GET', 'tenant-demo'),
('p', '7000000000000004', '/api/v1/guardianships/:id', 'GET', 'tenant-demo');

-- 普通用户策略
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`) VALUES
('p', '7000000000000003', '/api/v1/users/:id', 'GET', 'tenant-demo'),
('p', '7000000000000003', '/api/v1/children', 'GET', 'tenant-demo'),
('p', '7000000000000003', '/api/v1/children/:id', 'GET', 'tenant-demo');

-- ============================================================================
-- 8. 系统配置 (可选)
-- ============================================================================

INSERT INTO `system_configs` (`tenant_id`, `config_key`, `config_value`, `description`) VALUES
(NULL, 'jwt.access_token_ttl', '{"value": 3600}', 'Access Token 有效期（秒）'),
(NULL, 'jwt.refresh_token_ttl', '{"value": 604800}', 'Refresh Token 有效期（秒）'),
(NULL, 'jwt.algorithm', '{"value": "RS256"}', 'JWT 签名算法'),
('tenant-demo', 'user.max_children', '{"value": 5}', '每个用户最多可管理的儿童数量'),
('tenant-demo', 'child.min_age', '{"value": 0}', '儿童最小年龄（岁）'),
('tenant-demo', 'child.max_age', '{"value": 18}', '儿童最大年龄（岁）');

-- ============================================================================
-- 数据验证
-- ============================================================================

-- 检查用户数据
SELECT '=== 用户数据 ===' as '';
SELECT id, name, phone, email, status FROM users;

-- 检查儿童数据
SELECT '=== 儿童数据 ===' as '';
SELECT id, name, gender, birthday, height, weight FROM children;

-- 检查监护关系
SELECT '=== 监护关系 ===' as '';
SELECT 
    g.id,
    u.name as guardian_name,
    c.name as child_name,
    g.relation,
    g.established_at
FROM guardianships g
JOIN users u ON g.user_id = u.id
JOIN children c ON g.child_id = c.id;

-- 检查账号数据
SELECT '=== 账号数据 ===' as '';
SELECT 
    a.id,
    u.name as user_name,
    a.provider,
    a.external_id,
    a.status
FROM auth_accounts a
JOIN users u ON a.user_id = u.id;

-- 检查角色数据
SELECT '=== 角色数据 ===' as '';
SELECT id, name, display_name, tenant_id, description FROM authz_roles;

-- 检查角色赋权
SELECT '=== 角色赋权 ===' as '';
SELECT 
    a.id,
    a.subject_type,
    u.name as user_name,
    r.name as role_name,
    a.tenant_id,
    a.granted_at
FROM authz_assignments a
JOIN authz_roles r ON a.role_id = r.id
LEFT JOIN users u ON a.subject_id = CAST(u.id AS CHAR);

-- 检查资源数据
SELECT '=== 资源数据 ===' as '';
SELECT id, `key`, display_name, domain, type FROM authz_resources LIMIT 10;

-- ============================================================================
-- 完成
-- ============================================================================

SELECT '=== 种子数据加载完成 ===' as '';
SELECT 
    '用户' as '类型', COUNT(*) as '数量' FROM users
UNION ALL
SELECT '儿童', COUNT(*) FROM children
UNION ALL
SELECT '监护关系', COUNT(*) FROM guardianships
UNION ALL
SELECT '认证账号', COUNT(*) FROM auth_accounts
UNION ALL
SELECT '角色', COUNT(*) FROM authz_roles
UNION ALL
SELECT '角色赋权', COUNT(*) FROM authz_assignments
UNION ALL
SELECT '资源', COUNT(*) FROM authz_resources
UNION ALL
SELECT 'Casbin规则', COUNT(*) FROM casbin_rule;

-- ============================================================================
-- 说明
-- ============================================================================
-- 
-- 默认账号信息:
-- 
-- 1. 系统管理员
--    - 用户名: admin
--    - 密码: admin123
--    - 角色: 超级管理员
--    - 租户: tenant-system
--    
-- 2. 演示用户 - 张三
--    - 用户名: zhangsan
--    - 密码: admin123
--    - 角色: 管理员
--    - 租户: tenant-demo
--    - 监护关系: 小明的父亲
--    
-- 3. 演示用户 - 李四
--    - 用户名: lisi
--    - 密码: admin123
--    - 角色: 普通用户
--    - 租户: tenant-demo
--    - 监护关系: 小红的母亲
--    
-- 4. 演示用户 - 王五
--    - 用户名: 无 (未创建运营账号)
--    - 角色: 监护人
--    - 租户: tenant-demo
--    - 监护关系: 小刚的监护人
--    
-- 注意事项:
-- 1. 所有 ID 都使用大整数模拟 Snowflake ID
-- 2. 密码哈希使用简化的 SHA256，生产环境应使用 BCrypt/Argon2
-- 3. 测试时请修改为真实的密码哈希值
-- 4. 儿童身高/体重单位为 0.1 (如 1450 = 145.0cm, 350 = 35.0kg)
-- 
-- ============================================================================
