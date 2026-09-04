CREATE TABLE IF NOT EXISTS casbin_rule (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    ptype VARCHAR(100) NOT NULL DEFAULT '',
    v0 VARCHAR(100) NOT NULL DEFAULT '',
    v1 VARCHAR(100) NOT NULL DEFAULT '',
    v2 VARCHAR(100) NOT NULL DEFAULT '',
    v3 VARCHAR(100) NOT NULL DEFAULT '',
    v4 VARCHAR(100) NOT NULL DEFAULT '',
    v5 VARCHAR(100) NOT NULL DEFAULT '',
    UNIQUE KEY idx_casbin_rule (ptype, v0, v1, v2, v3, v4, v5)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO system_role (name, code, status, sort, remark)
SELECT '超级管理员', 'super_admin', 1, 0, '系统内置角色'
WHERE NOT EXISTS (
    SELECT 1 FROM system_role WHERE code = 'super_admin'
);
