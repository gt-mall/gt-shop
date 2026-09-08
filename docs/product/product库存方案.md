# 商品库存方案（第一版）

本方案在已实现的 `product_category`、`product`、`product_sku` 基础上增加库存。此前 `product提示词.md` 中“库存本次不做”只约束商品第一阶段；执行本目录的新库存提示词时，按本方案扩展，不重做商品模块。

交付文件：

- [建表与存量回填 SQL](product库存.sql)
- [后端 AI 执行提示词](product库存后端提示词.md)：先执行，产出 Swagger 和生成客户端。
- [前端 AI 执行提示词](product库存前端提示词.md)：后执行，直接使用生成接口。

此文档任务只产出方案、SQL 和提示词，不执行业务库迁移，也不实现库存业务代码。

## 1. 本期范围

采用**单仓库、按 SKU 管库存、整数件数**的方案。支持库存列表、人工入库、人工出库、盘点校正、预警阈值设置和只读库存流水。

不做多仓库、批次/保质期、采购单、调拨单、订单、支付、自动预占/释放、退款回库、库存成本核算和批量调整。一次写请求只操作一个 SKU；将来批量操作需另定整批事务与逐项幂等规则。

`locked_stock` 为后续订单预留，本期只在初始化时设为 0，不提供人工编辑或占用接口。人工调整必须保留原有锁定量，不能把已有非零值重置为 0。两张表能支持当前管理功能，不能代替未来“哪个订单占用了多少库存”的订单明细或占用记录。

## 2. 两张表与数量口径

| 表 | 作用 | 生命周期 |
| --- | --- | --- |
| `product_sku_inventory` | 每个 SKU 一条当前余额及预警配置 | SKU 恢复时复用，禁止业务删除 |
| `product_sku_inventory_log` | 每次成功库存/预警变更的前后值、原因和操作者 | 只追加，禁止修改或删除 |

余额字段：

| 字段 | 含义与约束 |
| --- | --- |
| `product_id` | 为商品过滤和删除检查保留的冗余关联，由服务端从 SKU 派生 |
| `sku_id` | 唯一，一个 SKU 一条余额；默认 SKU 也必须有库存 |
| `stock` | 账面总库存，包含已锁定但尚未实际扣减的部分 |
| `locked_stock` | 已被业务占用的数量，必须在 0 到 stock 之间 |
| `available_stock` | 接口计算字段：stock - locked_stock，**不额外存一份** |
| `warning_stock` | 可用库存预警阈值，0 表示关闭预警 |
| `version` | 每次库存或阈值成功变更递增 1，用于识别过期页面 |

例如：总库存 100、锁定 20，可用库存为 80。不能将“可用库存”和“总库存”混叫成库存后混用。

数量使用有符号 `INT` 存储，业务范围 `0..2147483647`；流水 delta 可以为负。Go 使用 `int64` 做边界检查和加减，写入前验证 INT 范围。件数不是金额，不使用 DECIMAL；已有商品价格仍保持 DECIMAL 元及精确计算。

版本使用 `INT UNSIGNED`，范围 `0..4294967295`，API 使用 number 不损失精度。达到上限时拒绝变更，不能回绕。ID 使用 BIGINT / Go int64；与当前商品 API 的 ID 风格保持兼容，不擅自修改已有 INT 主键。现有生成客户端将 int64 映射为 number；若未来 ID 可能超过 `Number.MAX_SAFE_INTEGER`，需要统一调整项目 ID 契约，本期不要给新库存 ID 引入雪花算法。

SQL 不增加外键，也不扩大改动到已建的三张商品表；已有商品 ID 的实际 INT/BIGINT、有符号属性先只读核对。关联完整性由事务、初始化和巡检保证。余额与流水都不设 `deleted_at`，避免商品软删除时库存凭空消失。

## 3. 操作定义

本期四种变更共用流水；`quantity` 的含义随操作明确区分。

| change_type | 页面文案 | quantity | stock_delta | 其他变更 |
| --- | --- | --- | --- | --- |
| `manual_in` | 人工入库 | 本次增加件数，必须 > 0 | +quantity | 锁定及阈值不变 |
| `manual_out` | 人工出库 | 本次减少件数，必须 > 0 | -quantity | quantity 不得超过可用库存 |
| `stocktake` | 盘点校正 | 盘点后的**目标总库存**，允许 0 | quantity - 原 stock | 目标不能小于锁定库存 |
| `warning_set` | 设置预警 | 目标阈值，允许 0 | 0 | 更新 warning_stock，锁定不变 |

