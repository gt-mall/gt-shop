-- =============================================================================
-- Mall 商城系统正式数据库基线（V1 Baseline）
-- 字符集：utf8mb4 / 校对规则：utf8mb4_0900_ai_ci / 存储引擎：InnoDB
-- 说明：包含系统全部26张核心业务表完整结构、主键、唯一键、索引、CHECK约束及中文注释。
-- 注意：本脚本仅负责数据库表结构创建，不包含默认初始化业务数据。
-- =============================================================================

-- =============================================================================
-- 1. 系统与权限模块 (System & RBAC)
-- =============================================================================

-- 1.1 系统用户表
CREATE TABLE `system_user` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '系统用户ID',
    `username` VARCHAR(64) NOT NULL COMMENT '登录用户名',
    `password_hash` VARCHAR(255) NOT NULL COMMENT '密码哈希值',
    `nickname` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '用户昵称',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '用户状态：0禁用，1启用',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='系统用户表';

-- 1.2 系统角色表
CREATE TABLE `system_role` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '系统角色ID',
    `name` VARCHAR(64) NOT NULL COMMENT '角色名称',
    `code` VARCHAR(64) NOT NULL COMMENT '角色编码',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '角色状态：0禁用，1启用',
    `sort` INT NOT NULL DEFAULT 0 COMMENT '排序权重，数值越小越靠前',
    `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '角色备注说明',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='系统角色表';

-- 1.3 Casbin权限规则表
CREATE TABLE `casbin_rule` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '规则ID',
    `ptype` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '规则类型：p为权限策略，g为角色继承映射',
    `v0` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '主体标识：角色编码或用户标识(user:id)',
    `v1` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '资源路径或角色编码',
    `v2` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '操作权限动作或保留参数',
    `v3` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '保留扩展参数3',
    `v4` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '保留扩展参数4',
    `v5` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '保留扩展参数5',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Casbin权限规则表';


-- =============================================================================
-- 2. 系统资源与配置模块 (System Resources & Config)
-- =============================================================================

-- 2.1 系统文件分组表
CREATE TABLE `system_file_group` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '文件分组ID',
    `name` VARCHAR(64) NOT NULL COMMENT '分组名称',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_file_group_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='系统文件分组表';

-- 2.2 系统文件资源表
CREATE TABLE `system_file` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '文件ID',
    `group_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '文件分组ID，0表示未分组',
    `storage` VARCHAR(16) NOT NULL COMMENT '存储引擎：local本地、oss阿里云、cos腾讯云、s3亚马逊',
    `name` VARCHAR(255) NOT NULL COMMENT '文件显示名称',
    `original_name` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '上传时的原始文件名',
    `path` VARCHAR(255) NOT NULL COMMENT '文件存储相对路径，格式：YYYY/MM/UUID.ext',
    `ext` VARCHAR(10) NOT NULL COMMENT '文件扩展名（不含点），如jpg/png',
    `mime_type` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '文件MIME媒体类型，如image/jpeg',
    `file_type` VARCHAR(32) NOT NULL DEFAULT 'image' COMMENT '文件类型：image图片、video视频、audio音频、document文档、archive压缩包、other其他',
    `scene` VARCHAR(64) NOT NULL DEFAULT 'admin_material' COMMENT '业务场景：admin_material后台素材、after_sale_proof售后凭证、member_avatar会员头像、system_generated系统生成、review评价',
    `owner_type` VARCHAR(32) NOT NULL DEFAULT 'admin' COMMENT '所有者类型：admin管理员、member会员、system系统',
    `owner_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所有者ID',
    `visibility` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '访问权限：1公开访问，2私有鉴权',
    `size` BIGINT UNSIGNED NOT NULL COMMENT '文件大小，单位：字节(B)',
    `width` INT UNSIGNED NOT NULL COMMENT '图片或视频宽度，单位：像素',
    `height` INT UNSIGNED NOT NULL COMMENT '图片或视频高度，单位：像素',
    `duration_ms` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '音视频时长，单位：毫秒',
    `cover_file_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '视频封面图片文件ID，关联system_file.id',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `deleted_at` DATETIME(6) NULL DEFAULT NULL COMMENT '软删除时间，NULL表示未删除',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_file_storage_path` (`storage`, `path`),
    KEY `idx_file_group` (`group_id`, `deleted_at`, `id`),
    KEY `idx_file_list` (`deleted_at`, `id`),
    KEY `idx_file_group_stat` (`deleted_at`, `group_id`, `size`),
    KEY `idx_file_scene_type` (`scene`, `file_type`),
    KEY `idx_file_owner` (`owner_type`, `owner_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='系统文件资源表';

