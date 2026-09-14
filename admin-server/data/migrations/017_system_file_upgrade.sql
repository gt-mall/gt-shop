-- 017_system_file_upgrade.sql
-- Upgrade system_file from image material to unified site-wide file resource table

ALTER TABLE system_file
    ADD COLUMN original_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'original filename' AFTER name,
    ADD COLUMN mime_type VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'mime type' AFTER ext,
    ADD COLUMN file_type VARCHAR(32) NOT NULL DEFAULT 'image' COMMENT 'image, video, audio, document, archive, other' AFTER mime_type,
    ADD COLUMN scene VARCHAR(64) NOT NULL DEFAULT 'admin_material' COMMENT 'business scene: admin_material, after_sale_proof, etc.' AFTER file_type,
    ADD COLUMN owner_type VARCHAR(32) NOT NULL DEFAULT 'admin' COMMENT 'admin, member, system' AFTER scene,
    ADD COLUMN owner_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'owner id' AFTER owner_type,
    ADD COLUMN visibility TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '1=public, 2=private' AFTER owner_id,
    ADD COLUMN duration_ms BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'duration in milliseconds' AFTER height,
    ADD COLUMN cover_file_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'cover image system_file.id' AFTER duration_ms,
    ADD KEY idx_file_scene_type (scene, file_type),
    ADD KEY idx_file_owner (owner_type, owner_id);

-- Backfill legacy records
UPDATE system_file SET original_name = name WHERE original_name = '' OR original_name IS NULL;
UPDATE system_file SET scene = 'admin_material' WHERE scene = '' OR scene IS NULL;
UPDATE system_file SET owner_type = 'admin' WHERE owner_type = '' OR owner_type IS NULL;
UPDATE system_file SET visibility = 1 WHERE visibility = 0 OR visibility IS NULL;

-- Infer file_type and mime_type for existing records based on extension
UPDATE system_file SET
    file_type = CASE
        WHEN LOWER(ext) IN ('jpg', 'jpeg', 'png', 'webp', 'gif', 'bmp', 'svg') THEN 'image'
        WHEN LOWER(ext) IN ('mp4', 'mov', 'webm', 'avi', 'mkv') THEN 'video'
        WHEN LOWER(ext) IN ('mp3', 'wav', 'aac', 'm4a', 'flac', 'ogg') THEN 'audio'
        WHEN LOWER(ext) IN ('pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'csv') THEN 'document'
        WHEN LOWER(ext) IN ('zip', 'rar', '7z', 'tar', 'gz') THEN 'archive'
        ELSE 'other'
    END,
    mime_type = CASE
        WHEN LOWER(ext) IN ('jpg', 'jpeg') THEN 'image/jpeg'
        WHEN LOWER(ext) = 'png' THEN 'image/png'
        WHEN LOWER(ext) = 'webp' THEN 'image/webp'
        WHEN LOWER(ext) = 'gif' THEN 'image/gif'
        WHEN LOWER(ext) = 'mp4' THEN 'video/mp4'
        WHEN LOWER(ext) = 'mov' THEN 'video/quicktime'
        WHEN LOWER(ext) = 'pdf' THEN 'application/pdf'
        ELSE mime_type
    END
WHERE file_type = '' OR file_type IS NULL OR file_type = 'image';