所有写操作必须提交 1..500 字符的原因 `remark`；总库存、锁定量、阈值、版本和日志原子保存。本期不允许客户端写 `locked_stock`。

盘点目标等于当前总库存、阈值等于当前阈值时，返回 `NO_CHANGE`，不增加版本、不写流水；不要伪造一条库存变化。阈值确有变化时，即使 stock_delta=0，也必须有 `warning_set` 流水。

缺货：`available_stock == 0`。低库存：`warning_stock > 0 && available_stock <= warning_stock`。两者可同时成立；禁用 SKU 也能查看和管理库存，上下架不增减库存，不因库存为 0 自动改商品状态。

未来接订单时再实现以下语义，本期只保留字段，不写占位服务：

| 未来业务 | stock_delta | locked_delta | 必要约束 |
| --- | --- | --- | --- |
| 订单预占 q | 0 | +q | 可用库存至少 q |
| 取消释放 q | 0 | -q | 必须释放该订单实际占用的数量 |
| 占用转实际出库 q | -q | -q | 对应订单仍有足够占用，不能重复扣 |
| 退货验收入库 q | +q | 0 | 有对应退货业务事实与幂等业务标识 |

未来要明确选择业务上的实际扣减节点，例如发货；不能付款扣一次、发货再扣一次。订单取消/发货还需要订单级状态和数量约束，仅检查 SKU 总 locked_stock 不够。

## 4. 流水与快照

日志保存 stock、locked_stock、warning_stock 的前后值、stock_delta、locked_delta，以及 version_before/after。满足：

```text
stock_after = stock_before + stock_delta
locked_after = locked_before + locked_delta
available_before = stock_before - locked_before
available_after = stock_after - locked_after
version_after = version_before + 1
```

`request_id` 全表唯一；`(inventory_id, version_after)` 也唯一，一次版本变化对应一条流水。后端限制 change_type、quantity 与 delta 的对应关系；数据库 CHECK 负责非负、锁定上限和前后值等基础约束。

`sku_snapshot` 由服务端读取当时商品和 SKU 生成，不能信任前端提交，API 返回对象：

```json
{
  "product_name": "基础款T恤",
  "sku_code": "TSHIRT-BLACK-L",
  "unit": "件",
  "specs": [
    {"spec_id": 1, "spec_name": "颜色", "value_id": 1, "value_name": "黑色"},
    {"spec_id": 2, "spec_name": "尺码", "value_id": 1, "value_name": "L"}
  ]
}
```

规格选项必须用 `(spec_id,value_id)` 联合定位。默认 SKU 的 specs=[]，显示“默认规格”；sku_code、unit 未填写返回 null。商品或规格改名后，旧流水仍展示快照；商品删除、操作者改名或删除也不影响历史展示。

操作者从 `auth.CurrentUser(c)` 获取，operator_name 使用昵称，没有昵称时使用用户名。前端不能传 operator_id。日志不返回内部 request_hash。

## 5. 事务、幂等与并发

### 锁顺序

写操作统一锁顺序：**product → product_sku → product_sku_inventory**。全部使用同一个 `tx`，相关锁持有至事务结束；同一商品的不同 SKU 暂时串行，适合当前管理端规模。不要混用其他路径“先锁库存再锁商品”的顺序。

调整入口可先用 SKU 找到 product_id 作为定位信息，但必须在事务锁住商品后重新锁定/核对 SKU 的存在、归属和软删除状态。商品更新、商品删除、未来 SKU 写接口也遵守上述顺序，多 SKU 按 ID 升序加锁。

### 一次写入的步骤

1. 校验登录、该接口权限、请求结构；归一化 UUID 小写和 remark 首尾空白，使用固定 DTO 字段计算 request_hash，包含 operator_id、sku_id、change_type、quantity、expected_version、remark。不能对无序 map、前端 hash 或原始 JSON 文本做业务等价判断。
2. 可先查已提交的 request_id；存在时校验操作者和 hash，相同直接返回原流水并标记 replayed=true；不同返回 IDEMPOTENCY_CONFLICT，不能泄露别人的流水。
3. 未命中则开启事务，按顺序锁定商品、SKU、库存，并核对库存归属。旧幂等回放允许发生在商品软删除以后；新的变更必须针对未删除商品和 SKU，商品下架/SKU 禁用不阻止人工调整。
4. 获锁后再次对 request_id 做**当前读**检查，确保并发重试能看到已经提交的流水；不要复用事务开始时的旧快照。幂等命中要在 expected_version 判断之前处理。
5. 对新请求校验 expected_version 与当前版本一致，校验可用量、溢出、目标值和 NO_CHANGE。
6. 计算前后值，更新余额及 version；建议 UPDATE 附带旧 version 条件，并检查 RowsAffected=1。随后写入完整流水；流水失败、唯一冲突或任一步出错，余额一起回滚。
7. 提交成功后返回该次流水；网络或 COMMIT 结果不确定时，不能告诉调用方“肯定失败”，允许原 request_id 查询式重试。

