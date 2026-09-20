# 实施复查与 AI 修复提示词

复查日期：2026-09-20；代码基线：`c975b26`。本次检查 `docs/next_task/01–06` 对应实现，没有修改业务代码。以下行号对应本次基线，后续执行须重新定位。

## 结论与证据

01、02、03A、03B、04、05A 已有实现，但仍存在以下问题；05B 和 06 尚未完成，不能把全部任务标记为已验收。本轮未确认 P0。

### 1. P1：公开评价详情泄露订单和内部审核字段（已复现）

- 位置：`admin-server/internal/miniapp/review/service.go:441`，尤其 `483–496`。
- 触发：会员 B 调用会员 A 已公开评价的 `GET /api/miniapp/reviews/:id`，包括匿名评价。
- 原因：只限制非公开评价的作者身份，公开评价仍复用 `MiniappMineReviewItemResponse`，返回 `order_id`、`order_item_id` 和 `audit_reason`。
- 临时测试实际读到：`order_id=1 order_item_id=1 audit_reason="internal moderation note"`。公开列表已脱敏，详情却绕过了字段边界。

### 2. P2：评价图片共用文件记录，审核可见性互相影响（已复现）

- 位置：`internal/miniapp/review/service.go:225` 附近图片校验；`internal/product/review/service.go:390` 附近可见性更新；`data/migrations/003_product_review.sql`。
- 同会员可将一个 `file_id` 绑定到不同评价，唯一键只约束 `(review_id, file_id)`。审核直接修改这份文件的全局 `visibility`。
- 已复现：两条待审核评价复用图片，第一条通过后，第二条关联图片也变为公开。反过来，隐藏其中一条会将另一条已公开评价的图片设为私有；重新审核其他引用又可改变它。
- 需要明确文件独占归属或按评价控制访问，不能只改评价 JSON。

### 3. P2：审核接口没有校验合法状态转换（已复现）

- 位置：`admin-server/internal/product/review/service.go:332`。
- `expected_status` 仅校验请求与数据库当前状态一致，没有约束 action 的来源状态。
- 已复现：待审核状态 10 携带 `action=restore, expected_status=10`，直接变为公开状态 20。公开评价也可携带新的期望状态反复 approve 并新增日志。
- 此问题发生在有审核权限的接口内部，并非证明普通会员能调用后台接口。前端按钮限制不能代替服务端状态机。

### 4. P2：评价事务缺少订单锁，幂等冲突分支漏比图片（源码确认，MySQL 并发未验证）

- 位置：`admin-server/internal/miniapp/review/service.go:162–175`、`321–334`。
- 注释写了 row locking，但读取订单实际是普通 `First`，没有 `clause.Locking`。售后申请在 `internal/order/aftersale/service.go:183` 使用订单行锁，两条流程没有形成互斥；评价读取资格后，售后可以先提交，评价仍按旧资格插入。
- 唯一键冲突后的重查只比较明细、评分、正文、匿名标记，没有比较图片 ID 和顺序；与事务前的幂等判断不一致。
- 同载荷并发请求还可能在进入事务后遇到“明细已评价”直接报错，没有收敛到同一评价结果。

### 5. P2：纯文本评价被 HTML 实体化（已复现）

- 位置：`admin-server/internal/miniapp/review/service.go:100`；`miniapp/src/components/ReviewCard.vue:42`。
- 提交 `A&B <3`，接口返回 `A&amp;B &lt;3`，Vue 文本插值会直接展示实体字符。
- 500 字校验发生在转义前，而 `content` 是 `VARCHAR(1000)`；例如 500 个 `&` 转义后变为 2500 字符，存在 MySQL 写入失败风险。本轮只复现文本往返错误，未运行真实 MySQL 长度校验。

### 6. P2：分页失败后跳页，首次失败被显示为空（页面源码复现）

- 位置：`miniapp/src/pages/review/product.vue:187`、`review/mine.vue:136`、`my/index.vue` 的 `onReachBottom`、`order/index.vue` 的分页。
- 这些路径在请求成功前递增页码，失败不回退。对真实 `review/product.vue` setup 注入第 2 页失败，实际请求序列是 `[1,2,3]`，正确重试应为 `[1,2,2]`。
- 商品评价页错误只 toast，首次请求失败后落入“暂无相关评价”；我的评价没有查询版本保护，刷新与分页也可能互相覆盖。

### 7. P2：私有页面及地址草稿缺少会话隔离（部分源码复现）

- 位置：`miniapp/src/pages/review/mine.vue:113`；`review/create.vue`、`review/detail.vue`；`pages/address/edit.vue:281`、`:530`。
- 我的评价既没有请求序号，也没有会话快照。真实 setup 测试中让新请求 B 先返回、旧请求 A 后返回，最终页面被 A 的内容覆盖。公共 request 层保护的是旧 401，不会自动拦截旧的成功响应。
- 微信导入只在获取/消费草稿时校验账号；填入编辑表单后没有保留草稿所属会话，保存只判断当前是否有 token。A 的草稿停留期间退出并登录 B，仍可能将 A 的地址保存到 B。
- 需要覆盖登录失效、返回保留页面、账号变化、卸载及原生选择器晚返回；本轮未做真机账号切换验收。

