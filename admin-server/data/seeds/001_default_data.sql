-- =============================================================================
-- Mall 商城系统默认初始数据（V1 Seed Data）
-- 字符集：utf8mb4 / 校对规则：utf8mb4_0900_ai_ci
-- 说明：仅保存系统首次安装必须的基础数据（默认管理员、内置角色、默认文件分组、系统基础配置）。
-- 注意：不包含测试商品、测试订单或演示数据；所有语句具备幂等性，可重复执行。
-- =============================================================================

-- =============================================================================
-- 1. 系统内置角色
-- =============================================================================
INSERT INTO `system_role` (`name`, `code`, `status`, `sort`, `remark`)
VALUES ('超级管理员', 'super_admin', 1, 0, '系统内置角色')
ON DUPLICATE KEY UPDATE
    `name` = VALUES(`name`),
    `status` = VALUES(`status`),
    `sort` = VALUES(`sort`),
    `remark` = VALUES(`remark`);

-- 默认管理员仅在账号不存在时创建；重复执行不会覆盖用户已经修改的密码。
-- 初始账号：admin，初始密码：123456。登录后由管理后台持续提醒修改密码。
INSERT INTO `system_user` (`username`, `password_hash`, `nickname`, `status`)
VALUES ('admin', '$2b$10$pEkfvLI/ocT0Y1/0tJeV3OkbTlvcq5GquUU2uI01FNfjwzSaXxqcm', '超级管理员', 1)
ON DUPLICATE KEY UPDATE
    `username` = VALUES(`username`);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'g', CONCAT('user:', `id`), 'super_admin', '', '', '', ''
FROM `system_user`
WHERE `username` = 'admin'
ON DUPLICATE KEY UPDATE
    `ptype` = VALUES(`ptype`);

-- =============================================================================
-- 2. 系统默认文件分组
-- =============================================================================
INSERT INTO `system_file_group` (`name`)
VALUES
    ('商品图片'),
    ('分类图片'),
    ('广告图片'),
    ('装修图片')
ON DUPLICATE KEY UPDATE
    `name` = VALUES(`name`);

-- =============================================================================
-- 3. 系统基础配置
-- =============================================================================
INSERT INTO `system_config` (`config_key`, `value`)
VALUES
    ('file_storage', '{"default_storage":"local","local":{"path":"uploads","domain":""},"oss":{},"cos":{},"s3":{}}'),
    ('shop_basic', '{"name":"Mall 商城","logo":"","logo_id":0,"favicon":"","favicon_id":0,"service_name":"商城客服","service_phone":"","service_wechat":"","service_wechat_qrcode":"","service_wechat_qrcode_id":0,"description":"全场景高性能电商系统"}'),
    ('site_basic', '{"title":"Mall 管理后台","keywords":"Mall,电商系统,管理后台","description":"Mall 电商管理系统","icp":"","copyright":"© 2026 Mall. All rights reserved."}'),
    ('mp_basic', '{"name":"Mall 小程序","original_id":"","qrcode":"","qrcode_id":0,"app_id":"","app_secret":"","request_domain":"","socket_domain":"","upload_domain":"","download_domain":"","udp_domain":"","tcp_domain":"","business_domain":"","upload_private_key":""}'),
    ('trade_basic', '{"order_close_minutes":30,"auto_receive_days":7,"after_sale_days":7,"allow_cancel_unshipped":true,"auto_refund_on_cancel":true}'),
    ('payment_basic', '{"wechat_enabled":true}'),
    ('logistics_basic', '{"sender_name":"","sender_phone":"","sender_address":""}'),
    ('member_basic', '{"register_enabled":true,"password_login_enabled":true,"wechat_login_enabled":true}'),
    ('review_basic', '{"enabled":true,"need_audit":true,"default_anonymous":false,"auto_review_enabled":false,"auto_review_days":7}'),
    ('agreement_basic', '{"user_agreement":"","privacy_policy":"","after_sale_policy":""}')
ON DUPLICATE KEY UPDATE
    `config_key` = VALUES(`config_key`);
