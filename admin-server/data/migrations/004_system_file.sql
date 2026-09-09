CREATE TABLE IF NOT EXISTS system_file_group (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 name VARCHAR(64) NOT NULL,
 created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 UNIQUE KEY uk_file_group_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS system_file (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 group_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '0 means ungrouped',
 storage VARCHAR(16) NOT NULL COMMENT 'local / oss / cos / s3',
 name VARCHAR(255) NOT NULL COMMENT 'original filename or display name',
 path VARCHAR(255) NOT NULL COMMENT 'relative YYYY/MM/UUID.ext',
 ext VARCHAR(10) NOT NULL,
 size BIGINT UNSIGNED NOT NULL COMMENT 'bytes',
 width INT UNSIGNED NOT NULL,
 height INT UNSIGNED NOT NULL,
 created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 deleted_at DATETIME(6) NULL COMMENT 'hide from library, preserve business references',
 UNIQUE KEY uk_file_storage_path (storage, path),
 KEY idx_file_group (group_id, deleted_at, id),
 KEY idx_file_list (deleted_at, id),
 KEY idx_file_group_stat (deleted_at, group_id, size)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS system_config (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 config_key VARCHAR(64) NOT NULL,
 value JSON NOT NULL,
 created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
 UNIQUE KEY uk_system_config_key (config_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO system_file_group (name) VALUES ('商品图片'), ('分类图片'), ('广告图片'), ('装修图片');
INSERT IGNORE INTO system_config (config_key, value) VALUES
 ('file_storage', '{"default_storage":"local","local":{"path":"uploads","domain":""},"oss":{},"cos":{},"s3":{}}');