### 8. P2：订单倒计时和状态重查在页面隐藏后仍运行（源码确认）

- 位置：`miniapp/src/composables/useOrderCountdown.ts:120`；`pages/order/index.vue`、`pages/order/detail.vue`。
- 共享 ticker 只在组件卸载时释放，两页的重试任务也只在卸载时清理，没有 `onHide` 暂停。导航到其他页面时，保留在页面栈中的订单页仍可能触发倒计时和状态查询。
- 同时检查缺少或非法服务端时间的路径：当前 `OrderActionBar` / 页面存在按 `status === 10` 放行支付的兜底，与 01 要求的“先刷新确认”不一致；后端支付截止校验仍是最终保护。

### 9. 搜索清空没有使旧请求失效（页面状态已复现）

- 位置：`miniapp/src/pages/search/index.vue:338`、`:368`。
- 空词提交和清空输入只清数组，没有递增 `requestSeq`。搜索 A 请求尚未结束时清空，A 晚返回仍写入列表与总数。
- 真实 setup 复现结果：`keyword="", ids=[88]`。当前发现态模板会遮住列表，因此本轮证明的是清空后状态被旧请求重新写入，未把它夸大为已观察到的错误商品展示。
- 同次修复应检查加载 guard 之前递增序号、卸载晚返回、刷新失败时旧内容展示和当前生效查询标识。

### 10. 尚未完成：05B 商品海报、06 骨架与弱网

- `miniapp/src/pages/product/detail.vue:69` 只有原生分享入口；仓库没有 `ProductPoster` / 商品海报 Canvas，也没有商品 `share-code` 路由、微信 access_token 管理和商品码持久化服务。提交标题提及海报不能作为实现证据。
- `miniapp/src` 未找到通用 `Skeleton`、`PageState` 或网络状态监听接入；06 明确要求修复的个人中心跳页问题仍在，新评价页又出现同类问题。
- 保留并完善已有原生分享，不要为海报另造商品落地参数。

## 本轮验证

已通过：

- 后端 `go test ./...`。
- 管理端 `pnpm run test`，29/29；`pnpm run build`。
- 小程序 `pnpm run test`、`pnpm run type-check`、`pnpm run build:h5`、`pnpm run build:mp-weixin`。

额外验证在 `/tmp` 中完成，没有写入业务测试目录：

- Go overlay 调用真实评价服务，以 SQLite 运行四个“正确行为应满足”的断言，四项均失败：详情字段隔离、restore 状态限制、图片可见性隔离、纯文本往返，确认上述缺陷。
- 编译并执行真实 Vue SFC 的 setup，仅替换网络和页面宿主，复现评价跳页、我的评价旧响应覆盖、搜索清空后旧响应写回。
- 现有 `miniapp/test/review-flow.test.mjs` 多处在测试文件中重新定义模拟逻辑，没有导入真实评价页面；其通过不能证明页面异步行为正确。

限制：没有配置 `TEST_DATABASE_DSN` / `ORDER_TEST_MYSQL_DSN`，未验证真实 MySQL 迁移、锁竞争与长度约束；未做浏览器视觉验收、微信开发者工具/真机、对象存储 ACL 和微信商品码外部联调。构建存在既有 Sass 弃用和大包提示。

## 交给 AI 的提示词 A：先修评价后端

```text
请在 /home/shop/mall 修复 docs/next_task/07-review-findings-and-fix-prompts.md 的问题 1–5。
先读 AGENTS.md、admin-server Skill、docs/next_task/README.md 和 03a-review-server.md，再复查最新源码。直接实现修复，保留已有改动。

1. 分离作者评价详情与公开评价详情的响应字段。其他会员查看公开评价不能拿到订单 ID、订单明细 ID、审核原因或其他私有字段；作者和有权限管理员仍可取得必要数据。不要只在前端隐藏。同步 DTO、Swagger 和调用方。
2. 解决多个评价共用 file_id 的审核可见性冲突。优先采用符合当前项目规模的简单归属方案，若限制文件只能绑定一条评价，必须有事务/数据库并发约束，并处理已有重复引用；不要仅先查再插。需要结构变化时新增下一编号 Migration，禁止改已存在迁移、清库或 align。验证本地和配置中的对象存储实际访问控制。
3. 服务端明确状态转换：10 只能 approve/reject，20 只能 hide，40 只能 restore，30 不提供重新审核入口。保留 expected_status 条件更新，非法动作与并发冲突不得写日志或改变图片可见性。
4. 评价创建沿用售后流程“先锁订单”的锁顺序，事务内重新检查归属、状态和售后资格。处理 MySQL 一致性读与行锁时序，不能只补一条注释。将幂等载荷比较集中复用，比较明细、评分、正文、匿名标记和有序图片 ID；事务前重试、锁后重查和唯一键冲突恢复采用同一规则。同载荷返回同一评价，不同载荷返回冲突。
5. 正文按纯文本保存与返回，由 Vue/React 安全文本渲染，不使用 v-html/dangerouslySetInnerHTML。兼容本次旧代码已保存的转义数据，说明迁移策略，避免盲目重复 decode 损坏用户本来输入的实体文字。Unicode 长度、存储长度和幂等比较保持一致。

补真实服务/API 回归：其他会员读取匿名/非匿名公开详情、非公开拒绝、状态转换矩阵、重复图片引用、隐藏与恢复访问、相同/不同图片的幂等请求、评价与售后并发、特殊字符和 500 字边界。
按 README 生成 Swagger；影响管理端契约时生成 Client，禁止手改 generated_api.ts。运行相关 Go/前端测试与构建，真实 MySQL 只使用隔离测试库；没有 DSN 时记录未验收，不用业务库。
不要为了测试通过删除权限校验或降低断言。提交简短结果：修复项、验证证据、剩余外部检查。
```

