-- IDP 模块数据库表结构

-- 微信应用表
CREATE TABLE IF NOT EXISTS `idp_wechat_apps` (
  `id` bigint unsigned NOT NULL COMMENT '主键 ID',
  `app_id` varchar(64) NOT NULL COMMENT '微信应用 ID',
  `name` varchar(255) NOT NULL COMMENT '应用名称',
  `type` varchar(32) NOT NULL COMMENT '应用类型：MiniProgram/MP',
  `status` varchar(32) NOT NULL DEFAULT 'Enabled' COMMENT '应用状态：Enabled/Disabled/Archived',
  
  -- 认证密钥（AppSecret）
  `auth_secret_cipher` blob COMMENT 'AppSecret 密文',
  `auth_secret_fp` varchar(128) COMMENT 'AppSecret 指纹（SHA256）',
  `auth_secret_version` int NOT NULL DEFAULT 0 COMMENT 'AppSecret 版本号',
  `auth_secret_rotated_at` datetime COMMENT 'AppSecret 最后轮换时间',
  
  -- 消息加解密密钥
  `msg_callback_token` varchar(128) COMMENT '消息推送回调 Token',
  `msg_aes_key_cipher` blob COMMENT '消息加密密钥密文',
  `msg_secret_version` int NOT NULL DEFAULT 0 COMMENT '消息密钥版本号',
  `msg_secret_rotated_at` datetime COMMENT '消息密钥最后轮换时间',
  
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_app_id` (`app_id`),
  KEY `idx_type` (`type`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='IDP 微信应用表';

-- 创建索引说明
-- uk_app_id: 确保 app_id 唯一性
-- idx_type: 按应用类型查询
-- idx_status: 按状态查询
-- idx_created_at: 按创建时间排序查询
