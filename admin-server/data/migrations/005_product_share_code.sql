-- =============================================================================
-- 005_product_share_code.sql
-- 商品分享小程序码持久化映射表
-- =============================================================================

CREATE TABLE IF NOT EXISTS `product_share_code` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `app_id` VARCHAR(64) NOT NULL COMMENT '小程序AppID',
    `env` VARCHAR(32) NOT NULL DEFAULT 'release' COMMENT '小程序版本环境：release正式版，trial体验版，develop开发版',
    `product_id` BIGINT NOT NULL COMMENT '商品ID',
    `scene_version` VARCHAR(32) NOT NULL DEFAULT 'v1' COMMENT '场景参数协议版本',
    `file_id` BIGINT NOT NULL COMMENT '系统文件ID',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_share_code_identity` (`app_id`, `env`, `product_id`, `scene_version`),
    KEY `idx_share_code_product` (`product_id`),
    KEY `idx_share_code_file` (`file_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品小程序分享码映射表';
