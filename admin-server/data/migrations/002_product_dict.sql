-- =============================================================================
-- 002_product_dict.sql
-- 商品字典维护数据表：服务保障表、商品单位表
-- =============================================================================

-- 9.1 商品服务保障表
CREATE TABLE IF NOT EXISTS `product_dict_service` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '服务保障ID',
    `name` VARCHAR(64) NOT NULL COMMENT '服务保障名称',
    `description` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '服务保障说明',
    `sort` INT NOT NULL DEFAULT 0 COMMENT '排序值，越小越靠前',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0禁用，1启用',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_dict_service_sort` (`sort`),
    KEY `idx_dict_service_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品服务保障字典表';

-- 9.2 商品单位表
CREATE TABLE IF NOT EXISTS `product_dict_unit` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '商品单位ID',
    `name` VARCHAR(32) NOT NULL COMMENT '商品单位名称',
    `description` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '单位说明',
    `sort` INT NOT NULL DEFAULT 0 COMMENT '排序值，越小越靠前',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0禁用，1启用',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_dict_unit_sort` (`sort`),
    KEY `idx_dict_unit_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品单位字典表';
