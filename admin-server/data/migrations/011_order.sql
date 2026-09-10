CREATE TABLE IF NOT EXISTS order_main (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_no VARCHAR(64) NOT NULL,
    member_id BIGINT NOT NULL,
    request_id VARCHAR(64) NOT NULL,
    status TINYINT NOT NULL DEFAULT 20 COMMENT '10:pending_payment, 20:pending_shipment, 30:pending_receipt, 40:completed, 50:cancelled',
    total_amount DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    pay_amount DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    freight_amount DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    discount_amount DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    express_company VARCHAR(64) NULL DEFAULT NULL,
    express_no VARCHAR(64) NULL DEFAULT NULL,
    remark VARCHAR(255) NOT NULL DEFAULT '',
    paid_at DATETIME(6) NULL DEFAULT NULL,
    shipped_at DATETIME(6) NULL DEFAULT NULL,
    completed_at DATETIME(6) NULL DEFAULT NULL,
    cancelled_at DATETIME(6) NULL DEFAULT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL DEFAULT NULL,
    UNIQUE KEY uk_order_no (order_no),
    UNIQUE KEY uk_order_member_request (member_id, request_id),
    KEY idx_order_main_member_id (member_id),
    KEY idx_order_main_status (status),
    KEY idx_order_main_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS order_item (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    sku_id BIGINT NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    sku_code VARCHAR(64) NOT NULL DEFAULT '',
    sku_spec_values JSON NULL,
    sku_image_id BIGINT UNSIGNED NULL,
    sku_image_url VARCHAR(1024) NOT NULL DEFAULT '',
    unit_price DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    quantity INT NOT NULL DEFAULT 1,
    total_amount DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    KEY idx_order_item_order_id (order_id),
    KEY idx_order_item_sku_id (sku_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS order_address (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    mobile VARCHAR(255) NOT NULL,
    province_code VARCHAR(255) NOT NULL DEFAULT '',
    province VARCHAR(255) NOT NULL,
    city_code VARCHAR(255) NOT NULL DEFAULT '',
    city VARCHAR(255) NOT NULL,
    district_code VARCHAR(255) NOT NULL DEFAULT '',
    district VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,
    postal_code VARCHAR(255) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    UNIQUE KEY uk_order_address_order_id (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS order_log (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT NOT NULL,
    action VARCHAR(64) NOT NULL,
    operator_type VARCHAR(32) NOT NULL,
    operator_id BIGINT NOT NULL DEFAULT 0,
    operator_name VARCHAR(100) NOT NULL DEFAULT '',
    remark VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    KEY idx_order_log_order_id (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
