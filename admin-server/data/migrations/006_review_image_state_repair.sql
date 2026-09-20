-- 004 可能已应用，使用补偿迁移重建剩余图片绑定的派生状态。
-- 已被 004 删除的绑定不猜测恢复；仅以现存绑定作为归属依据。
UPDATE `product_review` r
SET r.`has_image` = IF(EXISTS (
    SELECT 1 FROM `product_review_image` i WHERE i.`review_id` = r.`id`
), 1, 0);

-- 没有公开且未删除的评价引用时，评价文件必须为私有。
UPDATE `system_file` f
SET f.`visibility` = IF(EXISTS (
    SELECT 1 FROM `product_review_image` i
    INNER JOIN `product_review` r ON r.`id` = i.`review_id`
    WHERE i.`file_id` = f.`id` AND r.`status` = 20 AND r.`deleted_at` IS NULL
), 1, 2)
WHERE f.`scene` = 'review';
