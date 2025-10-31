-- 初始化数据库 Schema
-- 此文件从 configs/mysql/schema.sql 生成
-- 版本: 3.0.0
-- 日期: 2025-10-31

-- 创建数据库
CREATE DATABASE IF NOT EXISTS `iam_contracts`
    DEFAULT CHARACTER SET utf8mb4
    DEFAULT COLLATE utf8mb4_unicode_ci;

USE `iam_contracts`;

-- ============================================================================
-- Module 1: User Center (UC) 用户中心
-- ============================================================================

-- 1.1 用户表
CREATE TABLE IF NOT EXISTS `iam_users`
(
    `id`         BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '用户ID',
    `name`       VARCHAR(64)     NOT NULL COMMENT '用户名称',
    `phone`      VARCHAR(20)     NOT NULL COMMENT '手机号',
    `email`      VARCHAR(100)    NOT NULL COMMENT '邮箱',
    `id_card`    VARCHAR(20)     NOT NULL COMMENT '身份证号',
    `status`     INT             NOT NULL DEFAULT 1 COMMENT '用户状态: 1-正常, 2-禁用',
    `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME                 DEFAULT NULL COMMENT '删除时间',
    `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    `version`    INT UNSIGNED    NOT NULL DEFAULT 1 COMMENT '乐观锁版本号',
    UNIQUE KEY `uk_id_card` (`id_card`),
    KEY `idx_phone` (`phone`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='用户表';

-- 注意: 为了文件大小，这里只展示部分表结构
-- 完整版本请从 configs/mysql/schema.sql 复制所有表定义

-- ... 其他表定义 ...
