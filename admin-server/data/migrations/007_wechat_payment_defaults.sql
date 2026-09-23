-- 新支付单和退款单只能由正式微信支付流程创建；历史记录保持原渠道。
ALTER TABLE `order_payment`
  MODIFY COLUMN `channel` VARCHAR(32) NOT NULL DEFAULT 'wechat'
  COMMENT '支付渠道：wechat微信支付；mock为历史测试数据';
ALTER TABLE `order_refund`
  MODIFY COLUMN `provider` VARCHAR(32) NOT NULL DEFAULT 'wechat'
  COMMENT '退款渠道：wechat微信支付；mock为历史测试数据';
