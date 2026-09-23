# 小程序微信支付

小程序订单仅通过微信支付。会员在微信小程序中登录后，订单详情页调用 POST /api/miniapp/payments/prepay 获取微信预支付参数，再调用 uni.requestPayment。客户端的支付成功提示不改变订单状态；后端以微信支付通知或主动查询到的成功交易为准，核对商户号、AppID、付款人 OpenID、金额和交易号后，才扣减锁定库存并将订单转为待发货。GET /api/miniapp/payments/{payment_no} 可查询并同步支付状态。

微信支付和退款通知分别是 /api/miniapp/payments/wechat/notify 与 /api/miniapp/payments/wechat/refund-notify。通知必须经过微信支付签名与报文校验。旧模拟支付接口不再注册，后端也不再提供模拟支付成功方法。历史模拟支付记录仅用于兼容查询和关闭遗留待支付单。

正式使用前，在后台配置小程序 AppID/AppSecret、微信支付商户号、证书序列号、商户私钥、微信支付公钥及 ID、APIv3 密钥和公网 HTTPS 回调域名；服务端还必须设置 PAYMENT_CONFIG_KEY 以加密数据库中的商户凭据。不要把这些密钥下发给小程序。H5 订单详情页不调起微信小程序支付。

## 超时与库存

下单时锁定库存。支付成功后，后端确认交易并扣减库存。未支付订单超时由 worker 扫描关闭；如已有微信待支付单，先向微信查询并关单，再释放锁定库存，避免成功付款与取消同时发生。订单和支付单的状态更新在事务中完成。

## 验证

在 admin-server 执行 go test ./internal/test/miniapp ./internal/test/order。MySQL 并发测试需单独设置指向可丢弃 _test 库的 ORDER_TEST_MYSQL_DSN，执行 go test -v ./internal/test/miniapp -run '^TestPaymentMySQLConcurrency$' -count=1。微信商户配置、通知回调、退款和真机扣款仍需在真实商户环境验证。
