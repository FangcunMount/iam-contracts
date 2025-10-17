-- Migration: Create jwks_keys table
-- Description: 存储 JWKS 密钥的生命周期管理数据
-- Author: System
-- Date: 2025-01-17

CREATE TABLE IF NOT EXISTS `jwks_keys` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `kid` varchar(64) NOT NULL COMMENT '密钥唯一标识符（Key ID）',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '密钥状态：1=Active（激活），2=Grace（宽限期），3=Retired（已退役）',
  `kty` varchar(32) NOT NULL COMMENT '密钥类型：RSA, EC, etc',
  `use` varchar(16) NOT NULL COMMENT '密钥用途：sig=签名, enc=加密',
  `alg` varchar(32) NOT NULL COMMENT '算法标识：RS256, RS384, RS512, etc',
  `jwk_json` json NOT NULL COMMENT '完整的公钥 JWK JSON（包含 n, e 等参数）',
  `not_before` datetime DEFAULT NULL COMMENT '密钥生效时间（可选）',
  `not_after` datetime DEFAULT NULL COMMENT '密钥过期时间（可选）',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除时间',
  `created_by` varchar(50) DEFAULT NULL COMMENT '创建人ID',
  `updated_by` varchar(50) DEFAULT NULL COMMENT '更新人ID',
  `deleted_by` varchar(50) DEFAULT NULL COMMENT '删除人ID',
  `version` int unsigned NOT NULL DEFAULT '1' COMMENT '乐观锁版本号',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_kid` (`kid`),
  KEY `idx_status` (`status`),
  KEY `idx_alg` (`alg`),
  KEY `idx_not_after` (`not_after`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='JWKS 密钥表';

-- 索引说明：
-- 1. idx_kid: 唯一索引，用于快速查询单个密钥
-- 2. idx_status: 用于按状态查询（FindByStatus, FindPublishable）
-- 3. idx_alg: 用于按算法过滤
-- 4. idx_not_after: 用于查找过期密钥（FindExpired）
-- 5. idx_deleted_at: 软删除支持

-- 使用示例：
-- INSERT INTO jwks_keys (kid, status, kty, use, alg, jwk_json, not_before, not_after)
-- VALUES ('key-2025-01', 1, 'RSA', 'sig', 'RS256', 
--   '{"kty":"RSA","use":"sig","kid":"key-2025-01","alg":"RS256","n":"...","e":"AQAB"}',
--   '2025-01-01 00:00:00', '2025-12-31 23:59:59');
