# 微信小程序正式支付接入

当前仅支持普通商户的小程序支付。后端默认监听 `8100`，小程序生产 API 域名为 `https://gt-shop.xi-soft.cn`。

## 上线前配置

1. 在后端运行环境设置 `PAYMENT_CONFIG_KEY`：用 `openssl rand -base64 32` 生成一次，妥善备份，并让所有后端实例使用同一个值。此值不能放进 Git；更换它前必须先迁移数据库中的支付配置密文。
2. 在后台“小程序设置”填写已绑定商户号的小程序 AppID、AppSecret。AppSecret 留空会保留原值，页面不回显密钥。
3. 在后台“小程序设置 → 微信小程序支付”填写普通商户号、商户 API 证书序列号、商户私钥 `apiclient_key.pem`、微信支付公钥 ID 和公钥 PEM、32 字节 APIv3 密钥。密钥留空会保留原值；后端对整份支付配置加密保存，后台只返回已配置状态。
4. 确认公网 HTTPS 反向代理将 `/api/miniapp/*` 转发到后端 `8100`，并允许微信支付回调到达：
   - 支付：`https://gt-shop.xi-soft.cn/api/miniapp/payments/wechat/notify`
   - 退款：`https://gt-shop.xi-soft.cn/api/miniapp/payments/wechat/refund-notify`
5. 在微信公众平台把 `https://gt-shop.xi-soft.cn` 配为小程序 `request` 合法域名，发布新小程序包。会员须通过微信小程序登录，取得该 AppID 对应的 OpenID。
6. 在独立低金额订单上用真机做付款、取消、超时关单和退款验收。微信支付凭据未填写之前，预支付接口会拒绝请求，不会回退到模拟付款。

支付成功以签名回调或服务端查单为准。退款申请被微信受理后保持处理中，直到回调或定时查询确认最终结果。支付与退款均每分钟最多扫描 100 笔待确认记录；大量积压时需要监控与分批处理。
