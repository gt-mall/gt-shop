-- =============================================================================
-- 004_review_image_unique.sql
-- 评价图片唯一归属约束与历史评价内容转义修复
-- =============================================================================

-- 1. 清理已有重复引用的评价图片（若有），仅保留每张图片最早的绑定记录
DELETE pri FROM `product_review_image` pri
INNER JOIN (
    SELECT `file_id`, MIN(`id`) AS keep_id
    FROM `product_review_image`
    GROUP BY `file_id`
    HAVING COUNT(*) > 1
) dup ON pri.`file_id` = dup.`file_id` AND pri.`id` != dup.keep_id;

-- 2. 移除旧的 (review_id, file_id) 联合唯一键，添加 file_id 全局唯一键，确保一个文件仅能归属于一条评价
ALTER TABLE `product_review_image` DROP INDEX `uk_review_file`;
ALTER TABLE `product_review_image` ADD UNIQUE KEY `uk_review_image_file` (`file_id`);

-- 3. 修复历史评价中可能被 HTML 实体化的字符
UPDATE `product_review` SET `content` = REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(`content`, '&quot;', '"'), '&#34;', '"'), '&#39;', '\''), '&lt;', '<'), '&gt;', '>') WHERE `content` LIKE '%&%';
UPDATE `product_review` SET `content` = REPLACE(`content`, '&amp;', '&') WHERE `content` LIKE '%&amp;%';
