-- ============================================================================
-- IAM Contracts 种子数据脚本
-- ============================================================================
-- 描述: 初始化系统所需的基础数据
-- 包含: 租户、管理员用户、系统角色、基础资源、测试数据
-- ============================================================================

USE iam_contracts;

-- ============================================================================
-- 1. 租户数据
-- ============================================================================

INSERT INTO `tenants` (`id`, `name`, `code`, `contact_name`, `contact_phone`, `contact_email`, `status`) VALUES
('tenant-system', '系统租户', 'SYSTEM', '系统管理员', '10086', 'admin@system.com', 'active'),
('tenant-demo', '演示租户', 'DEMO', '张三', '13800138000', 'demo@example.com', 'active');

-- ============================================================================
-- 2. 管理员用户数据
-- ============================================================================

-- 系统管理员
INSERT INTO `users` (`id`, `tenant_id`, `name`, `phone`, `email`, `status`) VALUES
('user-admin', 'tenant-system', '系统管理员', '10086', 'admin@system.com', 'active'),
('user-demo-001', 'tenant-demo', '张三', '13800138000', 'zhangsan@example.com', 'active'),
('user-demo-002', 'tenant-demo', '李四', '13800138001', 'lisi@example.com', 'active');

-- 管理员账户（密码: admin123）
INSERT INTO `accounts` (`id`, `tenant_id`, `user_id`, `provider`, `external_id`, `password_hash`, `status`) VALUES
('account-admin', 'tenant-system', 'user-admin', 'local', 'admin', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iAt6Z5EH', 'active'),
('account-demo-001', 'tenant-demo', 'user-demo-001', 'local', 'zhangsan', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iAt6Z5EH', 'active'),
('account-demo-002', 'tenant-demo', 'user-demo-002', 'local', 'lisi', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iAt6Z5EH', 'active');

-- ============================================================================
-- 3. 系统角色数据
-- ============================================================================

INSERT INTO `roles` (`id`, `tenant_id`, `name`, `code`, `description`, `is_system`, `status`) VALUES
-- 系统租户角色
('role-superadmin', 'tenant-system', '超级管理员', 'SUPER_ADMIN', '拥有系统所有权限', 1, 'active'),

-- 演示租户角色
('role-admin', 'tenant-demo', '管理员', 'ADMIN', '租户管理员，拥有租户内所有权限', 1, 'active'),
('role-user', 'tenant-demo', '普通用户', 'USER', '普通用户，拥有基本权限', 1, 'active'),
('role-guardian', 'tenant-demo', '监护人', 'GUARDIAN', '监护人角色，可管理儿童信息', 0, 'active');

-- ============================================================================
-- 4. 用户角色关联
-- ============================================================================

INSERT INTO `user_roles` (`id`, `tenant_id`, `user_id`, `role_id`, `granted_by`) VALUES
('ur-001', 'tenant-system', 'user-admin', 'role-superadmin', NULL),
('ur-002', 'tenant-demo', 'user-demo-001', 'role-admin', NULL),
('ur-003', 'tenant-demo', 'user-demo-002', 'role-guardian', NULL);

-- ============================================================================
-- 5. 资源数据（API资源）
-- ============================================================================

-- 用户中心资源
INSERT INTO `resources` (`id`, `tenant_id`, `name`, `resource_type`, `resource_path`, `method`, `description`, `status`) VALUES
-- 用户管理
('res-user-list', 'tenant-demo', '用户列表', 'api', '/api/v1/users', 'GET', '查询用户列表', 'active'),
('res-user-create', 'tenant-demo', '创建用户', 'api', '/api/v1/users', 'POST', '创建新用户', 'active'),
('res-user-get', 'tenant-demo', '获取用户详情', 'api', '/api/v1/users/:id', 'GET', '获取用户详细信息', 'active'),
('res-user-update', 'tenant-demo', '更新用户', 'api', '/api/v1/users/:id', 'PUT', '更新用户信息', 'active'),
('res-user-delete', 'tenant-demo', '删除用户', 'api', '/api/v1/users/:id', 'DELETE', '删除用户', 'active'),

-- 儿童管理
('res-child-list', 'tenant-demo', '儿童列表', 'api', '/api/v1/children', 'GET', '查询儿童列表', 'active'),
('res-child-create', 'tenant-demo', '创建儿童', 'api', '/api/v1/children', 'POST', '创建儿童档案', 'active'),
('res-child-get', 'tenant-demo', '获取儿童详情', 'api', '/api/v1/children/:id', 'GET', '获取儿童详细信息', 'active'),
('res-child-update', 'tenant-demo', '更新儿童', 'api', '/api/v1/children/:id', 'PUT', '更新儿童信息', 'active'),
('res-child-delete', 'tenant-demo', '删除儿童', 'api', '/api/v1/children/:id', 'DELETE', '删除儿童档案', 'active'),

-- 监护关系管理
('res-guardianship-list', 'tenant-demo', '监护关系列表', 'api', '/api/v1/guardianships', 'GET', '查询监护关系列表', 'active'),
('res-guardianship-create', 'tenant-demo', '创建监护关系', 'api', '/api/v1/guardianships', 'POST', '建立监护关系', 'active'),
('res-guardianship-get', 'tenant-demo', '获取监护关系详情', 'api', '/api/v1/guardianships/:id', 'GET', '获取监护关系详情', 'active'),
('res-guardianship-revoke', 'tenant-demo', '撤销监护关系', 'api', '/api/v1/guardianships/:id/revoke', 'POST', '撤销监护关系', 'active');

-- 认证资源
INSERT INTO `resources` (`id`, `tenant_id`, `name`, `resource_type`, `resource_path`, `method`, `description`, `status`) VALUES
('res-auth-login', 'tenant-demo', '用户登录', 'api', '/api/v1/auth/login', 'POST', '用户登录接口', 'active'),
('res-auth-logout', 'tenant-demo', '用户登出', 'api', '/api/v1/auth/logout', 'POST', '用户登出接口', 'active'),
('res-auth-refresh', 'tenant-demo', '刷新Token', 'api', '/api/v1/auth/refresh', 'POST', '刷新访问令牌', 'active'),
('res-auth-jwks', 'tenant-demo', 'JWKS公钥', 'api', '/.well-known/jwks.json', 'GET', '获取JWT公钥集', 'active');

-- 角色管理资源
INSERT INTO `resources` (`id`, `tenant_id`, `name`, `resource_type`, `resource_path`, `method`, `description`, `status`) VALUES
('res-role-list', 'tenant-demo', '角色列表', 'api', '/api/v1/roles', 'GET', '查询角色列表', 'active'),
('res-role-create', 'tenant-demo', '创建角色', 'api', '/api/v1/roles', 'POST', '创建新角色', 'active'),
('res-role-get', 'tenant-demo', '获取角色详情', 'api', '/api/v1/roles/:id', 'GET', '获取角色详细信息', 'active'),
('res-role-update', 'tenant-demo', '更新角色', 'api', '/api/v1/roles/:id', 'PUT', '更新角色信息', 'active'),
('res-role-delete', 'tenant-demo', '删除角色', 'api', '/api/v1/roles/:id', 'DELETE', '删除角色', 'active'),
('res-role-assign', 'tenant-demo', '分配角色', 'api', '/api/v1/users/:id/roles', 'POST', '为用户分配角色', 'active');

-- ============================================================================
-- 6. 角色资源关联（权限配置）
-- ============================================================================

-- 管理员角色权限（拥有所有权限）
INSERT INTO `role_resources` (`id`, `tenant_id`, `role_id`, `resource_id`, `actions`) 
SELECT 
    CONCAT('rr-admin-', SUBSTRING(id, 5)) as id,
    tenant_id,
    'role-admin' as role_id,
    id as resource_id,
    'read,write,delete' as actions
FROM `resources` 
WHERE tenant_id = 'tenant-demo';

-- 监护人角色权限（儿童和监护关系管理）
INSERT INTO `role_resources` (`id`, `tenant_id`, `role_id`, `resource_id`, `actions`) VALUES
('rr-guardian-child-list', 'tenant-demo', 'role-guardian', 'res-child-list', 'read'),
('rr-guardian-child-get', 'tenant-demo', 'role-guardian', 'res-child-get', 'read'),
('rr-guardian-child-update', 'tenant-demo', 'role-guardian', 'res-child-update', 'write'),
('rr-guardian-guardianship-list', 'tenant-demo', 'role-guardian', 'res-guardianship-list', 'read'),
('rr-guardian-guardianship-get', 'tenant-demo', 'role-guardian', 'res-guardianship-get', 'read');

-- 普通用户角色权限（只读）
INSERT INTO `role_resources` (`id`, `tenant_id`, `role_id`, `resource_id`, `actions`) VALUES
('rr-user-user-get', 'tenant-demo', 'role-user', 'res-user-get', 'read'),
('rr-user-child-list', 'tenant-demo', 'role-user', 'res-child-list', 'read'),
('rr-user-child-get', 'tenant-demo', 'role-user', 'res-child-get', 'read');

-- ============================================================================
-- 7. Casbin 策略规则
-- ============================================================================

-- 管理员策略（所有权限）
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`) VALUES
('p', 'role-admin', '/api/v1/*', '*', 'tenant-demo'),
('p', 'role-admin', '/.well-known/*', 'GET', 'tenant-demo');

-- 监护人策略
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`) VALUES
('p', 'role-guardian', '/api/v1/children', 'GET', 'tenant-demo'),
('p', 'role-guardian', '/api/v1/children/:id', 'GET', 'tenant-demo'),
('p', 'role-guardian', '/api/v1/children/:id', 'PUT', 'tenant-demo'),
('p', 'role-guardian', '/api/v1/guardianships', 'GET', 'tenant-demo'),
('p', 'role-guardian', '/api/v1/guardianships/:id', 'GET', 'tenant-demo');

-- 普通用户策略
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`) VALUES
('p', 'role-user', '/api/v1/users/:id', 'GET', 'tenant-demo'),
('p', 'role-user', '/api/v1/children', 'GET', 'tenant-demo'),
('p', 'role-user', '/api/v1/children/:id', 'GET', 'tenant-demo');

-- 角色继承（暂时为空，可根据需要配置）
-- INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`) VALUES
-- ('g', 'role-guardian', 'role-user', 'tenant-demo');

-- ============================================================================
-- 8. 测试数据（儿童和监护关系）
-- ============================================================================

-- 测试儿童
INSERT INTO `children` (`id`, `tenant_id`, `name`, `gender`, `birthday`, `height`, `weight`) VALUES
('child-001', 'tenant-demo', '小明', 'male', '2018-05-15', 120.5, 25.3),
('child-002', 'tenant-demo', '小红', 'female', '2019-08-20', 110.2, 22.1),
('child-003', 'tenant-demo', '小刚', 'male', '2020-03-10', 95.8, 18.5);

-- 测试监护关系
INSERT INTO `guardianships` (`id`, `tenant_id`, `user_id`, `child_id`, `relation`) VALUES
('guardianship-001', 'tenant-demo', 'user-demo-001', 'child-001', 'father'),
('guardianship-002', 'tenant-demo', 'user-demo-001', 'child-002', 'father'),
('guardianship-003', 'tenant-demo', 'user-demo-002', 'child-001', 'mother');

-- ============================================================================
-- 9. 系统配置
-- ============================================================================

INSERT INTO `system_configs` (`id`, `tenant_id`, `config_key`, `config_value`, `description`) VALUES
('config-global-jwt', NULL, 'jwt.access_token_ttl', '{"value": 3600, "unit": "seconds"}', 'Access Token 有效期'),
('config-global-refresh', NULL, 'jwt.refresh_token_ttl', '{"value": 2592000, "unit": "seconds"}', 'Refresh Token 有效期（30天）'),
('config-global-key-rotation', NULL, 'jwt.key_rotation_days', '{"value": 90}', 'JWT密钥轮换周期（天）'),
('config-demo-max-children', 'tenant-demo', 'guardian.max_children_per_user', '{"value": 5}', '每个用户最多可监护的儿童数量');

-- ============================================================================
-- 10. 生成初始签名密钥（示例）
-- ============================================================================
-- 注意：实际生产环境中，私钥应该使用加密存储，这里仅为演示
-- 实际部署时应该通过应用程序生成真实的RSA密钥对

INSERT INTO `signing_keys` (`id`, `kid`, `algorithm`, `public_key`, `private_key`, `status`, `expires_at`) VALUES
('key-001', 
 'iam-key-2025-01', 
 'RS256',
 '-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...(示例公钥)\n-----END PUBLIC KEY-----',
 '-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...(示例私钥-实际应加密存储)\n-----END RSA PRIVATE KEY-----',
 'active',
 DATE_ADD(NOW(), INTERVAL 90 DAY));

-- ============================================================================
-- 数据验证查询
-- ============================================================================

-- 查看租户数据
SELECT '租户数据' as '数据类型';
SELECT id, name, code, status FROM tenants;

-- 查看用户数据
SELECT '用户数据' as '数据类型';
SELECT u.id, u.tenant_id, u.name, u.phone, u.email, u.status 
FROM users u;

-- 查看角色数据
SELECT '角色数据' as '数据类型';
SELECT r.id, r.tenant_id, r.name, r.code, r.is_system 
FROM roles r;

-- 查看用户角色关联
SELECT '用户角色关联' as '数据类型';
SELECT ur.id, u.name as '用户名', r.name as '角色名', ur.granted_at 
FROM user_roles ur
JOIN users u ON ur.user_id = u.id
JOIN roles r ON ur.role_id = r.id;

-- 查看资源数量
SELECT '资源统计' as '数据类型';
SELECT resource_type as '资源类型', COUNT(*) as '数量' 
FROM resources 
GROUP BY resource_type;

-- 查看儿童和监护关系
SELECT '儿童与监护关系' as '数据类型';
SELECT 
    c.name as '儿童姓名',
    c.gender as '性别',
    c.birthday as '生日',
    u.name as '监护人',
    g.relation as '关系'
FROM guardianships g
JOIN children c ON g.child_id = c.id
JOIN users u ON g.user_id = u.id
WHERE g.revoked_at IS NULL;

-- 查看Casbin策略数量
SELECT 'Casbin策略统计' as '数据类型';
SELECT ptype as '策略类型', COUNT(*) as '数量' 
FROM casbin_rule 
GROUP BY ptype;

-- ============================================================================
-- 种子数据加载完成
-- ============================================================================
SELECT '============================================' as '';
SELECT '种子数据加载完成！' as '状态';
SELECT '============================================' as '';
SELECT '默认管理员账户:' as '';
SELECT '  用户名: admin / 密码: admin123' as '系统管理员';
SELECT '  用户名: zhangsan / 密码: admin123' as '演示租户管理员';
SELECT '  用户名: lisi / 密码: admin123' as '演示租户监护人';
SELECT '============================================' as '';
