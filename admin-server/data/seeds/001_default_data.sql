-- =============================================================================
-- Mall 商城系统默认初始数据（V1 Seed Data）
-- 字符集：utf8mb4 / 校对规则：utf8mb4_0900_ai_ci
-- 说明：仅保存系统首次安装必须的基础数据（内置角色、默认文件分组、系统基础配置）。
-- 注意：不包含测试用户、测试商品、测试订单或演示数据；所有语句具备幂等性，可重复执行。
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
    ('mp_basic', '{"name":"Mall 小程序","original_id":"","qrcode":"","qrcode_id":0,"app_id":"","app_secret":"","request_domain":"","socket_domain":"","upload_domain":"","download_domain":"","udp_domain":"","tcp_domain":"","business_domain":"","upload_private_key":""}')
ON DUPLICATE KEY UPDATE
    `config_key` = VALUES(`config_key`);