请求唯一键竞争时，回滚当前事务后，使用新查询读取已提交日志并校验 hash；不要在旧快照中反复找，也不要见到任意重复键就返回成功。请求唯一键、库存唯一键、版本唯一键需要分别处理。

死锁/锁等待失败最多做少量有界重试或返回 RETRYABLE，重试整个事务且保持原请求不变；不重试任意业务错误，不靠进程 mutex、前端按钮禁用或 Redis 缓存保证正确性。

两个操作基于同一 expected_version 并发到达，只能一个成功；另一个返回 VERSION_CONFLICT。页面刷新并让用户确认新余额后，才以新 request_id 发起新的业务操作。

失败请求未写日志，不占用幂等键；幂等承诺覆盖已提交的成功操作。超时未必失败，必须使用原 request_id 和原 payload 重试，不允许换 UUID 重做。

## 6. 与现有商品生命周期衔接

- 新商品默认 SKU、新商品多 SKU、商品编辑新增 SKU：创建 SKU 后在**商品原事务**中初始化唯一的零库存，不能提交商品后再补建。初始化失败则整单回滚。
- 普通商品编辑不接收 stock、locked_stock、warning_stock、version，也不保存库存。商品 DTO 即使出现未知库存键也不能由模型绑定到数据库；不要在商品页增加会被聚合提交覆盖的库存输入框。
- SKU 同组合恢复仍复用原 SKU ID 和已有余额；不能重置库存和预警。首次上线前通过回填保证旧 SKU 均有记录；运行中发现旧 SKU 缺库存应报错并修复迁移，不临时填 0 掩盖问题。
- 全部存量 SKU（包括禁用/软删除）先补零余额，不写“0 到 0”的库存流水。真实期初量走人工入库并留原因，不从 sales_count、virtual_sales 或 cost_price 推算。
- 商品删除先锁商品、全部 SKU 及其库存，包含历史软删除 SKU；任一库存缺失、归属异常或 stock/locked_stock 不为 0 时拒绝。库存归零后才允许商品与 SKU 软删除，余额和流水始终保留。不要在删除时自动清零。
- 库存服务只依赖模型和已有公共能力。当前 product/product.Service 可单向调用 inventory 的“使用调用方 tx 初始化”方法，inventory 不反向调用商品 service；无需为这次扩展重构整个商品模块。

## 7. API 与权限交接契约

统一 POST JSON，沿用 `Response{code,message,total,data}`；code=1 成功。操作方法名在 Swagger 的 `@ID` 固定，前端以生成结果为准。

| 路径 | @ID | 权限 | 功能 |
| --- | --- | --- | --- |
| `/api/product/inventories/list` | `listProductInventories` | `product:inventory:list` | 按 SKU 分页 |
| `/api/product/inventories/detail` | `getProductInventory` | `product:inventory:list` | 打开调整框前取最新余额，{sku_id} |
| `/api/product/inventories/adjust` | `adjustProductInventory` | `product:inventory:adjust` | manual_in/manual_out/stocktake |
| `/api/product/inventories/warning` | `setProductInventoryWarning` | `product:inventory:warning` | warning_set |
| `/api/product/inventories/logs/list` | `listProductInventoryLogs` | `product:inventory:log` | 只读流水 |

adjust/warning 路由同时要求 inventory:list；新增库存权限使用现有 Casbin 权限定义和角色授权，不建权限表。log 可独立授权，流水页不依赖库存列表/商品详情权限。库存权限不自动包含商品编辑、商品详情、图片素材写权限。

### 列表与详情

列表请求：page（默认1）、page_size（默认20，最大100）、keyword、product_id、sku_id、product_status、sku_status、stock_filter。stock_filter 允许 `all/low/empty/normal/uninitialized`；normal 表示有可用量且不低库存。状态 0、sku_id 等零值校验要区分“未传”。keyword 搜索当前商品名称、SKU 编码和条码。

