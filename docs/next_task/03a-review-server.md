# 03A 商品评价与晒单：后端契约和审核

## 已有代码与前置条件

- 当前未找到商品评价表、业务路由和评价状态；`order_main` 状态 40 为已完成，`order_item` 保存商品/SKU 快照，可作为购买凭证。
- `internal/model/system_file.go` 有 `SceneReview`，但 `internal/system/file/service.go` 的 `isAllowedExtForScene` 没有 review 分支。仅传 scene=review 不能上传成功。
- 售后上传已有会员文件归属校验、文件类型检测、存储适配；复用其公共文件能力，不能冒用售后上传接口。
- 接下来 [03B](03b-review-clients.md) 依赖本任务的 Swagger。先读 README，补齐模型生成流程后再落表，不手工改生成代码。

## 执行提示词

### 第一版业务规则（本方案默认）

1. 已登录会员仅能评价自己已完成订单的某条 `order_item`，一条明细只能评价一次（数量大于 1 也只评价一次）。待付款/待发货/待收货/已取消/已退款订单不可首次评价；售后处理中不可首次评价。所有资格以服务端实时判断为准。
2. 星级 1–5 整数、正文 1–500 个 Unicode 字符、最多 6 张图片（单张不超过 5MB；jpg/jpeg/png/webp）；支持匿名展示。第一版不做视频、追评、点赞、自动好评、返现奖励。
3. 新评价为待审核，后台人工通过后才在商品端公开。待审核/驳回/隐藏时作者可在“我的评价”看状态，公众不可见；不按星级自动屏蔽差评。
4. 第一版提交后不可修改/重发；驳回展示原因，已审核评价可由管理员隐藏/恢复，审核结论和操作人时间留痕。退款后已有评价保留，不能自动改变订单或退款流程；新评价资格重新计算。

### 数据与模块

1. 核心代码建议归 `internal/product/review/{api,service,types}.go`，会员接口薄层归 `internal/miniapp/review/`，注册 `routers/product.go`（以实际文件为准）和 `routers/miniapp.go`。普通 GORM 事务，不引入 Repository/微服务。
2. 新增 `product_review`：bigint id、order_id、order_item_id、member_id、product_id、sku_id；rating、content、is_anonymous；status、审核原因/审核人/审核时间、created_at、updated_at；可保存商品名称/规格快照。订单/商品/会员关联字段必须由服务器从订单取得。
3. `order_item_id` 数据库唯一约束；同会员 `request_id` 唯一，用于弱网幂等。相同 request_id 相同正文/评分/图片返回同一结果，载荷不同明确冲突。禁止仅先查再插。状态建议 10 待审核、20 公开、30 驳回、40 隐藏，写进字段注释和 DTO。
4. 图片使用 `product_review_image(review_id,file_id,sort)`，唯一约束防同评价重复图片。图片只存 file_id，响应解析最新 URL。增加审核操作表或复用实际已有的审计能力，保留状态转换记录；不假定已有通用操作日志服务。
5. 按公开商品评价查询、会员查询、后台状态分页设计必要组合索引；平均分/有图数/总数仅计算公开评价。首版可直接聚合，不必建异步统计系统。评分计算避免不必要的浮点误差。

### 建议接口（需新实现并写入 Swagger）

| 端 | 接口 | 约束 |
| --- | --- | --- |
| 小程序 | `POST /api/miniapp/reviews/upload` | 会员上传，仅图片，绑定本人和 review 场景 |
| 小程序 | `POST /api/miniapp/reviews/create` | order_item_id、rating、content、image_ids、is_anonymous、request_id |
| 小程序 | `GET /api/miniapp/reviews/mine` | 只返回本人，分页 |
| 小程序 | `GET /api/miniapp/reviews/:id` | 未公开仅作者可查；管理端走自己的鉴权接口 |
| 公开 | `GET /api/miniapp/products/:id/reviews` | 公开评价列表、评分/有图过滤、分页；含统一 summary |
| 管理端 | `POST /api/product/reviews/list`、`detail`、`audit`、`reply` | 后端权限约束；audit 接收期望旧状态防并发覆盖 |

1. 订单明细响应扩展 `can_review`、`review_id`、`review_status`、`review_block_reason`；订单列表批量计算，禁止逐项 N+1。第一版不添加新的订单状态，待评价不等于 status=40。
2. 提交时在事务内验证订单归属和资格，复用订单/售后的锁顺序，处理确认收货、申请售后与提交评价竞态。不相信前端 `can_review`。
3. 所有 image_ids 必须存在、未删除、类型正确、scene=review、owner_type=member、owner_id=当前会员；拒绝其他会员、后台素材、售后凭证及重复 ID。复用文件大小、MIME 实测和存储校验。
4. 上传及未审核图片的访问也须考虑可见性：不能只过滤评价 JSON，而图片通过公开 URL 仍绕过审核。检查现有本地/对象存储实际访问控制；用受控读取或私有存储签名实现作者/审核者访问，审核通过再开放对应读取。不能仅把 visibility 字段改为 private 就声称完成。
5. 公开匿名评价不得返回 member_id、真实昵称、头像、手机号、order_id、order_no；非匿名也只暴露必要昵称/头像，不公开手机号/订单资料。正文按纯文本展示，禁止富文本注入。
6. 后台增加 list/detail/audit/reply 对应权限，复用 Casbin 和权限目录；商家回复最多 500 字，纯文本，记录操作人时间。评价和回复不允许伪造作者。
7. 优先独立拉取评价数据，避免商品详情缓存嵌入旧评价。若加入评价缓存或摘要，审核、隐藏、恢复、回复后失效相关缓存；Redis 不可用时降级读取。

## 验收与交付

- 真实购买资格、越权、一次评价、同 request_id 重试/冲突、并发提交、售后竞态；真实 MySQL 验证唯一约束和锁行为。
- 非法评分/空文本/超长 Unicode、图片数量/大小、伪造文件类型、文件归属和未审核图片访问。
- 审核只生效一次；隐藏后列表、评分和有图计数同步；匿名字段不泄漏；只有评价权限的管理员能正确访问。
- 增量迁移及隔离空库安装验证；扩展现有文件测试而非降低白名单。
- 运行 README 后端测试与 Swagger/Client 生成，把实际接口字段和状态语义交给 03B。
