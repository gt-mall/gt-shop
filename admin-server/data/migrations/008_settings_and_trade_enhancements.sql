-- =============================================================================
-- 008_settings_and_trade_enhancements.sql
-- 1. 新增退货地址表 (order_return_address)
-- 2. 订单表增加支付过期时间、自动收货时间、售后截止时间字段
-- 3. 初始配置项补充（交易、支付、配送、用户、评价、协议）
-- =============================================================================

-- 1. 退货地址表
CREATE TABLE IF NOT EXISTS `order_return_address` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '退货地址ID',
    `name` VARCHAR(64) NOT NULL COMMENT '地址名称/仓库名称',
    `receiver` VARCHAR(64) NOT NULL COMMENT '收件人姓名',
    `phone` VARCHAR(32) NOT NULL COMMENT '联系电话',
    `province_code` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '省份编码',
    `city_code` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '城市编码',
    `district_code` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '区县编码',
    `address` VARCHAR(255) NOT NULL COMMENT '详细地址',
    `is_default` TINYINT NOT NULL DEFAULT 0 COMMENT '是否默认：1是 0否',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1启用 0禁用',
    `sort` INT NOT NULL DEFAULT 0 COMMENT '排序值，越小越靠前',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_return_addr_status` (`status`),
    KEY `idx_return_addr_default` (`is_default`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='退货地址表';

-- 2. 订单表增加过期与截止时间字段
ALTER TABLE `order_main`
    ADD COLUMN `expire_at` DATETIME(6) NULL DEFAULT NULL COMMENT '订单支付截止过期时间' AFTER `cancelled_at`,
    ADD COLUMN `auto_receive_at` DATETIME(6) NULL DEFAULT NULL COMMENT '订单自动确认收货时间' AFTER `expire_at`,
    ADD COLUMN `after_sale_deadline` DATETIME(6) NULL DEFAULT NULL COMMENT '售后申请截止时间' AFTER `auto_receive_at`,
    ADD INDEX `idx_order_main_expire_at` (`expire_at`),
    ADD INDEX `idx_order_main_auto_receive` (`status`, `auto_receive_at`);

-- 3. 系统配置默认数据补充（幂等写入）
INSERT INTO `system_config` (`config_key`, `value`)
VALUES
    ('trade_basic', '{"order_close_minutes":30,"auto_receive_days":7,"after_sale_days":7,"allow_cancel_unshipped":true,"auto_refund_on_cancel":true}'),
    ('payment_basic', '{"wechat_enabled":true}'),
    ('logistics_basic', '{"sender_name":"","sender_phone":"","sender_address":""}'),
    ('member_basic', '{"register_enabled":true,"password_login_enabled":true,"wechat_login_enabled":true}'),
    ('review_basic', '{"enabled":true,"need_audit":true,"default_anonymous":false,"auto_review_enabled":false,"auto_review_days":7}'),
    ('agreement_basic', '{"user_agreement":"","privacy_policy":"","after_sale_policy":""}')
ON DUPLICATE KEY UPDATE
    `config_key` = VALUES(`config_key`);
