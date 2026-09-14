-- 019_after_sale_item_sku_code.sql
-- Expand sku_code length to 255 and backfill after_sale_item.sku_text from order_item

ALTER TABLE `after_sale_item`
    MODIFY `sku_code` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'sku code snapshot';

ALTER TABLE `order_item`
    MODIFY `sku_code` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'sku code';

UPDATE `after_sale_item` asi
JOIN `order_item` oi ON asi.order_item_id = oi.id
SET asi.sku_text = COALESCE(oi.sku_spec_values, '')
WHERE oi.sku_spec_values IS NOT NULL AND oi.sku_spec_values != '';
