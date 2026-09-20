-- =============================================================================
-- 003_product_review.sql
-- 商品评价与晒单数据表：商品评价表、评价图片关联表、评价审核与操作日志表
-- =============================================================================

-- 1. 商品评价主表
CREATE TABLE IF NOT EXISTS `product_review` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '评价ID',
    `order_id` BIGINT NOT NULL COMMENT '订单ID',
    `order_item_id` BIGINT NOT NULL COMMENT '订单明细ID',
    `member_id` BIGINT NOT NULL COMMENT '会员ID',
    `request_id` VARCHAR(64) NOT NULL COMMENT '客户端幂等请求ID',
    `product_id` BIGINT NOT NULL COMMENT '商品ID',
    `sku_id` BIGINT NOT NULL COMMENT 'SKUID',
    `product_name` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '商品名称快照',
    `sku_spec_values` JSON DEFAULT NULL COMMENT 'SKU规格值快照',
    `rating` TINYINT NOT NULL DEFAULT 5 COMMENT '评分星级：1-5星',
    `content` VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '评价内容：1-500个Unicode字符',
    `is_anonymous` TINYINT NOT NULL DEFAULT 0 COMMENT '是否匿名：0实名，1匿名',
    `status` TINYINT NOT NULL DEFAULT 10 COMMENT '审核状态：10待审核，20公开，30驳回，40隐藏',
    `has_image` TINYINT NOT NULL DEFAULT 0 COMMENT '是否有图：0无图，1有图',
    `audit_reason` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '审核驳回或操作原因',
    `audit_user_id` BIGINT DEFAULT NULL COMMENT '审核管理员ID',
    `audited_at` DATETIME(6) DEFAULT NULL COMMENT '审核时间',
    `reply_content` VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '商家回复内容（最多500字符）',
    `reply_user_id` BIGINT DEFAULT NULL COMMENT '回复管理员ID',
    `reply_at` DATETIME(6) DEFAULT NULL COMMENT '回复时间',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '最后更新时间',
    `deleted_at` DATETIME(6) DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_review_order_item_id` (`order_item_id`),
    UNIQUE KEY `uk_review_member_request` (`member_id`, `request_id`),
    KEY `idx_review_order_id` (`order_id`),
    KEY `idx_review_member_id` (`member_id`, `created_at`),
    KEY `idx_review_product_status` (`product_id`, `status`, `created_at`),
    KEY `idx_review_product_has_image` (`product_id`, `status`, `has_image`),
    KEY `idx_review_status_created` (`status`, `created_at`),
    KEY `idx_review_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品评价表';

-- 2. 商品评价图片关联表
CREATE TABLE IF NOT EXISTS `product_review_image` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `review_id` BIGINT NOT NULL COMMENT '评价ID',
    `file_id` BIGINT NOT NULL COMMENT '系统文件ID',
    `sort` INT NOT NULL DEFAULT 0 COMMENT '图片排序（0-5）',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_review_file` (`review_id`, `file_id`),
    KEY `idx_review_image_review_id` (`review_id`, `sort`),
    KEY `idx_review_image_file_id` (`file_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品评价图片关联表';

-- 3. 商品评价审核与操作日志表
CREATE TABLE IF NOT EXISTS `product_review_log` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '日志ID',
    `review_id` BIGINT NOT NULL COMMENT '评价ID',
    `operator_type` TINYINT NOT NULL COMMENT '操作人类型：1会员，2管理员，3系统',
    `operator_id` BIGINT NOT NULL DEFAULT 0 COMMENT '操作人ID',
    `operator_name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '操作人姓名/账号',
    `from_status` TINYINT NOT NULL DEFAULT 0 COMMENT '操作前状态',
    `to_status` TINYINT NOT NULL DEFAULT 0 COMMENT '操作后状态',
    `action` VARCHAR(64) NOT NULL COMMENT '操作动作：create, audit_approve, audit_reject, hide, restore, reply',
    `remark` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '操作备注或审核/驳回原因',
    `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '操作时间',
    PRIMARY KEY (`id`),
    KEY `idx_review_log_review_id` (`review_id`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='商品评价审核与操作日志表';
