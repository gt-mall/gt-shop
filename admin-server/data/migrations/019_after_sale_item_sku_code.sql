-- 019_after_sale_item_sku_code.sql
-- Expand sku_code to 255 and sku_text to 512 for after_sale_item and order_item

ALTER TABLE `after_sale_item`
    MODIFY `sku_code` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'sku code snapshot',
    MODIFY `sku_text` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'sku spec text snapshot';

ALTER TABLE `order_item`
    MODIFY `sku_code` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'sku code';
