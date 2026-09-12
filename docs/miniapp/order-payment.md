# 订单与模拟付款

当前使用模拟付款。小程序付款接口为 `POST /api/miniapp/payments/create` 和 `POST /api/miniapp/payments/mock-pay`，均要求会员登录。

- 下单锁定库存；模拟付款成功后扣减库存，订单转为待发货。
- 同一订单并发创建支付单返回同一张待支付单；重复模拟付款返回原成功支付记录。
- 成功、失败和重复请求都先校验订单归属。请求同时提供支付单号和订单 ID 时，两者必须匹配。
- 支付成功时关闭该订单遗留的其他待支付单，避免产生重复成功流水。
- 取消待付款订单释放锁定库存；取消待发货订单保留现有模拟业务行为并恢复库存。

## 自动关闭未付款订单

后端启动后立即扫描一次，此后每分钟扫描一次，将创建时间超过 30 分钟的待付款订单取消，并在同一事务中释放库存、关闭待支付单、写入订单日志。已付款订单不受影响。

每批查询 100 条。单笔失败会回滚，记录错误后继续处理其他订单，下一轮重试失败订单。多个服务实例同时扫描时，通过订单行锁避免重复处理。默认时间位于 `admin-server/internal/miniapp/payment/worker.go`。

重新启动更新后的后端时，历史上超过 30 分钟的待付款订单也会被扫描关闭。

## 验证

在 `admin-server` 目录运行：

```bash
go test ./internal/test/miniapp ./internal/test/order
```

MySQL 并发回归测试需单独配置 `ORDER_TEST_MYSQL_DSN`，指向名称以 `_test` 结尾的可丢弃测试库。测试会建表并写入测试数据，不读取应用的 `DATABASE_DSN`。

```bash
go test -v ./internal/test/miniapp -run '^TestPaymentMySQLConcurrency$' -count=1
```

测试覆盖同订单重复创建/付款、不同订单同时创建支付单，以及付款与取消/超时关闭的竞争。SQLite 单测另覆盖会员隔离、遗留重复支付单、回滚、分批扫描和失败重试。
