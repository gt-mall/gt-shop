# 第二轮复查与修复提示词

日期：2026-09-20。检查对象：`c975b26` 之后的当前工作区改动（本轮开始时尚未提交），含新增 share 服务、海报组件、004/005 迁移和网络组件。本轮没有修改业务代码。

## 上轮问题修复状态

| 上轮项目 | 当前结论 |
| --- | --- |
| 评价详情泄露订单/审核字段 | 已修复：非作者响应省略私有字段，已有真实服务回归 |
| 评价图片重复绑定 | 新创建路径已限制并加唯一键；历史迁移仍有遗漏，见 6 |
| 非法审核状态转换 | 已修复：服务端增加转换矩阵，回归通过 |
| 评价与售后并发、幂等 | 部分修复：已加订单锁及图片比较，但读快照时序和文本比较仍有问题，见 2、5 |
| 正文重复转义 | 新评价纯文本往返已修复；兼容分支产生新的幂等问题 |
| 分页失败跳页 | 常规失败重试已修复；翻页期间切筛选/账号出现新问题，见 3 |
| 会话隔离 | 多处补了会员检查；我的评价切号期间可能卡死，见 3 |
| 隐藏页倒计时/重查 | 已暂停，但返回页面后未恢复中断的重查，见 4 |
| 搜索清空旧响应回写 | 清空时已使旧请求序号失效 |
| 海报、骨架/弱网 | 已新增实现；海报仍有问题，06 的页面覆盖尚不完整 |

## 仍需修复的问题

### 1. P1 / 新增：微信请求失败时把 AppSecret/access_token 返回前端

- 位置：`admin-server/internal/miniapp/share/adapter.go:62`、`:119`；`share/api.go` 的 `response.Error(c, err.Error())`。
- `http.Client.Do` 的错误包含请求 URL。token URL 中有 `secret`，商品码 URL 中有 `access_token`，当前 `%w` 包装后沿 service 原样返回 API。
- 已用虚构凭证和模拟上游网络故障调用真实 API，确认 JSON 响应包含配置中的虚构 AppSecret；另用取消上下文确认商品码请求错误也包含 token。没有访问真实微信或输出真实凭证。
- 修复：边界剥离 URL 与凭证，向前端只返回安全错误类别和提示；日志也必须脱敏。不能只处理微信 JSON 业务错误。

### 2. P2 / 未修完整：订单锁之前已建立旧快照

