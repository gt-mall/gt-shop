-- 运费模板表及商品配送方式扩展
CREATE TABLE IF NOT EXISTS `freight_template` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '运费模板ID',
    `name` VARCHAR(100) NOT NULL COMMENT '模板名称',
    `base_fee` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '默认运费，0表示默认包邮',
    `extra_fees` JSON NULL COMMENT '地区附加运费',
    `no_delivery` JSON NULL COMMENT '不配送地区',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0禁用 1启用',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_0900_ai_ci
COMMENT='运费模板';

ALTER TABLE `product`
    ADD COLUMN `freight_type` TINYINT NOT NULL DEFAULT 1 COMMENT '配送方式：1包邮 2固定运费 3运费模板',
    ADD COLUMN `freight_fee` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '固定运费，仅 freight_type=2 使用',
    ADD COLUMN `freight_template_id` BIGINT NULL COMMENT '运费模板ID，仅 freight_type=3 使用';
