-- 012_order_optimize.sql
-- 统一订单金额字段为 DECIMAL(12,2)，并为 order_log 增加状态变更记录字段

ALTER TABLE order_main
    MODIFY COLUMN total_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    MODIFY COLUMN pay_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    MODIFY COLUMN freight_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    MODIFY COLUMN discount_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00;

ALTER TABLE order_item
    MODIFY COLUMN unit_price DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    MODIFY COLUMN total_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00;

ALTER TABLE order_log
    ADD COLUMN from_status TINYINT NOT NULL DEFAULT 0 AFTER order_id,
    ADD COLUMN to_status TINYINT NOT NULL DEFAULT 0 AFTER from_status;
