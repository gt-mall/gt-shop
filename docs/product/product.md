


product.specs

```json
[
  {
    "id": 1,
    "name": "颜色",
    "values": [
      {
        "id": 1,
        "name": "黑色",
      },
      {
        "id": 2,
        "name": "白色",
      }
    ]
  },
  {
    "id": 2,
    "name": "容量",
    "values": [
      {
        "id": 3,
        "name": "128GB"
      },
      {
        "id": 4,
        "name": "256GB"
      }
    ]
  }
]
```

sku
```json
[
  { "spec_id": 1, "value_id": 1 },
  { "spec_id": 2, "value_id": 4 }
]
```


```sql
-- =========================================================
-- 1. 商品分类
-- =========================================================
CREATE TABLE `product_category` (
    `id` INT NOT NULL AUTO_INCREMENT COMMENT '分类ID',
    `parent_id` INT NOT NULL DEFAULT 0 COMMENT '父分类ID，0表示顶级分类',

    `name` VARCHAR(255) NOT NULL COMMENT '分类名称',
    `image_id` BIGINT UNSIGNED NULL COMMENT '分类图片ID，关联system_file.id',

    `sort` INT NOT NULL DEFAULT 0 COMMENT '排序值',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0禁用，1启用',

    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME NULL COMMENT '软删除时间',

    PRIMARY KEY (`id`),
    KEY `idx_parent_id` (`parent_id`)
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_0900_ai_ci
COMMENT='商品分类';


-- =========================================================
-- 2. 商品主表
-- =========================================================
CREATE TABLE `product` (
    `id` INT NOT NULL AUTO_INCREMENT COMMENT '商品ID',
    `category_id` INT NOT NULL DEFAULT 0 COMMENT '商品分类ID',

    `name` VARCHAR(255) NOT NULL COMMENT '商品名称',
    `subtitle` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '商品副标题/卖点',

    `main_image_id` BIGINT UNSIGNED NULL COMMENT '商品主图ID，关联system_file.id',

    `gallery_file_ids` JSON NULL COMMENT '商品轮播图片ID数组',
    `specs` JSON NULL COMMENT '商品规格定义',

    `detail` LONGTEXT NULL COMMENT '商品富文本详情HTML',

    `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态：0下架，1上架',
    `sort` INT NOT NULL DEFAULT 0 COMMENT '排序值',

    `sales_count` BIGINT NOT NULL DEFAULT 0 COMMENT '实际销量',
    `virtual_sales` INT NOT NULL DEFAULT 0 COMMENT '虚拟销量',

    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME NULL COMMENT '软删除时间',

    PRIMARY KEY (`id`),
    KEY `idx_category_id` (`category_id`)
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_0900_ai_ci
COMMENT='商品';


-- =========================================================
-- 3. 商品 SKU
-- =========================================================
CREATE TABLE `product_sku` (
    `id` INT NOT NULL AUTO_INCREMENT COMMENT 'SKU ID',
    `product_id` INT NOT NULL COMMENT '商品ID',

    `spec_values` JSON NOT NULL COMMENT 'SKU规格值组合',
    `spec_hash` CHAR(64) NOT NULL COMMENT '规格组合SHA256',

    `sku_code` VARCHAR(255) NULL COMMENT '商家SKU编码',
    `barcode` VARCHAR(255) NULL COMMENT '商品条形码',

    `image_id` BIGINT UNSIGNED NULL COMMENT 'SKU图片ID，关联system_file.id',

    `price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '销售价格，单位元',
    `market_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '划线价格，单位元',
    `cost_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '成本价格，单位元',

    `unit` VARCHAR(255) NULL COMMENT '销售单位，如件/个/瓶/盒/箱/套',
    `weight_gram` INT NULL COMMENT '物流计费重量，单位克，可选',

    `sort` INT NOT NULL DEFAULT 0 COMMENT '排序值',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0禁用，1启用',

    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME NULL COMMENT '软删除时间',

    PRIMARY KEY (`id`),

    UNIQUE KEY `uk_product_spec` (`product_id`, `spec_hash`),
    UNIQUE KEY `uk_sku_code` (`sku_code`),

    KEY `idx_product_id` (`product_id`)
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_0900_ai_ci
COMMENT='商品SKU';


```