- 位置：`admin-server/internal/miniapp/review/service.go:180`，普通读取明细在 `183`，订单锁在 `192`，售后普通查询在 `254`。
- 在 MySQL `REPEATABLE READ` 下，事务内首个普通 SELECT 建立一致性读快照。若售后事务已持有订单锁，评价先读明细，再等待订单锁；售后提交后评价获得锁，但随后的普通售后查询仍可能使用之前的快照，漏掉刚提交的售后。
- 这是根据当前代码和 [MySQL 官方一致性读说明](https://dev.mysql.com/doc/refman/8.4/en/innodb-consistent-read.html) 推导的条件性缺陷；本轮没有真实 MySQL 复现，不应声称已经验证生产发生。
- 修复：使订单锁先于本事务第一次快照读取，或采用明确的当前读方案；保留订单优先锁顺序。不要通过全局修改隔离级别掩盖局部问题。

### 3. P2 / 新增：分页锁阻断切换查询，甚至让我的评价永久卡住

- 位置：`miniapp/src/pages/review/product.vue:170`、`:178`；`review/mine.vue:137`、`:166`。
- 商品评价第 2 页请求中点“差评”，标签会变化，但 `loadingMore` 使 `fetchData(true)` 提前返回，既不发新请求也不使旧请求失效。真实 SFC setup 复现：标签为 `bad`，两次请求仍是 `all`，旧的 5 星评价被追加到“差评”列表。
- 我的评价第 2 页请求中切换账号，watch 递增序号后调用刷新，但同样被 `loadingMore` 拦截；旧请求 finally 因序号失效不能清除 loadingMore，后续刷新始终被拦截。实际复现 `loadingMore=true`、列表空、新账号未发请求。
- 修复：新查询重置应能取消旧查询归属并重置加载标志；分页防重只阻止同一查询的重复追加，不能挡住切号/切筛选。

### 4. P2 / 新增：超时确认中隐藏再显示，停在“确认中”

- 位置：`miniapp/src/pages/order/detail.vue:271`、`:279`、`:310`；列表的卡片确认也应同步检查。
- 到期触发重查后，onHide 清理 timer，但保留 `isConfirming=true`；onShow 只请求一次详情。若 worker 尚未关单、服务器仍返回状态 10，既不会重启重试，又因 confirming 已为 true 拒绝新 timeout 事件。
- 真实 SFC setup 复现：返回后 `confirming=true, timers=0, failed=false`，没有手动失败重试入口。
- 修复：恢复可见时重新确认并启动有限重查，或进入明确可重试状态；同时处理旧网络响应和列表原卡片对象被刷新替换的情况。

### 5. P2 / 新增：历史转义兼容把不同正文当作同一幂等载荷

- 位置：`admin-server/internal/miniapp/review/service.go:96`。
- 已复现：先以正文 `A&amp;B` 创建新评价，再用同一 request_id 提交 `A&B`，接口返回成功。新记录本来就是纯文本，不能不加区分地 `html.UnescapeString` 后比较。
- 修复：严格比较新记录；历史兼容依据明确迁移/版本标记，不要按字符串含 `&` 猜测。回归覆盖字面实体、真正特殊字符、图片顺序、同载荷重试和不同载荷冲突。

### 6. P2 / 未修完整：004 清理旧图片绑定后未重建统计和访问状态

- 位置：`admin-server/data/migrations/004_review_image_unique.sql:7`。
- 脚本仅保留最早绑定，删除其余绑定，未重算 `product_review.has_image`，也未按剩余评价状态校正 `system_file.visibility`。
- 有图评价可能已经没有图片却仍计入“有图”统计。若最早绑定属于隐藏评价，而文件因后来的公开评价变为 public，删除后者绑定后文件仍保持 public。
- 此项为迁移源码确认，未运行真实 MySQL 迁移。应处理已有重复引用的业务归属并修正派生字段；迁移是否已应用必须先查清，已应用迁移只能新增补偿迁移，禁止直接改写历史。

### 7. P2 / 新增：商品码生成没有并发合并和失败补偿

- 位置：`admin-server/internal/miniapp/share/service.go:105`、`:117`、`:140`。
- 先检查缓存、调用微信、上传文件，最后才 upsert；唯一键只防映射行重复，不能防重复生成。SQLite + Mock 真实服务并发测试确认，同商品两次同时请求调用了两次微信商品码 API。
- 映射写入失败没有清理刚上传文件。错误恢复分支还把 `existing.FileID` 与新上传的 `fileResp.URL` 拼成一份响应，文件 ID 与 URL 可能不对应。
- 修复：按完整商品码身份合并生成；失败清理自己生成且未被引用的资源；竞争结束后解析最终胜出的 file_id 对应 URL；补并发、保存失败和补偿测试。

### 8. P2 / 新增：token 缓存忽略凭证变更，Redis TTL 与真实期限脱节

- 位置：`admin-server/internal/miniapp/share/token.go:41`。
- key 和内存索引只用 AppID，AppSecret 变化不会使旧缓存失效。已复现：相同 AppID 换 secret 后，再次 GetToken 仍只调用一次微信客户端。
- Redis 固定缓存 6900 秒，内存才保存实际 expires_in；Redis 命中会跳过有效期检查。短有效期 token 或从内存回填剩余寿命较短的 token，会被缓存过久。
- 修复：按凭证版本失效，缓存明确 expiresAt，并以剩余有效期设置 TTL；不能把 secret 明文放在缓存键或日志中。

### 9. P2 / 新增：海报重新打开仍展示旧价格

- 位置：`miniapp/src/components/ProductPoster.vue:121`、`:237`。
- 有父页面商品数据就不刷新详情；已有 posterImagePath 就不重新绘图。实际 SFC setup 复现：首次绘制 10.00，关闭后把传入价格改为 30.00，再打开仍只绘制过 10.00，商品查询调用次数为 0。
- 修复：每次明确生成/打开应按最新商品信息验证或使海报缓存随商品内容失效；只长期复用商品码。补真实 ProductPoster 源码测试，目前 product-poster.test.mjs 仍重写模拟流程，未测试组件本身。

### 10. 任务缺口：06 只接入了部分页面

- Skeleton/PageState 主要用于评价页；首页、分类、商品详情、购物车/抽屉、收货地址没有完成原文要求的骨架与弱网接入，搜索/订单/个人中心多为新增 NetworkBanner。
- 多个页面传入 useNetwork 的刷新回调使用 `void loadXXX()`，导致 composable 无法等待请求完成，同一时刻只允许一个刷新并未成立；应返回 Promise 或在共享查询入口合并。
- 所以原 06 仍不能标记为完整完成；不能把“新增公共组件文件”等同于所有目标页面验收。

## 验证结果

- 已通过：`go test ./...`；miniapp `test`、`type-check`、`build:h5`、`build:mp-weixin`；`git diff --check`。
- 额外临时 Go overlay：5 个安全/正确性断言均复现缺陷，包括真实 API 错误凭证泄露；全程使用虚构凭证、SQLite 和 Mock，不接业务库。
- 额外执行真实 Vue SFC setup：复现筛选错乱、切号卡住、超时确认无法恢复、海报旧价格。生命周期/网络/Canvas 宿主为替身，不等同于浏览器或真机验收。
- 未运行：真实 MySQL 锁竞争/迁移、微信真机及对象存储 ACL、真实微信商品码联调。未设置专用测试 DSN。本轮管理端源文件未变更，未重复上轮管理端构建。

## 给 AI 的下一轮修复提示词

```text
请复查并修复 /home/shop/mall/docs/next_task/08-second-review-and-fix-prompt.md。
先读 AGENTS.md、相关端 Skill 和原 next_task README；保留当前工作区全部已有改动，不重做已通过的修复。

按顺序处理：
1. 先修 P1：微信 adapter 的 transport 错误不得将带 secret/access_token 的 URL 传给 API 或日志。用虚构凭证和模拟网络失败测试最终 JSON，而不是只测试客户端返回错误。
2. 修评价事务快照时序与文本幂等；在 MySQL REPEATABLE READ 下用 barrier 明确制造“先读明细、等待售后订单锁、售后先提交”的竞争，保证不能漏掉售后。使用专用可丢弃测试库，缺 DSN 时如实记录。
3. 修评价分页中切筛选/切账号，以及订单超时确认 hide/show 恢复。重置查询要能使旧请求和加载标志失效，原查询晚返回不影响新查询。恢复订单确认使用有限重查，不能无限“确认中”。
4. 修历史重复图片迁移后的 has_image 和 visibility。先确认 004 是否已应用；已应用只新增补偿迁移，不改历史，不清库、不 align。
5. 修商品码完整身份的并发合并、上传/映射失败补偿、最终 file_id 与 URL 一致；token 凭证变更失效，TTL 按真实剩余有效期。
6. 海报使用最新商品数据并在商品内容变化时重画；将 product-poster.test.mjs 的关键模拟测试改为真实组件 setup/源码测试，涵盖重新打开、快速切换、关闭与下载/绘图失败。
7. 按原 06 列出的目标页面完成骨架和弱网接入，网络恢复回调返回 Promise 并合并只读请求，禁止自动重放写操作。

前端尽量不用 try/catch，但保证 reject 被处理。接口变化先维护 Go DTO/注解、再生成 Swagger/Client。运行受影响端测试与构建及 git diff --check。报告每项“修复/未完成/外部未验证”，不要仅凭现有测试全绿宣称所有任务完成。
```