以未删除 SKU + 未删除商品为基础 LEFT JOIN 库存；默认包含下架商品和禁用 SKU。缺库存的 SKU 仍显示且 initialized=false，数量和 version 返回 null，不能 inner join 隐藏或 COALESCE 成正常零库存。uninitialized 筛选这些异常；其他数量筛选只匹配已初始化记录。

`ProductInventoryRow` 至少包含：inventory_id、product_id、sku_id、product_name、sku_code、barcode、unit、specs（含规格名称的对象数组）、image_url、product_status、sku_status、initialized、stock、locked_stock、available_stock、warning_stock、version、is_low_stock、is_out_of_stock、updated_at。缺库存时 inventory_id、数量、version、预警/缺货标记、updated_at 为 null；有库存时是确定值。Swagger 标记 nullable，API 的 specs 始终数组。

沿用当前项目已验证的 `extensions:"x-nullable"` 声明可空字段，并检查生成结果确实包含 `| null`。详情与列表不要分别发明两套字段名。带名称的 specs 元素复用第4节快照中的规格结构。

列表稳定按 SKU id DESC；分页总数、筛选和返回数据一致，不连接流水造成重复 SKU。图片批量解析并复用历史图片读取能力，SKU 图缺失回退商品主图，不强制用户具有图库管理权限。规格按对应商品的定义解析，避免 N+1 查询。

详情请求 {sku_id}，成功返回一行最新 ProductInventoryRow；目标删除/不存在或未初始化时返回 code=0、data=null 和明确中文原因（如“库存未初始化”），不能进行隐式写入。只读接口沿用现有错误包裹；下面的 data.error_code 约定用于两个写接口。

### 写请求与响应

库存调整：

```json
{
  "sku_id": 101,
  "request_id": "ee6e49d6-c8b3-4c24-a193-682d14cf6cdf",
  "expected_version": 0,
  "change_type": "manual_in",
  "quantity": 100,
  "remark": "首批到货入库"
}
```

预警设置：

```json
{
  "sku_id": 101,
  "request_id": "c89d4210-64f6-40fc-a1e1-cbb6d6c91859",
  "expected_version": 1,
  "quantity": 10,
  "remark": "设置日常补货提醒"
}
```

warning 接口由服务端固定 change_type=warning_set；quantity 在界面标为“预警阈值”。expected_version 和 quantity 必填且允许 0，Go DTO 使用指针或存在性校验，不能简单给非指针数值写 binding:required 而误拒绝合法 0。

写接口共用 `ProductInventoryMutationResult`：可选 error_code、可选 log（ProductInventoryLogRow）、replayed。成功时 code=1、log 非空、error_code 为空；幂等回放返回原流水的前后值，replayed=true，不伪装成最新余额。页面成功后重新拉列表/详情。

例如，第一次以 version=0 从 0 入库 100 件，流水为 stock_before=0、stock_delta=100、stock_after=100、version_before=0、version_after=1。随后将预警设为 10，产生 warning_set 流水，stock 前后均为 100、warning_before=0、warning_after=10、版本 1→2。此后重放最初入库请求仍返回最初的版本 0→1 流水，余额维持 100、当前版本维持 2。

业务错误 code=0，data.error_code 使用 `INVALID_ARGUMENT/NOT_FOUND/NOT_INITIALIZED/VERSION_CONFLICT/INSUFFICIENT_STOCK/NO_CHANGE/IDEMPOTENCY_CONFLICT/RETRYABLE/RESULT_UNKNOWN`，message 使用可读中文；不改公共 Response。绑定失败也只写一次 JSON。公共 HTTP 产生的网络错误可能没有 data.error_code，前端按结果不确定处理。

### 流水查询

请求：page、page_size、product_id、sku_id、request_id、change_type、operator_id、created_from、created_to。时间接收带时区的 RFC3339，范围为 `[created_from,created_to)`，from/to 同时存在时校验先后。ID、类型、时间都可单独过滤，未传不限制。

排序 created_at DESC、id DESC，返回全部日志字段（除 request_hash）及可计算的 available_before/after；sku_snapshot 返回明确 DTO，不能是转义 JSON 字符串。分页查日志本身，不用“只查未删除商品”的关联过滤掉历史，也不要求商品仍然存在。