-- 2.3 系统配置表
CREATE TABLE `system_config` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '配置ID',
    `config_key` VARCHAR(64) NOT NULL COMMENT '配置键名',
    `value` JSON NOT NULL COMMENT '配置值JSON',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_system_config_key` (`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='系统配置表';


-- =============================================================================
-- 3. 商品与运费模块 (Product & Freight)
-- =============================================================================

-- 3.1 商品分类表
CREATE TABLE `product_category` (
    `id` INT NOT NULL AUTO_INCREMENT COMMENT '分类ID',
    `parent_id` INT NOT NULL DEFAULT 0 COMMENT '父分类ID，0表示顶级分类',
    `name` VARCHAR(255) NOT NULL COMMENT '分类名称',
    `image_id` BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '分类图片文件ID，关联system_file.id',
    `sort` INT NOT NULL DEFAULT 0 COMMENT '排序权重，数值越小越靠前',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '分类状态：0禁用，1启用',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    `deleted_at` DATETIME NULL DEFAULT NULL COMMENT '软删除时间，NULL表示未删除',
    PRIMARY KEY (`id`),
    KEY `idx_parent_id` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品分类表';

-- 3.2 运费模板表
CREATE TABLE `freight_template` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '运费模板ID',
    `name` VARCHAR(100) NOT NULL COMMENT '模板名称',
    `base_fee` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '默认基础运费，单位：元，0表示默认包邮',
    `extra_fees` JSON NULL DEFAULT NULL COMMENT '指定地区附加运费配置JSON',
    `no_delivery` JSON NULL DEFAULT NULL COMMENT '不配送地区配置JSON',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '模板状态：0禁用，1启用',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    `deleted_at` DATETIME NULL DEFAULT NULL COMMENT '软删除时间，NULL表示未删除',
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='运费模板表';

-- 3.3 商品主表
CREATE TABLE `product` (
    `id` INT NOT NULL AUTO_INCREMENT COMMENT '商品ID',
    `category_id` INT NOT NULL DEFAULT 0 COMMENT '商品分类ID',
    `name` VARCHAR(255) NOT NULL COMMENT '商品名称',
    `subtitle` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '商品副标题/卖点',
    `main_image_id` BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '商品主图文件ID，关联system_file.id',
    `gallery_file_ids` JSON NULL DEFAULT NULL COMMENT '商品轮播图片文件ID列表JSON数组',
    `specs` JSON NULL DEFAULT NULL COMMENT '商品规格定义JSON，包含规格项及规格值列表',
    `params` JSON NULL DEFAULT NULL COMMENT '商品参数属性JSON，包含[{name, value}]键值对列表',
    `detail` LONGTEXT NULL DEFAULT NULL COMMENT '商品富文本详情HTML内容',
    `status` TINYINT NOT NULL DEFAULT 0 COMMENT '商品状态：0下架，1上架',
    `sort` INT NOT NULL DEFAULT 0 COMMENT '排序权重，数值越小越靠前',
    `sales_count` BIGINT NOT NULL DEFAULT 0 COMMENT '实际销量数量',
    `virtual_sales` INT NOT NULL DEFAULT 0 COMMENT '虚拟销量展示数量',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    `deleted_at` DATETIME NULL DEFAULT NULL COMMENT '软删除时间，NULL表示未删除',
    `freight_type` TINYINT NOT NULL DEFAULT 1 COMMENT '运费类型：1包邮，2固定运费，3运费模板',
    `freight_fee` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '固定运费，单位：元，仅freight_type=2有效',
    `freight_template_id` BIGINT NULL DEFAULT NULL COMMENT '运费模板ID，关联freight_template.id，仅freight_type=3有效',
    PRIMARY KEY (`id`),
    KEY `idx_category_id` (`category_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品主表';

-- 3.4 商品SKU表
CREATE TABLE `product_sku` (
    `id` INT NOT NULL AUTO_INCREMENT COMMENT 'SKU ID',
    `product_id` INT NOT NULL COMMENT '商品ID',
    `spec_values` JSON NOT NULL COMMENT 'SKU规格值组合JSON，包含spec_id/spec_name/value_id/value_name',
    `spec_hash` CHAR(64) NOT NULL COMMENT '规格组合SHA256哈希值，用于唯一性校验',
    `sku_code` VARCHAR(255) NULL DEFAULT NULL COMMENT 'SKU编码',
    `barcode` VARCHAR(255) NULL DEFAULT NULL COMMENT '商品国际条形码',
    `image_id` BIGINT UNSIGNED NULL DEFAULT NULL COMMENT 'SKU图片文件ID，关联system_file.id',
    `price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '销售价格，单位：元',
    `market_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '划线市场价，单位：元',
    `cost_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '成本价格，单位：元',
    `unit` VARCHAR(255) NULL DEFAULT NULL COMMENT '销售单位，如件/个/瓶/盒/箱/套',
    `weight_gram` INT NULL DEFAULT NULL COMMENT '物流计费重量，单位：克',
    `sort` INT NOT NULL DEFAULT 0 COMMENT '排序权重，数值越小越靠前',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT 'SKU状态：0禁用，1启用',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
    `deleted_at` DATETIME NULL DEFAULT NULL COMMENT '软删除时间，NULL表示未删除',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_product_spec` (`product_id`, `spec_hash`),
    UNIQUE KEY `uk_sku_code` (`sku_code`),
    KEY `idx_product_id` (`product_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品SKU表';


-- =============================================================================
-- 4. 库存管理模块 (Inventory)
-- =============================================================================

-- 4.1 SKU库存表
CREATE TABLE `product_sku_inventory` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '库存记录ID',
    `product_id` BIGINT NOT NULL COMMENT '商品ID',
    `sku_id` BIGINT NOT NULL COMMENT 'SKU ID，一对一关联product_sku.id',
    `stock` INT NOT NULL DEFAULT 0 COMMENT '实际库存数量（包含锁定库存）',
    `locked_stock` INT NOT NULL DEFAULT 0 COMMENT '订单锁定库存数量',
    `warning_stock` INT NOT NULL DEFAULT 0 COMMENT '可用库存预警阈值，0表示未启用预警',
    `version` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '乐观锁版本号，每次变动递增1',
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_inventory_sku` (`sku_id`),
    KEY `idx_inventory_product_sku` (`product_id`, `sku_id`),
    CONSTRAINT `chk_inventory_owner` CHECK (`product_id` > 0 AND `sku_id` > 0),
    CONSTRAINT `chk_inventory_stock` CHECK (`stock` >= 0),
    CONSTRAINT `chk_inventory_locked` CHECK (`locked_stock` >= 0 AND `locked_stock` <= `stock`),
    CONSTRAINT `chk_inventory_warning` CHECK (`warning_stock` >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='SKU库存表';

-- 4.2 SKU库存变动日志表
CREATE TABLE `product_sku_inventory_log` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '库存流水ID',
    `inventory_id` BIGINT NOT NULL COMMENT '库存记录ID',
    `product_id` BIGINT NOT NULL COMMENT '商品ID快照',
    `sku_id` BIGINT NOT NULL COMMENT 'SKU ID快照',
    `request_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT '客户端幂等请求UUID',
    `request_hash` CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT '规范化请求和操作者的SHA256哈希值',
    `change_type` VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT '变动类型：manual_in入库，manual_out出库，stocktake盘点，warning_set预警设置，order_lock锁定，order_unlock解锁，order_deduct扣减，after_sale_restock售后重新入库',
    `quantity` INT NOT NULL COMMENT '原始操作数值：盘点为目标总库存，预警为目标阈值，出入库为操作件数',
    `stock_before` INT NOT NULL COMMENT '变动前总库存',
    `stock_delta` INT NOT NULL COMMENT '总库存变动量，可正可负或为0',
    `stock_after` INT NOT NULL COMMENT '变动后总库存',
    `locked_before` INT NOT NULL COMMENT '变动前锁定库存',
    `locked_delta` INT NOT NULL COMMENT '锁定库存变动量，可正可负或为0',
    `locked_after` INT NOT NULL COMMENT '变动后锁定库存',
    `warning_before` INT NOT NULL COMMENT '变动前预警阈值',
    `warning_after` INT NOT NULL COMMENT '变动后预警阈值',
    `version_before` INT UNSIGNED NOT NULL COMMENT '变动前版本号',
    `version_after` INT UNSIGNED NOT NULL COMMENT '变动后版本号',
    `sku_snapshot` JSON NOT NULL COMMENT '当时的商品名、SKU编码、规格名称和单位快照JSON',
    `operator_id` BIGINT NOT NULL COMMENT '操作人ID：0表示系统操作，>0表示实际操作用户',
    `operator_name` VARCHAR(100) NOT NULL COMMENT '操作人姓名快照',
    `remark` VARCHAR(500) NOT NULL COMMENT '操作原因说明备注',
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '记录创建时间',
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
        AND `stock_after` = (`stock_before` + `stock_delta`)
    ),
    CONSTRAINT `chk_inventory_log_locked` CHECK (
        `locked_before` >= 0 AND `locked_after` >= 0
        AND `locked_before` <= `stock_before` AND `locked_after` <= `stock_after`
        AND `locked_after` = (`locked_before` + `locked_delta`)
    ),
    CONSTRAINT `chk_inventory_log_warning` CHECK (`warning_before` >= 0 AND `warning_after` >= 0),
    CONSTRAINT `chk_inventory_log_version` CHECK (`version_after` = (`version_before` + 1))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='SKU库存变动日志表';


-- =============================================================================
-- 5. 会员模块 (Member)
-- =============================================================================

-- 5.1 会员主表
CREATE TABLE `member` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '会员ID',
    `nickname` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '会员昵称',
    `avatar_id` BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '头像文件ID，关联system_file.id',
    `mobile` VARCHAR(255) NULL DEFAULT NULL COMMENT '会员手机号',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '会员状态：0禁用，1启用',
    `last_login_at` DATETIME NULL DEFAULT NULL COMMENT '最后登录时间',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    `deleted_at` DATETIME(6) NULL DEFAULT NULL COMMENT '软删除时间，NULL表示未删除',
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='会员主表';

-- 5.2 会员第三方授权表
CREATE TABLE `member_auth` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '授权记录ID',
    `member_id` BIGINT NOT NULL COMMENT '会员ID',
    `auth_type` VARCHAR(255) NOT NULL COMMENT '授权类型：wechat_mp微信小程序、wechat_app微信App等',
    `app_id` VARCHAR(255) NOT NULL COMMENT '第三方平台应用ID',
    `openid` VARCHAR(255) NOT NULL COMMENT '第三方平台唯一标识OpenID',
    `unionid` VARCHAR(255) NULL DEFAULT NULL COMMENT '第三方平台开放平台统一标识UnionID',
    `provider_data` JSON NULL DEFAULT NULL COMMENT '第三方平台原始授权信息JSON',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_auth_type_app_openid` (`auth_type`, `app_id`, `openid`),
    KEY `idx_member_auth_member_id` (`member_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='会员第三方授权表';

-- 5.3 会员账号表
CREATE TABLE `member_account` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '账号记录ID',
    `member_id` BIGINT NOT NULL COMMENT '会员ID',
    `account_type` VARCHAR(255) NOT NULL COMMENT '账号类型：username用户名、mobile手机号、email邮箱',
    `account` VARCHAR(255) NOT NULL COMMENT '账号登录名',
    `password_hash` VARCHAR(255) NOT NULL COMMENT '密码哈希值',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_account_type_account` (`account_type`, `account`),
    KEY `idx_member_account_member_id` (`member_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='会员账号表';

-- 5.4 会员收货地址表
CREATE TABLE `member_address` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '收货地址ID',
    `member_id` BIGINT NOT NULL COMMENT '会员ID',
    `name` VARCHAR(255) NOT NULL COMMENT '收货人姓名',
    `mobile` VARCHAR(255) NOT NULL COMMENT '收货人手机号',
    `province_code` VARCHAR(255) NOT NULL COMMENT '省份行政编码',
    `province` VARCHAR(255) NOT NULL COMMENT '省份名称',
    `city_code` VARCHAR(255) NOT NULL COMMENT '城市行政编码',
    `city` VARCHAR(255) NOT NULL COMMENT '城市名称',
    `district_code` VARCHAR(255) NOT NULL COMMENT '区县行政编码',
    `district` VARCHAR(255) NOT NULL COMMENT '区县名称',
    `address` VARCHAR(255) NOT NULL COMMENT '详细收货地址',
    `postal_code` VARCHAR(255) NULL DEFAULT NULL COMMENT '邮政编码',
    `is_default` TINYINT NOT NULL DEFAULT 0 COMMENT '是否默认地址：0否，1是',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    `deleted_at` DATETIME(6) NULL DEFAULT NULL COMMENT '软删除时间，NULL表示未删除',
    PRIMARY KEY (`id`),
    KEY `idx_member_address_member_id` (`member_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='会员收货地址表';


-- =============================================================================
-- 6. 购物车模块 (Cart)
-- =============================================================================

-- 6.1 购物车明细表
CREATE TABLE `cart_item` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '购物车明细ID',
    `member_id` BIGINT NOT NULL COMMENT '会员ID',
    `product_id` BIGINT NOT NULL COMMENT '商品ID',
    `sku_id` BIGINT NOT NULL COMMENT 'SKU ID',
    `quantity` INT NOT NULL DEFAULT 1 COMMENT '加购商品数量',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_cart_item_member_sku` (`member_id`, `sku_id`),
    KEY `idx_cart_item_member_id` (`member_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='购物车明细表';


-- =============================================================================
-- 7. 订单与支付模块 (Order & Payment)
-- =============================================================================

-- 7.1 订单主表
CREATE TABLE `order_main` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '订单ID',
    `order_no` VARCHAR(64) NOT NULL COMMENT '订单业务编号',
    `member_id` BIGINT NOT NULL COMMENT '会员ID',
    `request_id` VARCHAR(64) NOT NULL COMMENT '客户端幂等请求ID',
    `status` TINYINT NOT NULL DEFAULT 20 COMMENT '订单状态：10待付款，20待发货，30待收货，40已完成，50已取消，60已退款',
    `total_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '订单商品总金额，单位：元',
    `pay_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '实付金额，单位：元',
    `freight_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '运费金额，单位：元',
    `discount_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '优惠金额，单位：元',
    `express_company` VARCHAR(64) NULL DEFAULT NULL COMMENT '发货快递物流公司名称',
    `express_no` VARCHAR(64) NULL DEFAULT NULL COMMENT '发货快递物流单号',
    `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '订单买家留言备注',
    `paid_at` DATETIME(6) NULL DEFAULT NULL COMMENT '支付成功时间',
    `shipped_at` DATETIME(6) NULL DEFAULT NULL COMMENT '商家发货时间',
    `completed_at` DATETIME(6) NULL DEFAULT NULL COMMENT '订单完成时间',
    `cancelled_at` DATETIME(6) NULL DEFAULT NULL COMMENT '订单取消时间',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    `deleted_at` DATETIME(6) NULL DEFAULT NULL COMMENT '软删除时间，NULL表示未删除',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_order_no` (`order_no`),
    UNIQUE KEY `uk_order_member_request` (`member_id`, `request_id`),
    KEY `idx_order_main_member_id` (`member_id`),
    KEY `idx_order_main_status` (`status`),
    KEY `idx_order_main_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='订单主表';

-- 7.2 订单商品快照表
CREATE TABLE `order_item` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '订单明细ID',
    `order_id` BIGINT NOT NULL COMMENT '订单ID',
    `product_id` BIGINT NOT NULL COMMENT '商品ID',
    `sku_id` BIGINT NOT NULL COMMENT 'SKU ID',
    `product_name` VARCHAR(255) NOT NULL COMMENT '下单时商品名称快照',
    `sku_code` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '下单时SKU编码快照',
    `sku_spec_values` JSON NULL DEFAULT NULL COMMENT '下单时SKU规格快照JSON',
    `sku_image_id` BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '下单时SKU图片文件ID快照',
    `sku_image_url` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '下单时SKU图片地址快照',
    `unit_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '下单时商品单价快照，单位：元',
    `quantity` INT NOT NULL DEFAULT 1 COMMENT '购买商品数量',
    `total_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '下单时商品行总金额快照，单位：元',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_order_item_order_id` (`order_id`),
    KEY `idx_order_item_sku_id` (`sku_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='订单商品快照表';

-- 7.3 订单收货地址快照表
CREATE TABLE `order_address` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '订单收货地址ID',
    `order_id` BIGINT NOT NULL COMMENT '订单ID',
    `name` VARCHAR(255) NOT NULL COMMENT '下单时收货人姓名快照',
    `mobile` VARCHAR(255) NOT NULL COMMENT '下单时收货人手机号快照',
    `province_code` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '下单时省份行政编码快照',
    `province` VARCHAR(255) NOT NULL COMMENT '下单时省份名称快照',
    `city_code` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '下单时城市行政编码快照',
    `city` VARCHAR(255) NOT NULL COMMENT '下单时城市名称快照',
    `district_code` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '下单时区县行政编码快照',
    `district` VARCHAR(255) NOT NULL COMMENT '下单时区县名称快照',
    `address` VARCHAR(255) NOT NULL COMMENT '下单时详细收货地址快照',
    `postal_code` VARCHAR(255) NULL DEFAULT NULL COMMENT '下单时邮政编码快照',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_order_address_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='订单收货地址快照表';

-- 7.4 订单操作日志表
CREATE TABLE `order_log` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '订单操作日志ID',
    `order_id` BIGINT NOT NULL COMMENT '订单ID',
    `from_status` TINYINT NOT NULL DEFAULT 0 COMMENT '变更前订单状态：0初始/无，10待付款，20待发货，30待收货，40已完成，50已取消，60已退款',
    `to_status` TINYINT NOT NULL DEFAULT 0 COMMENT '变更后订单状态：10待付款，20待发货，30待收货，40已完成，50已取消，60已退款',
    `action` VARCHAR(64) NOT NULL COMMENT '操作动作名称',
    `operator_type` VARCHAR(32) NOT NULL COMMENT '操作人类型：user用户，admin管理员，system系统',
    `operator_id` BIGINT NOT NULL DEFAULT 0 COMMENT '操作人ID：0表示系统操作，>0表示实际操作用户',
    `operator_name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '操作人姓名快照',
    `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '操作备注说明',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '操作时间',
    PRIMARY KEY (`id`),
    KEY `idx_order_log_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='订单操作日志表';

-- 7.5 订单支付记录表
CREATE TABLE `order_payment` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '支付记录ID',
    `payment_no` VARCHAR(64) NOT NULL COMMENT '支付单业务编号',
    `order_id` BIGINT NOT NULL COMMENT '订单ID',
    `channel` VARCHAR(32) NOT NULL DEFAULT 'mock' COMMENT '支付渠道：mock模拟支付、wechat微信支付',
    `amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '支付金额，单位：元',
    `status` TINYINT NOT NULL DEFAULT 10 COMMENT '支付状态：10待支付，20支付成功，30支付失败，40已关闭',
    `transaction_no` VARCHAR(128) NULL DEFAULT NULL COMMENT '第三方支付交易流水号',
    `paid_at` DATETIME(6) NULL DEFAULT NULL COMMENT '支付成功时间',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_order_payment_payment_no` (`payment_no`),
    KEY `idx_order_payment_order_id` (`order_id`),
    KEY `idx_order_payment_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='订单支付记录表';


-- =============================================================================
-- 8. 售后与退款模块 (After Sale & Refund)
-- =============================================================================

-- 8.1 售后申请主表
CREATE TABLE `after_sale` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '售后单ID',
    `after_sale_no` VARCHAR(64) NOT NULL COMMENT '售后业务编号',
    `request_id` VARCHAR(64) NOT NULL COMMENT '客户端幂等请求ID',
    `order_id` BIGINT NOT NULL COMMENT '订单ID',
    `member_id` BIGINT NOT NULL COMMENT '会员ID',
    `type` TINYINT NOT NULL COMMENT '售后类型：1仅退款，2退货退款',
    `status` TINYINT NOT NULL DEFAULT 10 COMMENT '售后状态：10待审核，20待买家退货，30待商家收货，40退款处理中，50已完成，60已拒绝，70已取消，80退款失败',
    `order_status_at_apply` TINYINT NOT NULL COMMENT '申请售后时的订单状态快照：10待付款，20待发货，30待收货，40已完成，50已取消，60已退款',
    `reason` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '申请售后原因',
    `description` VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '售后问题详细描述',
    `proof_file_ids` JSON NULL DEFAULT NULL COMMENT '售后凭证图片文件ID列表JSON数组',
    `refund_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '申请退款总金额，单位：元',
    `return_receiver` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '商家退货收件人姓名',
    `return_phone` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '商家退货联系电话',
    `return_address` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '商家退货收货详细地址',
    `buyer_return_company` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '买家退货物流快递公司',
    `buyer_return_no` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '买家退货快递物流单号',
    `reject_reason` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '商家拒绝售后原因',
    `applied_at` DATETIME(6) NOT NULL COMMENT '售后申请提交时间',
    `audited_at` DATETIME(6) NULL DEFAULT NULL COMMENT '售后审核完成时间',
    `buyer_returned_at` DATETIME(6) NULL DEFAULT NULL COMMENT '买家退货发货时间',
    `merchant_received_at` DATETIME(6) NULL DEFAULT NULL COMMENT '商家确认收到退货时间',
    `refund_started_at` DATETIME(6) NULL DEFAULT NULL COMMENT '退款流程开始时间',
    `completed_at` DATETIME(6) NULL DEFAULT NULL COMMENT '售后单完成时间',
    `cancelled_at` DATETIME(6) NULL DEFAULT NULL COMMENT '售后单取消时间',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_after_sale_no` (`after_sale_no`),
    UNIQUE KEY `uk_after_sale_member_request` (`member_id`, `request_id`),
    KEY `idx_after_sale_order_id` (`order_id`),
    KEY `idx_after_sale_member_status` (`member_id`, `status`),
    KEY `idx_after_sale_status_created` (`status`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='售后申请主表';

-- 8.2 售后商品快照表
CREATE TABLE `after_sale_item` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '售后明细ID',
    `after_sale_id` BIGINT NOT NULL COMMENT '售后单ID',
    `order_item_id` BIGINT NOT NULL COMMENT '订单商品明细ID',
    `product_id` BIGINT NOT NULL COMMENT '商品ID快照',
    `sku_id` BIGINT NOT NULL COMMENT 'SKU ID快照',
    `quantity` INT NOT NULL DEFAULT 1 COMMENT '申请售后商品数量',
    `refund_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '售后商品行退款金额快照，单位：元',
    `product_name` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '售后时商品名称快照',
    `sku_text` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '售后商品规格文本快照',
    `sku_code` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '售后SKU编码快照',
    `sku_image_url` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '售后SKU图片地址快照',
    `restock_quantity` INT NOT NULL DEFAULT 0 COMMENT '售后退货实际重新入库数量',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    PRIMARY KEY (`id`),
    KEY `idx_after_sale_item_after_sale_id` (`after_sale_id`),
    KEY `idx_after_sale_item_order_item_id` (`order_item_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='售后商品快照表';

-- 8.3 售后操作日志表
CREATE TABLE `after_sale_log` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '售后操作日志ID',
    `after_sale_id` BIGINT NOT NULL COMMENT '售后单ID',
    `operator_type` TINYINT NOT NULL COMMENT '操作人类型：1用户，2管理员，3系统',
    `operator_id` BIGINT NOT NULL DEFAULT 0 COMMENT '操作人ID：0表示系统操作，>0表示实际操作用户',
    `operator_name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '操作人姓名快照',
    `from_status` TINYINT NOT NULL DEFAULT 0 COMMENT '变更前售后状态：0初始/无，10待审核，20待买家退货，30待商家收货，40退款处理中，50已完成，60已拒绝，70已取消，80退款失败',
    `to_status` TINYINT NOT NULL DEFAULT 0 COMMENT '变更后售后状态：10待审核，20待买家退货，30待商家收货，40退款处理中，50已完成，60已拒绝，70已取消，80退款失败',
    `action` VARCHAR(64) NOT NULL COMMENT '操作动作名称',
    `remark` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '操作备注说明',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '操作时间',
    PRIMARY KEY (`id`),
    KEY `idx_after_sale_log_after_sale_id_id` (`after_sale_id`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='售后操作日志表';

-- 8.4 退款记录表
CREATE TABLE `order_refund` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '退款记录ID',
    `refund_no` VARCHAR(64) NOT NULL COMMENT '退款业务编号',
    `after_sale_id` BIGINT NOT NULL COMMENT '售后单ID',
    `order_id` BIGINT NOT NULL COMMENT '订单ID',
    `payment_id` BIGINT NOT NULL DEFAULT 0 COMMENT '支付记录ID',
    `member_id` BIGINT NOT NULL COMMENT '会员ID',
    `refund_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '退款金额，单位：元',
    `status` TINYINT NOT NULL DEFAULT 10 COMMENT '退款状态：10待退款，20退款处理中，30退款成功，40退款失败',
    `provider` VARCHAR(32) NOT NULL DEFAULT 'mock' COMMENT '退款服务商：mock模拟退款、wechat微信支付',
    `provider_refund_no` VARCHAR(128) NULL DEFAULT NULL COMMENT '第三方退款交易流水号',
    `retry_count` INT NOT NULL DEFAULT 0 COMMENT '重试退款调用次数',
    `last_error` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '最后一次退款错误信息',
    `requested_at` DATETIME(6) NULL DEFAULT NULL COMMENT '最后一次发起退款调用时间',
    `refunded_at` DATETIME(6) NULL DEFAULT NULL COMMENT '第三方退款到账成功时间',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_order_refund_refund_no` (`refund_no`),
    UNIQUE KEY `uk_order_refund_after_sale_id` (`after_sale_id`),
    KEY `idx_order_refund_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='退款记录表';
