-- 商品库存第一版：单仓库、整数件数；规划 SQL，不会由文档任务自动执行。
-- 目标：MySQL 8.4 / InnoDB，使用默认强制执行的 CHECK 约束。
-- 执行前核对真实 product/product_sku 结构；不要重建现有商品三张表。
-- A 段建表只执行一次；表已存在时先 SHOW CREATE TABLE 比较，再编写增量迁移。
-- B 段只补缺失的零库存，不能用它导入实际期初库存或覆盖已有余额。
-- 发布时短暂暂停商品/SKU 写入，完成建表、回填和新代码部署后恢复写入。

-- A.1 每个 SKU 唯一的库存余额
CREATE TABLE `product_sku_inventory` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '库存记录ID',
    `product_id` BIGINT NOT NULL COMMENT '商品ID，由服务端从SKU派生',
    `sku_id` BIGINT NOT NULL COMMENT 'SKU ID，一对一关联product_sku.id',
    `stock` INT NOT NULL DEFAULT 0 COMMENT '账面实物库存，包含已锁定数量',
    `locked_stock` INT NOT NULL DEFAULT 0 COMMENT '锁定库存，属于stock的子集；本期只初始化为0',
    `warning_stock` INT NOT NULL DEFAULT 0 COMMENT '可用库存预警阈值，0表示不启用预警',
    `version` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '每次成功变更递增1，API可精确使用number',
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_inventory_sku` (`sku_id`),
    KEY `idx_inventory_product_sku` (`product_id`, `sku_id`),
    CONSTRAINT `chk_inventory_owner` CHECK (`product_id` > 0 AND `sku_id` > 0),
    CONSTRAINT `chk_inventory_stock` CHECK (`stock` >= 0),
    CONSTRAINT `chk_inventory_locked` CHECK (`locked_stock` >= 0 AND `locked_stock` <= `stock`),
    CONSTRAINT `chk_inventory_warning` CHECK (`warning_stock` >= 0)
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_0900_ai_ci
COMMENT='SKU库存余额，单仓库；可用库存=stock-locked_stock';

-- A.2 只追加的库存/预警变更流水
CREATE TABLE `product_sku_inventory_log` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '库存流水ID',
    `inventory_id` BIGINT NOT NULL COMMENT '库存记录ID',
    `product_id` BIGINT NOT NULL COMMENT '商品ID快照',
    `sku_id` BIGINT NOT NULL COMMENT 'SKU ID快照',
    `request_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT '小写UUID，一次操作一个幂等键',
    `request_hash` CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT '规范化请求和操作者的SHA256',
    `change_type` VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT 'manual_in/manual_out/stocktake/warning_set',
    `quantity` INT NOT NULL COMMENT '原始操作数量；盘点为目标总库存，warning_set为目标阈值',
    `stock_before` INT NOT NULL COMMENT '变更前总库存',
    `stock_delta` INT NOT NULL COMMENT '总库存变化量，可正可负或为0',
    `stock_after` INT NOT NULL COMMENT '变更后总库存',
    `locked_before` INT NOT NULL COMMENT '变更前锁定库存',
    `locked_delta` INT NOT NULL COMMENT '锁定库存变化量，本期为0',
    `locked_after` INT NOT NULL COMMENT '变更后锁定库存',
    `warning_before` INT NOT NULL COMMENT '变更前预警阈值',
    `warning_after` INT NOT NULL COMMENT '变更后预警阈值',
    `version_before` INT UNSIGNED NOT NULL COMMENT '变更前版本，也是原请求expected_version',
    `version_after` INT UNSIGNED NOT NULL COMMENT '变更后版本',
    `sku_snapshot` JSON NOT NULL COMMENT '当时的商品名、SKU编码、规格名称和单位快照',
    `operator_id` BIGINT NOT NULL COMMENT '操作者ID，后台操作从登录上下文取',
    `operator_name` VARCHAR(100) NOT NULL COMMENT '操作者昵称或用户名快照',
    `remark` VARCHAR(500) NOT NULL COMMENT '操作原因，后台提交必填',
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '记录时间，不可修改',

    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_inventory_log_request` (`request_id`),
    UNIQUE KEY `uk_inventory_log_version` (`inventory_id`, `version_after`),
    KEY `idx_inventory_log_sku_time` (`sku_id`, `created_at`, `id`),
    KEY `idx_inventory_log_product_time` (`product_id`, `created_at`, `id`),
    KEY `idx_inventory_log_time` (`created_at`, `id`),
    CONSTRAINT `chk_inventory_log_owner` CHECK (
        `inventory_id` > 0 AND `product_id` > 0 AND `sku_id` > 0 AND `operator_id` >= 0
    ),
    CONSTRAINT `chk_inventory_log_quantity` CHECK (`quantity` >= 0),
    CONSTRAINT `chk_inventory_log_stock` CHECK (
        `stock_before` >= 0 AND `stock_after` >= 0
        AND `stock_after` = `stock_before` + `stock_delta`
    ),
    CONSTRAINT `chk_inventory_log_locked` CHECK (
        `locked_before` >= 0 AND `locked_after` >= 0
        AND `locked_before` <= `stock_before` AND `locked_after` <= `stock_after`
        AND `locked_after` = `locked_before` + `locked_delta`
    ),
    CONSTRAINT `chk_inventory_log_warning` CHECK (`warning_before` >= 0 AND `warning_after` >= 0),
    CONSTRAINT `chk_inventory_log_version` CHECK (`version_after` = `version_before` + 1)
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_0900_ai_ci
COMMENT='SKU库存变更流水，只追加；商品或SKU删除后仍保留';

-- B. 为存量 SKU 补零库存；包括禁用/软删除 SKU，保证以后恢复时仍按原 ID 关联。
-- product 使用内连接检查父记录存在，不按 deleted_at 排除历史商品。
-- 没有任何存量库存来源时初始化为0；真实期初库存必须走带流水的人工入库。
-- 只补不存在的库存记录；不使用 REPLACE、INSERT IGNORE 或重复键覆盖余额。
INSERT INTO `product_sku_inventory`
    (`product_id`, `sku_id`, `stock`, `locked_stock`, `warning_stock`, `version`)
SELECT s.`product_id`, s.`id`, 0, 0, 0, 0
FROM `product_sku` AS s
INNER JOIN `product` AS p ON p.`id` = s.`product_id`
LEFT JOIN `product_sku_inventory` AS i ON i.`sku_id` = s.`id`
WHERE i.`id` IS NULL
ORDER BY s.`id`;

-- C. 只读核对：以下两个结果都应为0。异常记录需要定位，不能静默当成0库存。
SELECT COUNT(*) AS `missing_inventory_count`
FROM `product_sku` AS s
LEFT JOIN `product_sku_inventory` AS i ON i.`sku_id` = s.`id`
WHERE i.`id` IS NULL;

SELECT COUNT(*) AS `inventory_owner_mismatch_count`
FROM `product_sku_inventory` AS i
LEFT JOIN `product_sku` AS s ON s.`id` = i.`sku_id`
LEFT JOIN `product` AS p ON p.`id` = i.`product_id`
WHERE s.`id` IS NULL OR p.`id` IS NULL OR i.`product_id` <> s.`product_id`;