## 8. 页面方案

商品管理大模块下增加“库存管理”和“库存流水”：

- `/product/inventory`：独立 SKU 库存列表，行内“入库”“出库”“盘点”“设置预警”“流水”；操作前用 detail 拉最新数据，再打开 Modal + Form。
- `/product/inventory/log`：全局只读流水；支持 `?sku_id=101&product_id=10` 深链接和 F5 保留筛选，当前商品删除后仍能查看。
- 商品列表/详情只加有权限才显示的“查看库存”入口，跳到 `?product_id=...`，不把库存混进商品聚合保存。

没有对应操作权限就不显示按钮；有库存权限但无商品权限时，库存模块仍独立可用。流水权限单独控制入口与页面。库存缺失显示“未初始化”，禁用调整，不能显示成 0。

提交时冻结本次请求的 UUID、版本、数量、原因；按钮 loading 防连点。超时/断网/结果不确定时，保留原请求，显示“确认上次操作结果/重试”，不能生成新 UUID。可在 sessionStorage 按用户保存一个待确认请求，刷新后恢复，完成或明确业务拒绝后清除；禁止把 Token 存入该记录，也不能由另一账号重放。

用户在明确版本冲突后刷新余额，检查数量和原因，再提交新的请求。尤其盘点使用的是目标总库存，不能在刷新版本后自动覆盖别人的新变化。

## 9. 验证与发布

实施 AI 必须验证：

1. 唯一 SKU 库存、零初始化、存量回填重跑不覆盖余额、SKU 恢复保留原余额。
2. 入库/出库/盘点/阈值变更的前后值、负数/小数/溢出、合法 0、NO_CHANGE。
3. 同请求串行/并发重放只有一次变更、一条日志；同 UUID 换数量、SKU、原因或操作者会冲突。
4. 旧 expected_version 拒绝；并发超扣不能发生；模拟写日志失败时余额及版本回滚。
5. COMMIT 后响应丢失，再用原 payload 重试返回原流水，不重复增减。
6. 单连接池下事务内图片/操作者/幂等查询不额外借连接；MySQL 当前读、行锁、唯一键冲突和 CHECK 在隔离环境验证。
7. 编辑商品不改库存；正库存/锁定库存/缺库存阻止商品删除；禁用或下架不改余额，历史流水仍能读。
8. 菜单、路由、按钮、接口权限一致；只有 inventory:list 的账号可看不可改，只有 inventory:log 的账号可看历史。
9. 前端数量单位、盘点目标、阈值关闭、重复选项 ID、深链接、网络失败后同请求重试和页面刷新恢复。

DDL 与回填不等同于应用事务，建表后回填失败要定位并重跑 B 段，不能声称整份 SQL 自动回滚。上线顺序：核对表结构 → 暂停商品写入 → 建表/回填 → 部署带初始化与删除保护的后端 → 验证无缺失库存 → 恢复写入 → 发布前端。

SQL 以 MySQL 8.4 为目标。数据库 CHECK 用于余额及流水基本约束，服务端仍校验业务规则；语法与执行方式参见 [MySQL CHECK Constraints](https://dev.mysql.com/doc/refman/8.4/en/create-table-check-constraints.html)。并发校验采用事务内锁定读，参见 [MySQL Locking Reads](https://dev.mysql.com/doc/refman/8.4/en/innodb-locking-reads.html)。其他数据库版本必须另行验证，不能默认按本版本特性运行。

## 10. 本次文档与 SQL 验证记录

2026-09-08：在独立 MySQL 8.4 容器、临时数据库及模拟商品/SKU 数据上执行本 SQL，16 项检查通过，包括：

- 两张表建表及首次零库存回填；禁用/软删除 SKU 同样获得余额。
- 重跑回填保留已有数量及版本，只补新出现的 SKU。
- 拒绝负库存、锁定量大于总量、负阈值、INT 数量溢出及同 SKU 重复余额。
- JSON 快照正常保存，request_id 唯一，同一库存版本只能有一条日志。
- 拒绝错误的 before/delta/after 关系和不连续版本；日志插入失败时事务可回滚余额。

文档链接、代码块、JSON 示例、两份提示词的 API 方法名及空白检查通过。本次未执行业务库迁移，未编写库存接口或页面；实际服务的并发锁、幂等回放、权限、商品生命周期和浏览器交互，仍按上述提示词在实现阶段验证。SQL 实测不代表这些应用功能已实现。