## 交给 AI 的提示词 B：修复小程序异步状态并完成 06

```text
请在 /home/shop/mall 修复 docs/next_task/07-review-findings-and-fix-prompts.md 的问题 6–9，并完成原 06-skeleton-network.md。
先读 AGENTS.md、miniapp Skill、next_task README、01、02、03b、04、06。保留已有页面和公共 Store，前端尽量不用 try/catch，复用公共错误处理，同时确保 reject 被处理。

1. 修复商品评价、我的评价、个人中心推荐和订单分页：nextPage 只作请求局部值，成功后提交页码；失败重试同页；按稳定 ID 去重；刷新、换筛选和分页共享查询版本；首次错误、真实空态、分页错误分别展示。有数据刷新失败保留可见内容与重试入口。
2. 私有查询和写操作捕获会话与页面生命周期，响应写回前检查；账号变化清理我的评价、详情、评价表单及地址表单。微信地址草稿填入表单后仍绑定原会话，不能把 A 的草稿用 B 的 token 保存。原生选择器、上传、保存的晚响应不得回填、toast 成功或导航到已离开的页面。仅 request 层处理旧 401 不够。
3. 搜索清空、提交空词、换查询、卸载应使旧请求失效；先判断是否允许加载再生成该次请求序号，避免无效调用使有效请求失效。失败保留旧结果时清晰标记其生效查询。
4. 订单页 onHide 暂停 ticker 订阅和状态重查，onShow 刷新服务器状态并重新校准；继续共用一个 ticker。失效时间字段先刷新，刷新失败禁用付款并显示重试。不要使用主动取消接口模拟超时关单，不要影响后端幂等付款。
5. 按 06 补轻量 Skeleton/PageState、可见页面网络提示及合并只读刷新，覆盖原文要求页面，也接入已实现的搜索/评价。网络恢复禁止自动重放写请求；结算预览失效时禁用提交。公共组件提供使用说明与实际接入示例。

将 review-flow.test.mjs 中复制业务逻辑的关键用例改为真实 SFC/composable/Store 源码测试，可沿用仓库 cart-drawer-lifecycle.test.mjs 的编译和宿主替换方式。
至少验证：第 2 页失败后的请求序列 [1,2,2]、旧刷新/分页晚返回、A 账号响应晚于 B、A 地址草稿不能提交给 B、清空搜索后旧响应忽略、onHide 无重试/计时器残留、恢复网络不自动提交/支付。
运行 miniapp test、type-check、H5/微信双端构建及 git diff --check，明确未完成的浏览器/微信真机检查。不要用静态测试自造状态机代替页面行为验证。
```

## 交给 AI 的提示词 C：补齐商品海报

```text
请在 /home/shop/mall 完整实施 docs/next_task/05b-product-poster.md。
先读 next_task README、05a、05b、07 的问题 10，以及 miniapp/admin-server Skill。当前只有原生分享与 id/scene 解析，海报尚未实现。
复用现有 product-share.ts 和商品详情入口，按 05b 完成服务端微信 token 管理、真实商品码生成与缓存/存储、登录鉴权和 Swagger，以及可复用 ProductPoster 的 Canvas 绘制、预览、保存、异常恢复和生命周期保护。
执行时核对微信/DCloud 官方契约；缺 AppID/AppSecret、合法域名或已发布页面时，完成可验证代码与 Mock 边界测试，明确外部验收缺口，不返回假二维码或宣称已联调。
新增结构使用下一编号增量 Migration；不要改历史迁移、调用 align 或把 Secret 返回前端。金额使用现有精确格式化，扫码落地复用 p=<id>。
为真实服务/组件补必要回归，执行 README 对应测试与构建，补组件说明及实际接入示例，并按“已完成/未验证”更新交付记录。
```

建议按 A → B → C 执行；每批完成后重新对照原任务验收，不需要重写已通过的业务流程。
