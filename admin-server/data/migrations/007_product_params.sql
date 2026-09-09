-- 增加商品参数字段
ALTER TABLE `product` ADD COLUMN `params` JSON NULL COMMENT '商品参数列表[{name, value}]' AFTER `specs`;
