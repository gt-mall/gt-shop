CREATE TABLE IF NOT EXISTS order_payment (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    payment_no VARCHAR(64) NOT NULL COMMENT 'P-prefixed payment business number',
    order_id BIGINT NOT NULL COMMENT 'associated order_id',
    channel VARCHAR(32) NOT NULL DEFAULT 'mock' COMMENT 'payment channel: mock, wechat, etc.',
    amount DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT 'payment amount in yuan',
    status TINYINT NOT NULL DEFAULT 10 COMMENT '10:pending, 20:success, 30:failed, 40:closed',
    transaction_no VARCHAR(128) NULL DEFAULT NULL COMMENT 'external transaction number',
    paid_at DATETIME(6) NULL DEFAULT NULL COMMENT 'payment success time',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    UNIQUE KEY uk_order_payment_payment_no (payment_no),
    KEY idx_order_payment_order_id (order_id),
    KEY idx_order_payment_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
