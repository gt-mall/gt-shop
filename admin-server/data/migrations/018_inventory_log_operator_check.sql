-- 018_inventory_log_operator_check.sql
-- Allow system operator (operator_id = 0) in inventory log CHECK constraint

ALTER TABLE `product_sku_inventory_log` DROP CHECK `chk_inventory_log_owner`;

ALTER TABLE `product_sku_inventory_log` ADD CONSTRAINT `chk_inventory_log_owner` CHECK (
    `inventory_id` > 0 AND `product_id` > 0 AND `sku_id` > 0 AND `operator_id` >= 0
);
